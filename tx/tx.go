package tx

import (
	"encoding/json"
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/errors"
	"github.com/zarbchain/zarb-go/sortition"
	"github.com/zarbchain/zarb-go/tx/payload"
)

type ID = crypto.Hash

type Tx struct {
	// TODO: Memorizing ID is thread safe?
	memorizedID   *ID
	sanityChecked bool

	data txData
}

type txData struct {
	Version   int
	Stamp     crypto.Hash
	Sequence  int
	Fee       int64
	Type      payload.Type
	Payload   payload.Payload
	Memo      string
	PublicKey *crypto.PublicKey
	Signature *crypto.Signature
}

func (tx *Tx) Version() int                 { return tx.data.Version }
func (tx *Tx) Stamp() crypto.Hash           { return tx.data.Stamp }
func (tx *Tx) Sequence() int                { return tx.data.Sequence }
func (tx *Tx) PayloadType() payload.Type    { return tx.data.Type }
func (tx *Tx) Payload() payload.Payload     { return tx.data.Payload }
func (tx *Tx) Fee() int64                   { return tx.data.Fee }
func (tx *Tx) Memo() string                 { return tx.data.Memo }
func (tx *Tx) PublicKey() *crypto.PublicKey { return tx.data.PublicKey }
func (tx *Tx) Signature() *crypto.Signature { return tx.data.Signature }

func (tx *Tx) SetSignature(sig crypto.Signature) {
	tx.sanityChecked = false
	tx.data.Signature = &sig
}

func (tx *Tx) SetPublicKey(pub crypto.PublicKey) {
	tx.sanityChecked = false
	tx.data.PublicKey = &pub
}

func (tx *Tx) SanityCheck() error {
	if tx.sanityChecked {
		return nil
	}
	if tx.Version() != 1 {
		return errors.Errorf(errors.ErrInvalidTx, "invalid version")
	}
	if tx.Sequence() < 0 {
		return errors.Errorf(errors.ErrInvalidTx, "invalid sequence")
	}
	if err := tx.checkFee(); err != nil {
		return err
	}
	if err := tx.checkSignature(); err != nil {
		return err
	}
	if tx.PayloadType() != tx.Payload().Type() {
		return errors.Errorf(errors.ErrInvalidTx, "invalid payload type")
	}
	if err := tx.Payload().SanityCheck(); err != nil {
		return err
	}

	tx.sanityChecked = true

	return nil
}

func (tx *Tx) checkFee() error {
	if tx.IsFreeTx() {
		if tx.Fee() != 0 {
			return errors.Errorf(errors.ErrInvalidTx, "fee should set to zero")
		}
	} else {
		if tx.Fee() <= 0 {
			return errors.Errorf(errors.ErrInvalidTx, "fee is invalid")
		}
	}

	return nil
}

func (tx *Tx) checkSignature() error {
	if tx.IsMintbaseTx() {
		if tx.PublicKey() != nil {
			return errors.Errorf(errors.ErrInvalidTx, "subsidy transaction should not have public key")
		}
		if tx.Signature() != nil {
			return errors.Errorf(errors.ErrInvalidTx, "subsidy transaction should not have signature")
		}
	} else {
		if tx.PublicKey() == nil {
			return errors.Errorf(errors.ErrInvalidTx, "no public key")
		}
		if tx.Signature() == nil {
			return errors.Errorf(errors.ErrInvalidTx, "no signature")
		}
		if err := tx.PublicKey().SanityCheck(); err != nil {
			return errors.Errorf(errors.ErrInvalidTx, "invalid pubic key")
		}
		if err := tx.Signature().SanityCheck(); err != nil {
			return errors.Errorf(errors.ErrInvalidTx, "invalid signature")
		}
		if !tx.Payload().Signer().Verify(*tx.PublicKey()) {
			return errors.Errorf(errors.ErrInvalidTx, "invalid public key")
		}
		bs := tx.SignBytes()
		if !tx.PublicKey().Verify(bs, *tx.Signature()) {
			return errors.Errorf(errors.ErrInvalidTx, "invalid signature")
		}
	}
	return nil
}

type _txData struct {
	Version   int               `cbor:"1,keyasint"`
	Stamp     crypto.Hash       `cbor:"2,keyasint"`
	Sequence  int               `cbor:"3,keyasint"`
	Fee       int64             `cbor:"4,keyasint"`
	Type      payload.Type      `cbor:"5,keyasint"`
	Payload   cbor.RawMessage   `cbor:"6,keyasint"`
	Memo      string            `cbor:"7,keyasint,omitempty"`
	PublicKey *crypto.PublicKey `cbor:"20,keyasint,omitempty"`
	Signature *crypto.Signature `cbor:"21,keyasint,omitempty"`
}

