package grpc

import (
	"context"

	"github.com/zarbchain/zarb-go/crypto"
	zarb "github.com/zarbchain/zarb-go/www/grpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (zs *zarbServer) GetBlockHeight(ctx context.Context, request *zarb.BlockHeightRequest) (*zarb.BlockHeightResponse, error) {
	h, err := crypto.HashFromString(request.GetHash())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Hash provided is not Valid")
	}
	height := zs.state.BlockHeight(h)
	if height == -1 {
		return nil, status.Errorf(codes.NotFound, "No block found with the Hash provided")
	}
	return &zarb.BlockHeightResponse{
		Height: int64(height),
	}, nil
}

func (zs *zarbServer) GetBlock(ctx context.Context, request *zarb.BlockRequest) (*zarb.BlockResponse, error) {
	height := request.GetHeight()
	block := zs.state.Block(int(height))
	if block == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Block not found")
	}
	hash := block.Hash().String()
	timestamp := timestamppb.New(block.Header().Time())
	info := &zarb.BlockInfo{}
	tranactions := make([]*zarb.TransactionInfo, 0)

	//populate BLOCK_DATA
	if request.Verbosity.Number() > 0 {

		SortitionSeed, err := block.Header().SortitionSeed().MarshalText()
		if err != nil {
			zs.logger.Error("couldn't marshal sortition seed: %v", err)
		}

		Committers := make([]int32, len(block.LastCertificate().Committers()))
		for c := range block.LastCertificate().Committers() {
			Committers = append(Committers, int32(c))
		}

		Absentees := make([]int32, len(block.LastCertificate().Absentees()))
		for c := range block.LastCertificate().Absentees() {
			Absentees = append(Absentees, int32(c))
		}

		info = &zarb.BlockInfo{
			Version:             int32(block.Header().Version()),
			LastBlockHash:       block.LastCertificate().BlockHash().String(),
			StateHash:           block.Header().StateHash().String(),
			TxIdsHash:           block.TxIDs().Hash().String(),
			LastCertificateHash: block.LastCertificate().Hash().String(),
			SortitionSeed:       SortitionSeed,
			ProposerAddress:     block.Header().ProposerAddress().String(),
			Round:               int64(block.LastCertificate().Round()),
			Committers:          Committers,
			Absentees:           Absentees,
			Signature:           block.LastCertificate().Signature().String(),
		}

	}

	//TODO: Cache for better performance
	//populate BLOCK_TRANSACTIONS
	if request.Verbosity.Number() > 1 {
		for _, id := range block.TxIDs().IDs() {
			t := zs.state.Transaction(id)
			tranactions = append(tranactions, zs.encodeTransaction(t))
		}
	}

	res := &zarb.BlockResponse{
		Hash:        hash,
		BlockTime:   timestamp,
		Info:        info,
		Tranactions: tranactions,
	}

	return res, nil
}
