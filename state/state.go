package state

import (
	"fmt"
	"sync"
	"time"

	"github.com/zarbchain/zarb-go/account"
	"github.com/zarbchain/zarb-go/block"
	"github.com/zarbchain/zarb-go/committee"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/errors"
	"github.com/zarbchain/zarb-go/execution"
	"github.com/zarbchain/zarb-go/genesis"
	"github.com/zarbchain/zarb-go/logger"
	"github.com/zarbchain/zarb-go/param"
	"github.com/zarbchain/zarb-go/sandbox"
	"github.com/zarbchain/zarb-go/sortition"
	"github.com/zarbchain/zarb-go/state/lastinfo"
	"github.com/zarbchain/zarb-go/store"
	"github.com/zarbchain/zarb-go/tx"
	"github.com/zarbchain/zarb-go/txpool"
	"github.com/zarbchain/zarb-go/util"
	"github.com/zarbchain/zarb-go/validator"
)

type state struct {
	lk sync.RWMutex

	config       *Config
	signer       crypto.Signer
	mintbaseAddr crypto.Address
	genDoc       *genesis.Genesis
	store        store.Store
	params       param.Params
	txPool       txpool.TxPool
	committee    *committee.Committee
	sortition    *sortition.Sortition
	lastInfo     *lastinfo.LastInfo
	logger       *logger.Logger
}

func LoadOrNewState(
	conf *Config,
	genDoc *genesis.Genesis,
	signer crypto.Signer,
	store store.Store,
	txPool txpool.TxPool) (Facade, error) {

	var mintbaseAddr crypto.Address
	if conf.MintbaseAddress != "" {
		addr, err := crypto.AddressFromString(conf.MintbaseAddress)
		if err != nil {
			return nil, err
		}
		mintbaseAddr = addr
	} else {
		mintbaseAddr = signer.Address()
	}

	st := &state{
		config:       conf,
		genDoc:       genDoc,
		txPool:       txPool,
		params:       genDoc.Params(),
		signer:       signer,
		store:        store,
		mintbaseAddr: mintbaseAddr,
		sortition:    sortition.NewSortition(),
		lastInfo:     lastinfo.NewLastInfo(store),
	}
	st.logger = logger.NewLogger("_state", st)
	st.store = store

	if store.HasAnyBlock() {
		err := st.tryLoadLastInfo()
		if err != nil {
			return nil, err
		}
	} else {
		err := st.makeGenesisState(genDoc)
		if err != nil {
			return nil, err
		}
	}

	txPool.SetNewSandboxAndRecheck(st.concreteSandbox())

	return st, nil
}

func (st *state) concreteSandbox() *sandbox.Concrete {
	return sandbox.NewSandbox(st.store, st.params, st.lastInfo.BlockHeight(), st.sortition, st.committee)
}

func (st *state) tryLoadLastInfo() error {
	// Make sure genesis hash is same
	//
	// This check is not important because genesis state is committed.
	// But it is good to have it to make sure genesis doc hasn't changed
	genHash := st.calculateGenesisStateHashFromGenesisDoc()
	blockOne, err := st.store.Block(1)
	if err != nil {
		return err
	}
	if !genHash.EqualsTo(blockOne.Header().StateHash()) {
		return fmt.Errorf("invalid genesis doc")
	}

	logger.Info("Try to load the last state info")
	committee, err := st.lastInfo.RestoreLastInfo(st.params.CommitteeSize, st.sortition)
	if err != nil {
		return err
	}

	st.committee = committee

	return nil
}

func (st *state) makeGenesisState(genDoc *genesis.Genesis) error {
	accs := genDoc.Accounts()
	for _, acc := range accs {
		st.store.UpdateAccount(acc)
	}

	totalStake := int64(0)
	vals := genDoc.Validators()
	for _, val := range vals {
		st.store.UpdateValidator(val)
		totalStake += val.Stake()
	}

	err := st.store.WriteBatch()
	if err != nil {
		return err
	}

	committee, err := committee.NewCommittee(vals, st.params.CommitteeSize, vals[0].Address())
	if err != nil {
		return err
	}
	st.committee = committee
	st.lastInfo.SetBlockTime(genDoc.GenesisTime())

	return nil
}