func (tx *Tx) MarshalCBOR() ([]byte, error) {
	bs, err := cbor.Marshal(tx.data.Payload)
	if err != nil {
		return nil, err
	}

	_data := &_txData{
		Version:   tx.data.Version,
		Stamp:     tx.data.Stamp,
		Sequence:  tx.data.Sequence,
		Type:      tx.data.Type,
		Payload:   bs,
		Fee:       tx.data.Fee,
		Memo:      tx.data.Memo,
		PublicKey: tx.data.PublicKey,
		Signature: tx.data.Signature,
	}

	return cbor.Marshal(_data)
}

func (tx *Tx) UnmarshalCBOR(bs []byte) error {
	var _data _txData
	err := cbor.Unmarshal(bs, &_data)
	if err != nil {
		return err
	}

	var p payload.Payload
	switch _data.Type {
	case payload.PayloadTypeSend:
		p = &payload.SendPayload{}
	case payload.PayloadTypeBond:
		p = &payload.BondPayload{}
	case payload.PayloadTypeUnbond:
		p = &payload.UnbondPayload{}
	case payload.PayloadTypeWithdraw:
		p = &payload.WithdrawPayload{}
	case payload.PayloadTypeSortition:
		p = &payload.SortitionPayload{}

	default:
		return errors.Errorf(errors.ErrInvalidMessage, "invalid payload")
	}

	tx.data.Version = _data.Version
	tx.data.Stamp = _data.Stamp
	tx.data.Sequence = _data.Sequence
	tx.data.Type = _data.Type
	tx.data.Payload = p
	tx.data.Fee = _data.Fee
	tx.data.Memo = _data.Memo
	tx.data.PublicKey = _data.PublicKey
	tx.data.Signature = _data.Signature

	return cbor.Unmarshal(_data.Payload, p)
}

func (tx *Tx) MarshalJSON() ([]byte, error) {
	return json.Marshal(tx.data)
}

func (tx *Tx) Encode() ([]byte, error) {
	return tx.MarshalCBOR()
}

func (tx *Tx) Decode(bs []byte) error {
	return tx.UnmarshalCBOR(bs)
}

func (tx *Tx) Fingerprint() string {
	return fmt.Sprintf("{⌘ %v 🏵 %v %v}",
		tx.ID().Fingerprint(),
		tx.data.Stamp.Fingerprint(),
		tx.data.Payload.Fingerprint())
}

func (tx Tx) SignBytes() []byte {
	tx.data.PublicKey = nil
	tx.data.Signature = nil

	bz, _ := tx.MarshalCBOR()
	return bz
}

func (tx *Tx) ID() ID {
	if tx.memorizedID == nil {
		id := crypto.HashH(tx.SignBytes())
		tx.memorizedID = &id
	}

	return *tx.memorizedID
}

func (tx *Tx) IsBondTx() bool {
	return tx.data.Type == payload.PayloadTypeBond
}

func (tx *Tx) IsMintbaseTx() bool {
	return tx.data.Type == payload.PayloadTypeSend &&
		tx.data.Payload.Signer().EqualsTo(crypto.TreasuryAddress)
}

func (tx *Tx) IsSortitionTx() bool {
	return tx.data.Type == payload.PayloadTypeSortition
}

func (tx *Tx) IsUnbondTx() bool {
	return tx.data.Type == payload.PayloadTypeUnbond
}

func (tx *Tx) IsWithdrawTx() bool {
	return tx.data.Type == payload.PayloadTypeWithdraw
}

//IsFreeTx will return if trx's fee is 0
func (tx *Tx) IsFreeTx() bool {
	return tx.IsMintbaseTx() || tx.IsSortitionTx() || tx.IsUnbondTx()
}

// ---------
// For tests
func GenerateTestSendTx() (*Tx, crypto.Signer) {
	h := crypto.GenerateTestHash()
	s := crypto.GenerateTestSigner()
	a, _, _ := crypto.GenerateTestKeyPair()
	tx := NewSendTx(h, 110, s.Address(), a, 1000, 1000, "test send-tx")
	s.SignMsg(tx)
	return tx, s
}

func GenerateTestBondTx() (*Tx, crypto.Signer) {
	h := crypto.GenerateTestHash()
	s := crypto.GenerateTestSigner()
	_, pb, _ := crypto.GenerateTestKeyPair()
	tx := NewBondTx(h, 110, s.Address(), pb, 1000, 1000, "test bond-tx")
	s.SignMsg(tx)
	return tx, s
}

func GenerateTestSortitionTx() (*Tx, crypto.Signer) {
	h := crypto.GenerateTestHash()
	s := crypto.GenerateTestSigner()
	proof := sortition.GenerateRandomProof()
	tx := NewSortitionTx(h, 110, s.Address(), proof)
	s.SignMsg(tx)
	return tx, s
}
