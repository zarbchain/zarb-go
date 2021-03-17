package txpool

import (
	"container/list"
	"fmt"
	"time"

	"github.com/sasha-s/go-deadlock"
	"github.com/zarbchain/zarb-go/execution"
	"github.com/zarbchain/zarb-go/libs/linkedmap"
	"github.com/zarbchain/zarb-go/logger"
	"github.com/zarbchain/zarb-go/sync/message"
	"github.com/zarbchain/zarb-go/tx"
)

type txPool struct {
	lk deadlock.RWMutex

	config      *Config
	checker     *execution.Execution
	pendings    *linkedmap.LinkedMap
	appendTxCh  chan *tx.Tx
	broadcastCh chan *message.Message
	logger      *logger.Logger
}

func NewTxPool(
	conf *Config,
	broadcastCh chan *message.Message) (TxPool, error) {
	pool := &txPool{
		config:      conf,
		pendings:    linkedmap.NewLinkedMap(conf.MaxSize),
		broadcastCh: broadcastCh,
	}

	pool.logger = logger.NewLogger("_pool", pool)
	return pool, nil
}

func (pool *txPool) SetChecker(checker *execution.Execution) {
	pool.checker = checker
}
func (pool *txPool) AppendTx(trx *tx.Tx) error {
	pool.lk.Lock()
	defer pool.lk.Unlock()

	if err := pool.appendTx(trx); err != nil {
		return err
	}

	if pool.appendTxCh != nil {
		pool.appendTxCh <- trx
	}

	return nil
}

func (pool *txPool) AppendTxAndBroadcast(trx *tx.Tx) error {
	pool.lk.Lock()
	defer pool.lk.Unlock()

	if err := pool.appendTx(trx); err != nil {
		return err
	}

	go func(_trx *tx.Tx) {
		msg := message.NewTransactionsMessage([]*tx.Tx{_trx})
		pool.broadcastCh <- msg
	}(trx)

	return nil
}

func (pool *txPool) appendTx(trx *tx.Tx) error {
	if pool.pendings.Has(trx.ID()) {
		pool.logger.Trace("Transaction is already in pool.", "id", trx.ID())
		return nil
	}

	if err := pool.checkTx(trx); err != nil {
		return err
	}

	pool.pendings.PushBack(trx.ID(), trx)
	pool.logger.Trace("Transaction appended into pool.", "id", trx.ID())

	return nil
}

func (pool *txPool) checkTx(trx *tx.Tx) error {
	if err := pool.checker.Execute(trx); err != nil {
		pool.logger.Debug("Invalid transaction", "tx", trx, "err", err)
		return err
	}
	return nil
}

func (pool *txPool) RemoveTx(id tx.ID) {
	pool.lk.Lock()
	defer pool.lk.Unlock()

	pool.pendings.Remove(id)
}

// QueryTx returns immediately a transaction  if we have, otherwise nil
func (pool *txPool) PendingTx(id tx.ID) *tx.Tx {
	pool.lk.Lock()
	defer pool.lk.Unlock()

	val, found := pool.pendings.Get(id)
	if found {
		trx := val.(*tx.Tx)
		return trx
	}

	return nil
}

// QueryTx returns immediately a transaction  if we have,
// it queries from other nodes
func (pool *txPool) QueryTx(id tx.ID) *tx.Tx {
	trx := pool.PendingTx(id)
	if trx != nil {
		return trx
	}

	pool.logger.Debug("Query transaction from nodes", "id", id)

	msg := message.NewOpaqueQueryTransactionsMessage([]tx.ID{id})
	pool.broadcastCh <- msg

	defer func() {
		if pool.appendTxCh != nil {
			close(pool.appendTxCh)
			pool.appendTxCh = nil
		}
	}()

	pool.lk.Lock()
	pool.appendTxCh = make(chan *tx.Tx, 100)
	pool.lk.Unlock()

	timeout := time.NewTimer(pool.config.WaitingTimeout)

	for {
		select {
		case <-timeout.C:
			pool.logger.Warn("no transaction received", "id", id, "timeout", pool.config.WaitingTimeout)
			return nil
		case trx := <-pool.appendTxCh:
			pool.logger.Debug("Transaction received", "id", id)
			if trx.ID().EqualsTo(id) {
				return trx
			}
		}
	}
}

func (pool *txPool) AllTransactions() []*tx.Tx {
	pool.lk.RLock()
	defer pool.lk.RUnlock()

	trxs := make([]*tx.Tx, 0, pool.pendings.Size())
	for e := pool.pendings.FirstElement(); e != nil; e = e.Next() {
		trx := e.Value.(*linkedmap.Pair).Second.(*tx.Tx)
		trxs = append(trxs, trx)
	}

	return trxs
}

func (pool *txPool) HasTx(id tx.ID) bool {
	pool.lk.RLock()
	defer pool.lk.RUnlock()

	return pool.pendings.Has(id)
}

func (pool *txPool) Size() int {
	pool.lk.RLock()
	defer pool.lk.RUnlock()

	return pool.pendings.Size()
}

func (pool *txPool) Recheck() {
	pool.lk.Lock()
	defer pool.lk.Unlock()

	pool.logger.Debug("Rechecking Transactions")

	pool.checker.Reset()

	var next *list.Element
	for e := pool.pendings.FirstElement(); e != nil; e = next {
		next = e.Next()
		trx := e.Value.(*linkedmap.Pair).Second.(*tx.Tx)

		if err := pool.checkTx(trx); err != nil {
			pool.logger.Debug("Invalid transaction after rechecking", "id", trx.ID())
			pool.pendings.Remove(trx.ID())
		}
	}
}

func (pool *txPool) Fingerprint() string {
	return fmt.Sprintf("{%v}", pool.pendings.Size())
}
