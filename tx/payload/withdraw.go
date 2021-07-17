package payload

import (
	"fmt"

	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/errors"
)

type WithdrawPayload struct {
	From   crypto.Address `cbor:"1,keyasint"` // withdraw from validator address
	To     crypto.Address `cbor:"2,keyasint"` // deposit to account address
	Amount int64          `cbor:"3,keyasint"` // amount to deposit
}

func (p *WithdrawPayload) Type() Type {
	return PayloadTypeWithdraw
}

func (p *WithdrawPayload) Signer() crypto.Address {
	return p.From
}

func (p *WithdrawPayload) Value() int64 {
	return p.Amount
}

func (p *WithdrawPayload) SanityCheck() error {
	if p.Amount < 0 {
		return errors.Errorf(errors.ErrInvalidTx, "invalid amount")
	}
	if err := p.To.SanityCheck(); err != nil {
		return errors.Errorf(errors.ErrInvalidTx, "invalid receiver address")
	}

	return nil
}

func (p *WithdrawPayload) Fingerprint() string {
	return fmt.Sprintf("{WithdrawPayload: %v->%v 💸 %v",
		p.From.Fingerprint(),
		p.To.Fingerprint(),
		p.Amount)
}
