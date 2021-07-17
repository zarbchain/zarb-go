package grpc

import (
	"context"

	"github.com/zarbchain/zarb-go/crypto"
	zarb "github.com/zarbchain/zarb-go/www/grpc/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (zs *zarbServer) GetValidatorByNumber(ctx context.Context, request *zarb.ValidatorByNumberRequest) (*zarb.ValidatorResponse, error) {
	validator := zs.state.ValidatorByNumber(int(request.Number))
	if validator == nil {
		return nil, status.Errorf(codes.NotFound, "NotFound Validator Address")
	}

	// TODO: make a function
	// proto validator from native validator
	return &zarb.ValidatorResponse{
		Validator: &zarb.Validator{
			PublicKey:         validator.PublicKey().String(),
			Address:           validator.Address().String(),
			Number:            int32(validator.Number()),
			Sequence:          int32(validator.Sequence()),
			Stake:             validator.Stake(),
			LastBondingHeight: int32(validator.LastBondingHeight()),
			LastJoinedHeight:  int32(validator.LastJoinedHeight()),
			UnbondingHeight:   int32(validator.UnbondingHeight()),
		},
	}, nil
}

func (zs *zarbServer) GetValidator(ctx context.Context, request *zarb.ValidatorRequest) (*zarb.ValidatorResponse, error) {
	addr, err := crypto.AddressFromString(request.Address)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Invalid Validator Address: %s", err.Error())
	}
	validator := zs.state.Validator(addr)
	if validator == nil {
		return nil, status.Errorf(codes.NotFound, "NotFound Validator Address")
	}

	// TODO: make a function
	// proto validator from native validator
	return &zarb.ValidatorResponse{
		Validator: &zarb.Validator{
			PublicKey:         validator.PublicKey().String(),
			Address:           validator.Address().String(),
			Number:            int32(validator.Number()),
			Sequence:          int32(validator.Sequence()),
			Stake:             validator.Stake(),
			LastBondingHeight: int32(validator.LastBondingHeight()),
			LastJoinedHeight:  int32(validator.LastJoinedHeight()),
			UnbondingHeight:   int32(validator.UnbondingHeight()),
		},
	}, nil
}
func (zs *zarbServer) GetValidators(ctx context.Context, request *zarb.ValidatorsRequest) (*zarb.ValidatorsResponse, error) {
	validators := zs.state.CommitteeValidators()
	validatorsResp := make([]*zarb.Validator, 0)
	for _, v := range validators {
		// TODO: make a function
		// proto validator from native validator
		validatorsResp = append(validatorsResp, &zarb.Validator{
			PublicKey:         v.PublicKey().String(),
			Address:           v.Address().String(),
			Number:            int32(v.Number()),
			Sequence:          int32(v.Sequence()),
			Stake:             v.Stake(),
			LastBondingHeight: int32(v.LastBondingHeight()),
			LastJoinedHeight:  int32(v.LastJoinedHeight()),
			UnbondingHeight:   int32(v.UnbondingHeight()),
		})
	}
	return &zarb.ValidatorsResponse{Validators: validatorsResp}, nil
}