func (st *state) Close() error {
	st.lk.RLock()
	defer st.lk.RUnlock()

	return st.store.Close()
}

func (st *state) GenesisHash() crypto.Hash {
	st.lk.RLock()
	defer st.lk.RUnlock()

	return st.genDoc.Hash()
}

func (st *state) LastBlockHeight() int {
	st.lk.RLock()
	defer st.lk.RUnlock()

	return st.lastInfo.BlockHeight()
}

func (st *state) LastBlockHash() crypto.Hash {
	st.lk.RLock()
	defer st.lk.RUnlock()

	return st.lastInfo.BlockHash()
}

func (st *state) LastBlockTime() time.Time {
	st.lk.RLock()
	defer st.lk.RUnlock()

	return st.lastInfo.BlockTime()
}

func (st *state) LastCertificate() *block.Certificate {
	st.lk.RLock()
	defer st.lk.RUnlock()

	return st.lastInfo.Certificate()
}

func (st *state) BlockTime() time.Duration {
	st.lk.RLock()
	defer st.lk.RUnlock()

	return st.params.BlockTime()
}

func (st *state) UpdateLastCertificate(cert *block.Certificate) error {
	st.lk.Lock()
	defer st.lk.Unlock()

	// Check if certificate has more signers ...
	if len(cert.Absentees()) < len(st.lastInfo.Certificate().Absentees()) {
		if err := st.validateCertificateForPreviousHeight(cert); err != nil {
			st.logger.Warn("Try to update last certificate, but it's invalid", "err", err)
			return err
		}
		st.lastInfo.SetCertificate(cert)
	}

	return nil
}

func (st *state) createSubsidyTx(fee int64) *tx.Tx {
	acc, err := st.store.Account(crypto.TreasuryAddress)
	if err != nil {
		return nil
	}
	stamp := st.lastInfo.BlockHash()
	seq := acc.Sequence() + 1
	tx := tx.NewMintbaseTx(stamp, seq, st.mintbaseAddr, st.params.BlockReward+fee, "")
	return tx
}

func (st *state) ProposeBlock(round int) (*block.Block, error) {
	st.lk.Lock()
	defer st.lk.Unlock()

	if !st.committee.IsProposer(st.signer.Address(), round) {
		return nil, errors.Errorf(errors.ErrInvalidAddress, "we are not propser for this round")
	}

	// Create new sandbox and execute transactions
	sb := st.concreteSandbox()
	exe := execution.NewExecution()

	txIDs := block.NewTxIDs()

	// Re-chaeck all transactions again, remove invalid ones
	trxs := st.txPool.AllTransactions()
	for _, trx := range trxs {
		// All subsidy transactions (probably from invalid rounds)
		// should be removed from the pool
		if trx.IsMintbaseTx() {
			st.logger.Debug("Found duplicated subsidy transaction", "tx", trx)
			st.txPool.RemoveTx(trx.ID())
			continue
		}

		if err := exe.Execute(trx, sb); err != nil {
			st.logger.Debug("Found invalid transaction", "tx", trx, "err", err)
			st.txPool.RemoveTx(trx.ID())
		} else {
			txIDs.Append(trx.ID())

			if txIDs.Len() >= st.params.MaximumTransactionPerBlock {
				break
			}
		}
	}

	subsidyTx := st.createSubsidyTx(exe.AccumulatedFee())
	if subsidyTx == nil {
		st.logger.Error("Probably the node is shutting down.")
		return nil, errors.Errorf(errors.ErrInvalidBlock, "no subsidy transaction")
	}
	if err := st.txPool.AppendTxAndBroadcast(subsidyTx); err != nil {
		st.logger.Error("Our subsidy transaction is invalid. Why?", "err", err)
		return nil, err
	}
	txIDs.Prepend(subsidyTx.ID())

	stateHash := st.stateHash()
	timestamp := st.proposeNextBlockTime()
	newSortitionSeed := st.lastInfo.SortitionSeed().Generate(st.signer)

	block := block.MakeBlock(
		st.params.BlockVersion,
		timestamp,
		txIDs,
		st.lastInfo.BlockHash(),
		stateHash,
		st.lastInfo.Certificate(),
		newSortitionSeed,
		st.signer.Address())

	return block, nil
}

