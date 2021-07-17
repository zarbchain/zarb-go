package sandbox

import (
	"sync"

	"github.com/zarbchain/zarb-go/account"
	"github.com/zarbchain/zarb-go/committee"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/errors"
	"github.com/zarbchain/zarb-go/logger"
	"github.com/zarbchain/zarb-go/param"
	"github.com/zarbchain/zarb-go/sortition"
	"github.com/zarbchain/zarb-go/store"
	"github.com/zarbchain/zarb-go/validator"
)

type Concrete struct {
	lk sync.RWMutex

	store            store.Reader
	sortition        *sortition.Sortition
	committee        committee.Reader
	accounts         map[crypto.Address]*AccountStatus
	validators       map[crypto.Address]*ValidatorStatus
	params           param.Params
	lastHeight       int
	totalAccounts    int
	totalValidators  int
	totalStakeChange int64
}

type ValidatorStatus struct {
	Validator       validator.Validator
	Updated         bool
	JoinedCommittee bool
}

type AccountStatus struct {
	Account account.Account
	Updated bool
}

func NewSandbox(store store.Reader, params param.Params, lastHeight int, sortition *sortition.Sortition, committee committee.Reader) *Concrete {
	sb := &Concrete{
		store:      store,
		sortition:  sortition,
		committee:  committee,
		lastHeight: lastHeight,
		params:     params,
	}

	sb.accounts = make(map[crypto.Address]*AccountStatus)
	sb.validators = make(map[crypto.Address]*ValidatorStatus)
	sb.totalAccounts = sb.store.TotalAccounts()
	sb.totalValidators = sb.store.TotalValidators()
	sb.totalStakeChange = 0

	return sb
}

func (sb *Concrete) shouldPanicForDuplicatedAddress() {
	//
	// Why we should panic here?
	//
	// Try to make a new item which already exists in store.
	//
	logger.Panic("Duplicated address")
}

func (sb *Concrete) shouldPanicForUnknownAddress() {
	//
	// Why we should panic here?
	//
	// We only update accounts or validators which we have them inside the sandbox.
	// We must either make a new one (i.e. `MakeNewAccount`) or get it from store (i.e. `Account`) in advance.
	//
	logger.Panic("Unknown address")
}

func (sb *Concrete) Account(addr crypto.Address) *account.Account {
	sb.lk.RLock()
	defer sb.lk.RUnlock()

	s, ok := sb.accounts[addr]
	if ok {
		copy := new(account.Account)
		*copy = s.Account
		return copy
	}

	acc, err := sb.store.Account(addr)
	if err != nil {
		return nil
	}
	sb.accounts[addr] = &AccountStatus{
		Account: *acc,
	}

	return acc
}
func (sb *Concrete) MakeNewAccount(addr crypto.Address) *account.Account {
	sb.lk.Lock()
	defer sb.lk.Unlock()

	if sb.store.HasAccount(addr) {
		sb.shouldPanicForDuplicatedAddress()
	}

	acc := account.NewAccount(addr, sb.totalAccounts)
	sb.accounts[addr] = &AccountStatus{
		Account: *acc,
		Updated: true,
	}
	sb.totalAccounts++
	return acc
}

func (sb *Concrete) UpdateAccount(acc *account.Account) {
	sb.lk.Lock()
	defer sb.lk.Unlock()

	addr := acc.Address()
	s, ok := sb.accounts[addr]
	if !ok {
		sb.shouldPanicForUnknownAddress()
	}
	s.Account = *acc
	s.Updated = true
}

func (sb *Concrete) Validator(addr crypto.Address) *validator.Validator {
	sb.lk.RLock()
	defer sb.lk.RUnlock()

	s, ok := sb.validators[addr]
	if ok {
		copy := new(validator.Validator)
		*copy = s.Validator
		return copy
	}

	val, err := sb.store.Validator(addr)
	if err != nil {
		return nil
	}
	sb.validators[addr] = &ValidatorStatus{
		Validator: *val,
	}
	return val
}

func (sb *Concrete) MakeNewValidator(pub crypto.PublicKey) *validator.Validator {
	sb.lk.Lock()
	defer sb.lk.Unlock()

	addr := pub.Address()
	if sb.store.HasValidator(addr) {
		sb.shouldPanicForDuplicatedAddress()
	}

	val := validator.NewValidator(pub, sb.totalValidators)
	sb.validators[addr] = &ValidatorStatus{
		Validator: *val,
		Updated:   true,
	}
	sb.totalValidators++
	return val
}

func (sb *Concrete) UpdateValidator(val *validator.Validator) {
	sb.lk.Lock()
	defer sb.lk.Unlock()

	addr := val.Address()
	s, ok := sb.validators[addr]
	if !ok {
		sb.shouldPanicForUnknownAddress()
	}

	// shouldn't this be power??
	sb.totalStakeChange += val.Stake() - s.Validator.Stake()
	s.Validator = *val
	s.Updated = true
}

