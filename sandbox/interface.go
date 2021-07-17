package sandbox

import (
	"github.com/zarbchain/zarb-go/account"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/sortition"
	"github.com/zarbchain/zarb-go/validator"
)

type Sandbox interface {
	Account(crypto.Address) *account.Account
	MakeNewAccount(crypto.Address) *account.Account
	UpdateAccount(*account.Account)

	Validator(crypto.Address) *validator.Validator
	MakeNewValidator(crypto.PublicKey) *validator.Validator
	UpdateValidator(*validator.Validator)
	IsInCommittee(crypto.Address) bool

	VerifySortition(crypto.Hash, sortition.Proof, *validator.Validator) bool
	EnterCommittee(crypto.Hash, crypto.Address) error

	CommitteeSize() int
	UnbondInterval() int
	CurrentHeight() int
	BlockHeight(crypto.Hash) int
	TransactionToLiveInterval() int
	MaxMemoLength() int
	FeeFraction() float64
	MinFee() int64
}
