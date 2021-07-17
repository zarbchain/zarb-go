package payload

import (
	"fmt"

	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/errors"
)

type BondPayload struct {
	Bonder    crypto.Address   `cbor:"1,keyasint"`
	Validator crypto.PublicKey `cbor:"2,keyasint"`
	Stake     int64            `cbor:"3,keyasint"`
}

func (p *BondPayload) Type() Type {
	return PayloadTypeBond
}

func (p *BondPayload) Signer() crypto.Address {
	return p.Bonder
}

func (p *BondPayload) Value() int64 {
	return p.Stake
}

func (p *BondPayload) SanityCheck() error {
	if p.Stake < 0 {
		return errors.Errorf(errors.ErrInvalidTx, "invalid amount")
	}
	if err := p.Bonder.SanityCheck(); err != nil {
		return errors.Errorf(errors.ErrInvalidTx, "invalid Bonder address")
	}
	if err := p.Validator.SanityCheck(); err != nil {
		return errors.Errorf(errors.ErrInvalidTx, "invalid receiver address")
	}

	return nil
}

func (p *BondPayload) Fingerprint() string {
	return fmt.Sprintf("{Bond: %v->%v 🔐 %v",
		p.Bonder.Fingerprint(),
		p.Validator.Address().Fingerprint(),
		p.Stake)
}
