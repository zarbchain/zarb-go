package payload

import (
	"fmt"

	"github.com/zarbchain/zarb-go/crypto"
)

type Type int

const (
	PayloadTypeSend      = Type(1)
	PayloadTypeBond      = Type(2)
	PayloadTypeSortition = Type(3)
	PayloadTypeUnbond    = Type(4)
	PayloadTypeWithdraw  = Type(5)
)

func (t Type) String() string {
	switch t {
	case PayloadTypeSend:
		return "send"
	case PayloadTypeBond:
		return "bond"
	case PayloadTypeUnbond:
		return "unbond"
	case PayloadTypeWithdraw:
		return "withdraw"
	case PayloadTypeSortition:
		return "sortition"
	}
	return fmt.Sprintf("%d", t)
}

type Payload interface {
	Signer() crypto.Address
	Value() int64
	Type() Type
	SanityCheck() error
	Fingerprint() string
}