func (st *state) ValidateBlock(block *block.Block) error {
	st.lk.Lock()
	defer st.lk.Unlock()

	if err := st.validateBlock(block); err != nil {
		return err
	}

	t := block.Header().Time()
	if err := st.validateBlockTime(t); err != nil {
		return err
	}

	sb := st.concreteSandbox()
	_, err := st.executeBlock(block, sb)
	if err != nil {
		return err
	}

	return nil
}

func (st *state) CommitBlock(height int, block *block.Block, cert *block.Certificate) error {
	st.lk.Lock()
	defer st.lk.Unlock()

	if height != st.lastInfo.BlockHeight()+1 {
		/// Returning error here will cause so many error logs during syncing blockchain
		/// Syncing is asynchronous job and we might receive blocks not in order
		st.logger.Debug("Unexpected block height", "height", height)
		return nil
	}

	err := st.validateCertificate(cert, block.Hash())
	if err != nil {
		return err
	}

	/// There are two modules that can commit a block: Consensus and Syncer.
	/// Consensus engine is ours, we have full control over that and we know when and why a block should be committed.
	/// In the other side, Syncer module receives new blocks from the network and tries to commit them.
	/// We should never have a fork in our blockchain. but if it happens, here we can catch it.
	if !block.Header().LastBlockHash().EqualsTo(st.lastInfo.BlockHash()) {
		st.logger.Panic("A possible fork is detected", "our hash", st.lastInfo.BlockHash(), "block hash", block.Header().LastBlockHash())
		return errors.Error(errors.ErrInvalidBlock)
	}

	err = st.validateBlock(block)
	if err != nil {
		return err
	}

	// Verify proposer
	proposer := st.committee.Proposer(cert.Round())
	if !proposer.Address().EqualsTo(block.Header().ProposerAddress()) {
		return errors.Errorf(errors.ErrInvalidBlock, "invalid proposer. Expected %s, got %s", proposer.Address(), block.Header().ProposerAddress())
	}
	// Validate sortition seed
	if !block.Header().SortitionSeed().Validate(proposer.PublicKey(), st.lastInfo.SortitionSeed()) {
		return errors.Errorf(errors.ErrInvalidBlock, "invalid sortition seed.")
	}

	// -----------------------------------
	// Execute block
	sb := st.concreteSandbox()
	trxs, err := st.executeBlock(block, sb)
	if err != nil {
		return err
	}

	// -----------------------------------
	// Commit block
	st.lastInfo.SetBlockHeight(st.lastInfo.BlockHeight() + 1)
	st.lastInfo.SetBlockHash(block.Hash())
	st.lastInfo.SetBlockTime(block.Header().Time())
	st.lastInfo.SetSortitionSeed(block.Header().SortitionSeed())
	st.lastInfo.SetCertificate(cert)
	st.lastInfo.SaveLastInfo()

	// Commit and update the committee
	st.commitSandbox(sb, cert.Round())

	st.store.SaveBlock(st.lastInfo.BlockHeight(), block)

	// Save txs and receipts
	for _, trx := range trxs {
		st.txPool.RemoveTx(trx.ID())
		st.store.SaveTransaction(trx)
	}

	if err := st.store.WriteBatch(); err != nil {
		st.logger.Panic("Unable to update state", "err", err)
	}

	st.logger.Info("New block is committed", "block", block, "round", cert.Round())

	// -----------------------------------
	// Update sortition params and evaluate sortition
	st.sortition.SetParams(block.Hash(), block.Header().SortitionSeed(), st.poolStake())

	// Evaluate sortition before updating the committee
	if st.evaluateSortition() {
		st.logger.Info("👏 This validator is chosen to be in the committee", "address", st.signer.Address())
	}

	// -----------------------------------
	// At this point we can assign new sandbox to tx pool
	st.txPool.SetNewSandboxAndRecheck(st.concreteSandbox())

	return nil
}

