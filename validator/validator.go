package validator

import (
	"encoding/json"
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/zarbchain/zarb-go/crypto"
)

type Validator struct {
	data validatorData
}

type validatorData struct {
	PublicKey         crypto.PublicKey `cbor:"1,keyasint"`
	Number            int              `cbor:"2,keyasint"`
	Sequence          int              `cbor:"3,keyasint"`
	Stake             int64            `cbor:"4,keyasint"`
	LastBondingHeight int              `cbor:"5,keyasint"`
	UnbondingHeight   int              `cbor:"6,keyasint"`
	LastJoinedHeight  int              `cbor:"7,keyasint"`
}

func NewValidator(publicKey crypto.PublicKey, number int) *Validator {
	val := &Validator{
		data: validatorData{
			PublicKey: publicKey,
			Number:    number,
		},
	}
	return val
}

func (val *Validator) PublicKey() crypto.PublicKey { return val.data.PublicKey }
func (val *Validator) Address() crypto.Address     { return val.data.PublicKey.Address() }
func (val *Validator) Number() int                 { return val.data.Number }
func (val *Validator) Sequence() int               { return val.data.Sequence }
func (val *Validator) Stake() int64                { return val.data.Stake }
func (val *Validator) LastBondingHeight() int      { return val.data.LastBondingHeight }
func (val *Validator) UnbondingHeight() int        { return val.data.UnbondingHeight }
func (val *Validator) LastJoinedHeight() int       { return val.data.LastJoinedHeight }

func (val Validator) Power() int64 {
	//if the validator requested to unbond ignore stake
	if val.data.UnbondingHeight > 0 {
		return 0
	} else if val.data.Stake == 0 { // Only bootstrap validators at genesis block has no stake
		return 1
	}
	return val.data.Stake
}

// AddToStake increases the stake by bonding transaction
func (val *Validator) AddToStake(amt int64) {
	val.data.Stake += amt
}

// IncSequence increases the sequence anytime this validator signs a transaction
func (val *Validator) IncSequence() {
	val.data.Sequence++
}

// UpdateLastJoinedHeight updates the last height that this validator joined the committee
func (val *Validator) UpdateLastJoinedHeight(height int) {
	val.data.LastJoinedHeight = height
}

// UpdateLastBondingHeight updates the last height that this validator bonded some stakes
func (val *Validator) UpdateLastBondingHeight(height int) {
	val.data.LastBondingHeight = height
}

// UpdateUnbondingHeight updates the UnbondingHeight validator requested unbonding it's stake
func (val *Validator) UpdateUnbondingHeight(height int) {
	val.data.UnbondingHeight = height
}

// Hash return the hash of this validator
func (val *Validator) Hash() crypto.Hash {
	bs, err := val.Encode()
	if err != nil {
		panic(err)
	}
	return crypto.HashH(bs)
}

///---- Serialization methods
func (val Validator) Encode() ([]byte, error) {
	return cbor.Marshal(val.data)
}

func (val *Validator) Decode(bs []byte) error {
	return cbor.Unmarshal(bs, &val.data)
}

func (val Validator) MarshalJSON() ([]byte, error) {
	return json.Marshal(val.data)
}

func (val *Validator) UnmarshalJSON(bs []byte) error {
	return json.Unmarshal(bs, &val.data)
}

func (val Validator) Fingerprint() string {
	return fmt.Sprintf("{%s %d %v}",
		val.Address().Fingerprint(),
		val.Sequence(),
		val.Stake())
}

// GenerateTestValidator generates a validator for testing purpose
func GenerateTestValidator(number int) (*Validator, crypto.Signer) {
	signer := crypto.GenerateTestSigner()
	val := NewValidator(signer.PublicKey(), number)
	val.data.Stake = 777777777
	val.data.Sequence = 77
	return val, signer
}