func (sb *Concrete) EnterCommittee(blockHash crypto.Hash, addr crypto.Address) error {
	sb.lk.Lock()
	defer sb.lk.Unlock()

	if sb.committee.Contains(addr) {
		return errors.Errorf(errors.ErrGeneric, "this validator already is in the committee")
	}

	valS, ok := sb.validators[addr]
	if !ok {
		return errors.Errorf(errors.ErrGeneric, "unknown validator")
	}

	if valS.JoinedCommittee {
		return errors.Errorf(errors.ErrGeneric, "this validator has joined into committee before")
	}

	if sb.committee.Size() >= sb.params.CommitteeSize {
		oldestJoinedHeight := sb.lastHeight
		committeeStake := int64(0)
		for _, v := range sb.committee.Validators() {
			committeeStake += v.Stake()
			if v.LastJoinedHeight() < oldestJoinedHeight {
				oldestJoinedHeight = v.LastJoinedHeight()
			}
		}
		if sb.lastHeight-oldestJoinedHeight < sb.params.CommitteeSize {
			return errors.Errorf(errors.ErrGeneric, "oldest validator still didn't propose any block")
		}
		joinedStake := int64(0)
		for _, s := range sb.validators {
			if s.JoinedCommittee {
				joinedStake += s.Validator.Stake()
			}
		}

		joinedStake += valS.Validator.Stake()
		if joinedStake >= (committeeStake / 3) {
			return errors.Errorf(errors.ErrGeneric, "in each height only 1/3 of stake can be changed")
		}
	}

	h, _ := sb.store.BlockHeight(blockHash)
	b, err := sb.store.Block(h)
	if err != nil {
		return errors.Errorf(errors.ErrGeneric, "invalid block hash")
	}
	committers := b.LastCertificate().Committers()
	for _, num := range committers {
		if valS.Validator.Number() == num {
			return errors.Errorf(errors.ErrGeneric, "this validator was in the committee in time of sending the sortition")
		}
	}

	valS.JoinedCommittee = true
	return nil
}

func (sb *Concrete) MaxMemoLength() int {
	sb.lk.Lock()
	defer sb.lk.Unlock()

	return sb.params.MaximumMemoLength
}

func (sb *Concrete) FeeFraction() float64 {
	sb.lk.Lock()
	defer sb.lk.Unlock()

	return sb.params.FeeFraction
}

func (sb *Concrete) MinFee() int64 {
	return sb.params.MinimumFee
}

func (sb *Concrete) TransactionToLiveInterval() int {
	sb.lk.RLock()
	defer sb.lk.RUnlock()

	return sb.params.TransactionToLiveInterval
}

func (sb *Concrete) BlockHeight(hash crypto.Hash) int {
	sb.lk.RLock()
	defer sb.lk.RUnlock()

	if hash.EqualsTo(crypto.UndefHash) {
		return 0
	}

	h, err := sb.store.BlockHeight(hash)
	if err != nil {
		return -1
	}

	return h
}

func (sb *Concrete) CurrentHeight() int {
	sb.lk.RLock()
	defer sb.lk.RUnlock()

	return sb.lastHeight + 1
}

func (sb *Concrete) LastBlockHash() crypto.Hash {
	sb.lk.RLock()
	defer sb.lk.RUnlock()

	b, _ := sb.store.Block(sb.lastHeight)
	return b.Hash()
}

func (sb *Concrete) VerifySortition(blockHash crypto.Hash, proof sortition.Proof, val *validator.Validator) bool {
	sb.lk.RLock()
	defer sb.lk.RUnlock()

	return sb.sortition.VerifyProof(blockHash, proof, val.PublicKey(), val.Stake())
}

func (sb *Concrete) IterateAccounts(consumer func(*AccountStatus)) {
	sb.lk.RLock()
	defer sb.lk.RUnlock()

	for _, as := range sb.accounts {
		consumer(as)
	}
}

func (sb *Concrete) IterateValidators(consumer func(*ValidatorStatus)) {
	sb.lk.RLock()
	defer sb.lk.RUnlock()

	for _, vs := range sb.validators {
		consumer(vs)
	}
}

func (sb *Concrete) CommitteeSize() int {
	sb.lk.RLock()
	defer sb.lk.RUnlock()

	return sb.params.CommitteeSize
}

func (sb *Concrete) UnbondInterval() int {
	sb.lk.RLock()
	defer sb.lk.RUnlock()

	return sb.params.UnbondInterval
}

func (sb *Concrete) IsInCommittee(addr crypto.Address) bool {
	return sb.committee.Contains(addr)
}