func (st *state) evaluateSortition() bool {
	if st.committee.Contains(st.signer.Address()) {
		// We are in the committee right now
		return false
	}

	val, _ := st.store.Validator(st.signer.Address())
	if val == nil {
		// We are not a validator
		return false
	}

	if st.lastInfo.BlockHeight()-val.LastBondingHeight() < 2*st.params.CommitteeSize {
		// Bonding period
		return false
	}

	if val.UnbondingHeight() > 0 {
		// we have Unbonded
		return false
	}

	ok, proof := st.sortition.EvaluateSortition(st.lastInfo.BlockHash(), st.signer, val.Stake())
	if ok {
		trx := tx.NewSortitionTx(st.lastInfo.BlockHash(), val.Sequence()+1, val.Address(), proof)
		st.signer.SignMsg(trx)

		err := st.txPool.AppendTxAndBroadcast(trx)
		if err == nil {
			st.logger.Debug("Sortition transaction broadcasted", "address", st.signer.Address(), "stake", val.Stake(), "tx", trx)
			return true
		}
		st.logger.Error("Our sortition transaction is invalid. Why?", "address", st.signer.Address(), "stake", val.Stake(), "tx", trx, "err", err)
	}

	return false
}

func (st *state) Fingerprint() string {
	return fmt.Sprintf("{#%d ⌘ %v 🕣 %v}",
		st.lastInfo.BlockHeight(),
		st.lastInfo.BlockHash().Fingerprint(),
		st.lastInfo.BlockTime().Format("15.04.05"))
}

func (st *state) commitSandbox(sb *sandbox.Concrete, round int) {
	joined := make([]*validator.Validator, 0)
	sb.IterateValidators(func(vs *sandbox.ValidatorStatus) {
		if vs.JoinedCommittee {
			st.logger.Info("New validator joined", "address", vs.Validator.Address(), "stake", vs.Validator.Stake())

			joined = append(joined, &vs.Validator)
		}
	})

	if err := st.committee.Update(round, joined); err != nil {
		//
		// We should panic here before updating the state
		//
		logger.Panic("An error occurred", "err", err)
	}

	sb.IterateAccounts(func(as *sandbox.AccountStatus) {
		if as.Updated {
			st.store.UpdateAccount(&as.Account)
		}
	})

	sb.IterateValidators(func(vs *sandbox.ValidatorStatus) {
		if vs.Updated {
			st.store.UpdateValidator(&vs.Validator)
		}
	})
}

func (st *state) validateBlockTime(t time.Time) error {
	if t.Second()%st.params.BlockTimeInSecond != 0 {
		return errors.Errorf(errors.ErrInvalidBlock, "block time (%s) is not rounded", t.String())
	}
	if t.Before(st.lastInfo.BlockTime()) {
		return errors.Errorf(errors.ErrInvalidBlock, "block time (%s) is before the last block time", t.String())
	}
	if t.Equal(st.lastInfo.BlockTime()) {
		return errors.Errorf(errors.ErrInvalidBlock, "block time (%s) is same as the last block time", t.String())
	}
	proposeTime := st.proposeNextBlockTime()
	threshold := st.params.BlockTime()
	if t.Before(proposeTime.Add(-threshold)) {
		return errors.Errorf(errors.ErrInvalidBlock, "block time (%s) is less than threshold (%s)", t.String(), proposeTime.String())
	}
	if t.After(proposeTime.Add(threshold)) {
		return errors.Errorf(errors.ErrInvalidBlock, "block time (%s) is more than threshold (%s)", t.String(), proposeTime.String())
	}

	return nil
}

