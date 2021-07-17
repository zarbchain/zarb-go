package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/tx"
)

func TestExecuteUnbondTx(t *testing.T) {
	setup(t)
	exe := NewUnbondExecutor(true)

	addr, _, _ := crypto.GenerateTestKeyPair()

	stamp := crypto.GenerateTestHash()
	tSandbox.AppendStampAndUpdateHeight(100, stamp)

	t.Run("Should fail, Invalid validator", func(t *testing.T) {
		trx := tx.NewUnbondTx(stamp, 1, addr, "invalid validator")
		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	t.Run("Should fail, Invalid sequence", func(t *testing.T) {
		trx := tx.NewUnbondTx(stamp, tSandbox.Validator(tVal1.Address()).Sequence()+2, tVal1.Address(), "invalid sequence")

		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	t.Run("Should fail, Inside committee", func(t *testing.T) {
		tSandbox.InCommittee = true
		trx := tx.NewUnbondTx(stamp, tSandbox.Validator(tVal1.Address()).Sequence()+1, tVal1.Address(), "inside committee")

		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	t.Run("Ok", func(t *testing.T) {
		tSandbox.InCommittee = false
		trx := tx.NewUnbondTx(stamp, tSandbox.Validator(tVal1.Address()).Sequence()+1, tVal1.Address(), "Ok")

		assert.NoError(t, exe.Execute(trx, tSandbox))

		// Replay
		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	t.Run("Should fail, Cann't unbond if unbonded already", func(t *testing.T) {
		tSandbox.InCommittee = false
		trx := tx.NewUnbondTx(stamp, tSandbox.Validator(tVal1.Address()).Sequence()+1, tVal1.Address(), "Ok")

		assert.Error(t, exe.Execute(trx, tSandbox))
	})
	assert.Equal(t, tSandbox.Validator(tVal1.Address()).Stake(), int64(5000000000))
	assert.Equal(t, tSandbox.Validator(tVal1.Address()).Power(), int64(0))
	assert.Equal(t, tSandbox.Validator(tVal1.Address()).UnbondingHeight(), 101)
	assert.Equal(t, exe.Fee(), int64(0))

	checkTotalCoin(t, 0)
}

func TestUnbondNonStrictMode(t *testing.T) {
	setup(t)
	exe1 := NewBondExecutor(false)

	tSandbox.InCommittee = true
	stamp := crypto.GenerateTestHash()
	tSandbox.AppendStampAndUpdateHeight(100, stamp)
	bonder := tAcc1.Address()
	_, pub, _ := crypto.GenerateTestKeyPair()

	mintbase1 := tx.NewBondTx(stamp, tSandbox.AccSeq(bonder)+1, bonder, pub, 1000, 1000, "")
	mintbase2 := tx.NewBondTx(stamp, tSandbox.AccSeq(bonder)+1, bonder, pub, 1000, 1000, "")

	assert.NoError(t, exe1.Execute(mintbase1, tSandbox))
	assert.Error(t, exe1.Execute(mintbase2, tSandbox)) // Invalid sequence
}
