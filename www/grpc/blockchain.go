package grpc

import (
	"context"

	"github.com/fxamacker/cbor/v2"
	zarb "github.com/zarbchain/zarb-go/www/grpc/proto"
)

func (zs *zarbServer) GetBlockchainInfo(ctx context.Context, request *zarb.BlockchainInfoRequest) (*zarb.BlockchainInfoResponse, error) {
	height := zs.state.LastBlockHeight()

	return &zarb.BlockchainInfoResponse{
		Height:        int64(height),
		LastBlockHash: zs.state.LastBlockHash().String(),
	}, nil
}
func (zs *zarbServer) GetNetworkInfo(ctx context.Context, request *zarb.NetworkInfoRequest) (*zarb.NetworkInfoResponse, error) {
	// Create response peers
	rps := make([]*zarb.PeerInfo, int32(len(zs.sync.Peers())))

	// cast to response peers from synced peers
	for i, peer := range zs.sync.Peers() {
		rps[i] = new(zarb.PeerInfo)
		p := rps[i]

		bs, err := cbor.Marshal(peer.NodeVersion())
		if err != nil {
			zs.logger.Error("Couldn't marshal peer version", "err", err)
			continue
		}
		p.NodeVersion = bs

		p.PeerId = peer.PeerID().String()
		p.Moniker = peer.Moniker()
		p.PublicKey = peer.PublicKey().String()
		p.InitialBlockDownload = peer.InitialBlockDownload()
		p.Height = int32(peer.Height())
		p.ReceivedMessages = int32(peer.ReceivedMessages())
		p.InvalidMessages = int32(peer.InvalidMessages())
		p.ReceivedBytes = int32(peer.ReceivedBytes())
	}

	return &zarb.NetworkInfoResponse{
		PeerId: zs.sync.SelfID().String(),
		Peers:  rps,
	}, nil
}