func (st *state) TotalStake() int64 {
	st.lk.Lock()
	defer st.lk.Unlock()

	return st.totalStake()
}

func (st *state) CommitteeStake() int64 {
	st.lk.Lock()
	defer st.lk.Unlock()

	return st.committeeStake()
}

func (st *state) PoolStake() int64 {
	st.lk.Lock()
	defer st.lk.Unlock()

	return st.poolStake()
}

// TODO: Improve performance of these calculations by two local variables: committeeStake and poolStake
func (st *state) totalStake() int64 {
	totalStake := int64(0)
	st.store.IterateValidators(func(val *validator.Validator) bool {
		totalStake += val.Stake()
		return false
	})
	return totalStake
}

func (st *state) committeeStake() int64 {
	return st.committee.TotalStake()
}

func (st *state) poolStake() int64 {
	poolStake := int64(0)
	st.store.IterateValidators(func(val *validator.Validator) bool {
		if !st.committee.Contains(val.Address()) {
			poolStake += val.Stake()
		}
		return false
	})
	return poolStake
}

func (st *state) proposeNextBlockTime() time.Time {
	timestamp := st.lastInfo.BlockTime().Add(st.params.BlockTime())

	now := util.Now()
	if now.After(timestamp.Add(1 * time.Second)) {
		st.logger.Debug("It looks the last block had delay", "delay", now.Sub(timestamp))
		timestamp = util.RoundNow(st.params.BlockTimeInSecond)
	}
	return timestamp
}

func (st *state) CommitteeValidators() []*validator.Validator {
	return st.committee.Validators()
}

func (st *state) IsInCommittee(addr crypto.Address) bool {
	return st.committee.Contains(addr)
}

func (st *state) Proposer(round int) *validator.Validator {
	return st.committee.Proposer(round)
}

func (st *state) IsProposer(addr crypto.Address, round int) bool {
	return st.committee.IsProposer(addr, round)
}

func (st *state) Transaction(id tx.ID) *tx.Tx {
	tx, err := st.store.Transaction(id)
	if err != nil {
		st.logger.Error("Transaction Search in local store failed", "trx", id, "err", err)
	}
	return tx
}

func (st *state) Block(height int) *block.Block {
	b, err := st.store.Block(height)
	if err != nil {
		st.logger.Trace("Error on retrieving block", "err", err)
	}
	return b
}

func (st *state) BlockHeight(hash crypto.Hash) int {
	h, err := st.store.BlockHeight(hash)
	if err != nil {
		st.logger.Trace("Error on retrieving block height", "err", err)
	}
	return h
}

func (st *state) Account(addr crypto.Address) *account.Account {
	acc, err := st.store.Account(addr)
	if err != nil {
		st.logger.Trace("Error on retrieving account", "err", err)
	}
	return acc
}

func (st *state) Validator(addr crypto.Address) *validator.Validator {
	val, err := st.store.Validator(addr)
	if err != nil {
		st.logger.Trace("Error on retrieving validator", "err", err)
	}
	return val
}

// ValidatorByNumber returns validator data based on validator number
func (st *state) ValidatorByNumber(n int) *validator.Validator {
	val, err := st.store.ValidatorByNumber(n)
	if err != nil {
		st.logger.Trace("Error on retrieving validator", "err", err)
	}
	return val
}
func (st *state) PendingTx(id tx.ID) *tx.Tx {
	return st.txPool.PendingTx(id)
}
func (st *state) AddPendingTx(trx *tx.Tx) error {
	return st.txPool.AppendTx(trx)
}
func (st *state) AddPendingTxAndBroadcast(trx *tx.Tx) error {
	return st.txPool.AppendTxAndBroadcast(trx)
}
