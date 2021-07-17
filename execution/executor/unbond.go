package executor

import (
	"github.com/zarbchain/zarb-go/errors"
	"github.com/zarbchain/zarb-go/sandbox"
	"github.com/zarbchain/zarb-go/tx"
	"github.com/zarbchain/zarb-go/tx/payload"
)

type UnbondExecutor struct {
	strict bool
}

func NewUnbondExecutor(strict bool) *UnbondExecutor {
	return &UnbondExecutor{strict: strict}
}

func (e *UnbondExecutor) Execute(trx *tx.Tx, sb sandbox.Sandbox) error {
	pld := trx.Payload().(*payload.UnbondPayload)

	val := sb.Validator(pld.Signer())

	if val == nil {
		//if couldn't retrieve the validator then cann't unbond it
		return errors.Errorf(errors.ErrInvalidTx, "Unable to retrieve validator assoiciated with this key")
	}
	if e.strict && sb.IsInCommittee(pld.Validator) {
		return errors.Errorf(errors.ErrInvalidTx, "Validator is in committee right now please wait")
	}
	if val.Sequence()+1 != trx.Sequence() {
		return errors.Errorf(errors.ErrInvalidTx, "Invalid sequence. Expected: %v, got: %v", val.Sequence()+1, trx.Sequence())
	}
	if val.UnbondingHeight() > 0 {
		return errors.Errorf(errors.ErrInvalidTx, "you already have unbonded at Height %v", val.UnbondingHeight())
	}

	val.IncSequence()
	val.UpdateUnbondingHeight(sb.CurrentHeight())
	sb.UpdateValidator(val)

	return nil
}

//Fee will return unbond execution fee
func (e *UnbondExecutor) Fee() int64 {
	return 0
}
