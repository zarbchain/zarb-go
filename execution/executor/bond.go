package executor

import (
	"github.com/zarbchain/zarb-go/errors"
	"github.com/zarbchain/zarb-go/tx"
	"github.com/zarbchain/zarb-go/tx/payload"
	"github.com/zarbchain/zarb-go/validator"
)

type BondExecutor struct {
	sandbox Sandbox
}

func NewBondExecutor(sandbox Sandbox) *BondExecutor {
	return &BondExecutor{sandbox}
}

func (e *BondExecutor) Execute(trx *tx.Tx) error {
	pld := trx.Payload().(*payload.BondPayload)

	bonderAcc := e.sandbox.Account(pld.Bonder)
	if bonderAcc == nil {
		return errors.Errorf(errors.ErrInvalidTx, "Unable to retrieve bonder account")
	}
	bondVal := e.sandbox.Validator(pld.Validator.Address())
	if bondVal == nil {
		bondVal = validator.NewValidator(pld.Validator, e.sandbox.CurrentHeight())
	}
	if bonderAcc.Sequence()+1 != trx.Sequence() {
		return errors.Errorf(errors.ErrInvalidTx, "Invalid sequence")
	}
	if bonderAcc.Balance() < pld.Stake+trx.Fee() {
		return errors.Errorf(errors.ErrInvalidTx, "Insufficient balance")
	}
	bonderAcc.IncSequence()
	bonderAcc.SubtractFromBalance(pld.Stake + trx.Fee())
	bondVal.AddToStake(pld.Stake)

	e.sandbox.UpdateAccount(bonderAcc)
	e.sandbox.UpdateValidator(bondVal)

	return nil
}