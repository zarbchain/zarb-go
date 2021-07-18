package grpc

import (
	"context"
	"encoding/hex"

	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/tx"
	"github.com/zarbchain/zarb-go/tx/payload"
	zarb "github.com/zarbchain/zarb-go/www/grpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (zs *zarbServer) GetTransaction(ctx context.Context, request *zarb.TransactionRequest) (*zarb.TransactionResponse, error) {
	id, err := crypto.HashFromString(request.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid transaction ID: %v", err.Error())

	}
	trx := zs.state.Transaction(id)
	if trx == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Transaction not found")
	}

	return &zarb.TransactionResponse{
		Tranaction: zs.encodeTransaction(trx),
	}, nil

}

func (zs *zarbServer) SendRawTransaction(ctx context.Context, request *zarb.SendRawTransactionRequest) (*zarb.SendRawTransactionResponse, error) {
	var tx tx.Tx

	hexDecoded, err := hex.DecodeString(request.Data)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Couldn't decode transaction: %v", err)
	}
	if err := tx.Decode(hexDecoded); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Couldn't decode transaction: %v", err)
	}

	if err := tx.SanityCheck(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Couldn't Verify Transaction:  %v", err)
	}

	if err := zs.state.AddPendingTxAndBroadcast(&tx); err != nil {
		return nil, status.Errorf(codes.Aborted, "Couldn't add to Pending pool: %v", err)
	}

	return &zarb.SendRawTransactionResponse{
		Id: tx.ID().String(),
	}, nil
}

func (zs *zarbServer) encodeTransaction(trx *tx.Tx) *zarb.TransactionInfo {
	transaction := &zarb.TransactionInfo{
		Id:        trx.ID().String(),
		Version:   int32(trx.Version()),
		Stamp:     trx.Stamp().String(),
		Sequence:  int64(trx.Sequence()),
		Fee:       trx.Fee(),
		Type:      zarb.PayloadType(trx.PayloadType()),
		Memo:      trx.Memo(),
		PublicKey: trx.PublicKey().String(),
		Signature: trx.Signature().String(),
	}

	switch trx.PayloadType() {
	case payload.PayloadTypeSend:
		pld := trx.Payload().(*payload.SendPayload)
		transaction.Payload = &zarb.TransactionInfo_Send{
			Send: &zarb.SEND_PAYLOAD{
				Sender:   pld.Sender.String(),
				Receiver: pld.Receiver.String(),
				Amount:   pld.Amount,
			},
		}
	case payload.PayloadTypeBond:
		pld := trx.Payload().(*payload.BondPayload)
		transaction.Payload = &zarb.TransactionInfo_Bond{
			Bond: &zarb.BOND_PAYLOAD{
				Bonder:    pld.Bonder.String(),
				Validator: pld.Validator.String(),
				Stake:     pld.Stake,
			},
		}
	case payload.PayloadTypeSortition:
		pld := trx.Payload().(*payload.SortitionPayload)
		proof, _ := pld.Proof.MarshalText()
		transaction.Payload = &zarb.TransactionInfo_Sortition{
			Sortition: &zarb.SORTITION_PAYLOAD{
				Address: pld.Address.String(),
				Proof:   string(proof),
			},
		}
	default:
		zs.logger.Error("payload type not defined", "Type", trx.PayloadType())
	}

	return transaction
}
