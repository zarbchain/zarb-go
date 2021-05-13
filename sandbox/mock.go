package sandbox

import (
	"fmt"

	"github.com/zarbchain/zarb-go/account"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/param"
	"github.com/zarbchain/zarb-go/sortition"
	"github.com/zarbchain/zarb-go/validator"
)

var _ Sandbox = &MockSandbox{}

// MockSandbox is a testing mock for sandbox
type MockSandbox struct {
	Accounts           map[crypto.Address]*account.Account
	Validators         map[crypto.Address]*validator.Validator
	Stamps             map[crypto.Hash]int
	CurHeight          int
	Params             param.Params
	TotalAccount       int
	TotalValidator     int
	AcceptSortition    bool
	WelcomeToCommittee bool
	InCommittee        bool
}

func MockingSandbox() *MockSandbox {
	return &MockSandbox{
		Accounts:   make(map[crypto.Address]*account.Account),
		Validators: make(map[crypto.Address]*validator.Validator),
		Stamps:     make(map[crypto.Hash]int),
		Params:     param.DefaultParams(),
	}
}

func (m *MockSandbox) Account(addr crypto.Address) *account.Account {
	acc, ok := m.Accounts[addr]
	if !ok {
		return nil
	}
	return acc
}
func (m *MockSandbox) MakeNewAccount(addr crypto.Address) *account.Account {
	a := account.NewAccount(addr, m.TotalAccount)
	m.TotalAccount++
	return a
}
func (m *MockSandbox) UpdateAccount(acc *account.Account) {
	m.Accounts[acc.Address()] = acc
}
func (m *MockSandbox) Validator(addr crypto.Address) *validator.Validator {
	val, ok := m.Validators[addr]
	if !ok {
		return nil
	}
	return val
}
func (m *MockSandbox) MakeNewValidator(pub crypto.PublicKey) *validator.Validator {
	v := validator.NewValidator(pub, m.TotalAccount, m.CurHeight)
	m.TotalValidator++
	return v
}
func (m *MockSandbox) UpdateValidator(val *validator.Validator) {
	m.Validators[val.Address()] = val

}
func (m *MockSandbox) EnterCommittee(hash crypto.Hash, addr crypto.Address) error {
	if !m.WelcomeToCommittee {
		return fmt.Errorf("cannot enter to the committee")
	}
	return nil
}
func (m *MockSandbox) VerifySortition(blockHash crypto.Hash, proof sortition.Proof, val *validator.Validator) bool {
	return m.AcceptSortition
}
func (m *MockSandbox) CurrentHeight() int {
	return m.CurHeight
}
func (m *MockSandbox) BlockHeight(hash crypto.Hash) int {
	h, ok := m.Stamps[hash]
	if !ok {
		return -1
	}
	return h
}
func (m *MockSandbox) TransactionToLiveInterval() int {
	return m.Params.TransactionToLiveInterval
}
func (m *MockSandbox) MaxMemoLength() int {
	return m.Params.MaximumMemoLength
}
func (m *MockSandbox) FeeFraction() float64 {
	return m.Params.FeeFraction
}
func (m *MockSandbox) MinFee() int64 {
	return m.Params.MinimumFee
}

func (m *MockSandbox) AppendStampAndUpdateHeight(height int, stamp crypto.Hash) {
	m.Stamps[stamp] = height
	m.CurHeight = height + 1
}

func (m *MockSandbox) AccSeq(a crypto.Address) int {
	return m.Accounts[a].Sequence()
}

func (m *MockSandbox) CommitteeSize() int {
	return m.Params.CommitteeSize
}

func (m *MockSandbox) IsInCommittee(crypto.Address) bool {
	return m.InCommittee
}
