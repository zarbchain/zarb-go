package execution

import (
	"github.com/zarbchain/zarb-go/config"
	"github.com/zarbchain/zarb-go/errors"
	"github.com/zarbchain/zarb-go/execution/executor"
	"github.com/zarbchain/zarb-go/logger"
	"github.com/zarbchain/zarb-go/tx"
)

type Executor struct {
	config         *config.Config
	sendExecutor   *executor.SendExecutor
	accumulatedFee int64
	logger         *logger.Logger
}

func NewExecutor(conf *config.Config, sandbox executor.Sandbox) (*Executor, error) {
	exe := &Executor{
		config:       conf,
		sendExecutor: executor.NewSendExecutor(sandbox),
	}
	exe.logger = logger.NewLogger("executor", exe)
	return exe, nil
}

func (exe *Executor) Execute(trx *tx.Tx, isMintbaseTx bool) (*tx.Receipt, error) {
	if !isMintbaseTx {
		if trx.IsMintbaseTx() {
			return nil, errors.Errorf(errors.ErrInvalidTx, "Duplicated mintbase transaction")
		}
	}

	exe.accumulatedFee += trx.Fee()

	if trx.IsCallTx() {
		// Call executor
	}

	return exe.sendExecutor.Execute(trx)
}
