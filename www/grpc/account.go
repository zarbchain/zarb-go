package grpc

import (
	"context"

	"github.com/zarbchain/zarb-go/crypto"
	zarb "github.com/zarbchain/zarb-go/www/grpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (zs *zarbServer) GetAccount(ctx context.Context, request *zarb.AccountRequest) (*zarb.AccountResponse, error) {
	addr, err := crypto.AddressFromString(request.Address)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid address: %v", err)

	}
	acc := zs.state.Account(addr)
	if acc == nil {
		return nil, status.Errorf(codes.InvalidArgument, "Account not found")

	}
	res := &zarb.AccountResponse{
		Account: &zarb.AccountInfo{
			Address:  acc.Address().String(),
			Number:   int32(acc.Number()),
			Sequence: int64(acc.Sequence()),
			Balance:  acc.Balance(),
		},
	}

	return res, nil

}
