package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/tx"
)

func Test_WithdrawExecutor(t *testing.T) {
	setup(t)
	exe := NewWithdrawExecutor(true)

	addr, _, _ := crypto.GenerateTestKeyPair()
	stamp := crypto.GenerateTestHash()

	tSandbox.AppendStampAndUpdateHeight(100, stamp)

	t.Run("Should fail, Invalid validator", func(t *testing.T) {
		trx := tx.NewWithdrawTx(stamp, 1, addr, tAcc1.Address(), 1000, 1000, "invalid validator")
		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	t.Run("Should fail, Invalid sequence", func(t *testing.T) {
		trx := tx.NewWithdrawTx(stamp, tSandbox.Validator(tVal1.Address()).Sequence()+2, tVal1.Address(), tAcc1.Address(), 1000, 1000, "invalid sequence")
		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	t.Run("should fail, stack amount secceded", func(t *testing.T) {
		assert.Equal(t, int64(5000000000), tVal1.Stake())
		trx := tx.NewWithdrawTx(stamp, tSandbox.Validator(tVal1.Address()).Sequence()+1, tAcc1.Address(), tAcc1.Address(), 5000000000, 1000, "need to unbond first")
		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	t.Run("Should fail, hasn't unbonded yet", func(t *testing.T) {
		assert.Equal(t, 0, tVal1.UnbondingHeight())
		trx := tx.NewWithdrawTx(stamp, tSandbox.Validator(tVal1.Address()).Sequence()+1, tVal1.Address(), tAcc1.Address(), 1000, 1000, "need to unbond first")
		assert.Error(t, exe.Execute(trx, tSandbox))
	})
	t.Run("Should fail, hasn't passed unbonding interval", func(t *testing.T) {
		assert.Equal(t, 0, tVal1.UnbondingHeight())
		tVal1.UpdateUnbondingHeight(101)
		stamp1 := crypto.GenerateTestHash()
		tSandbox.AppendStampAndUpdateHeight(201, stamp1)
		assert.Equal(t, 101, tVal1.UnbondingHeight())

		trx := tx.NewWithdrawTx(stamp, tSandbox.Validator(tVal1.Address()).Sequence()+1, tVal1.Address(), tAcc1.Address(), 1000, 1000, "not passed unbonding interval")
		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	t.Run("Should pass", func(t *testing.T) {
		stamp1 := crypto.GenerateTestHash()
		tSandbox.AppendStampAndUpdateHeight(tVal1.UnbondingHeight()+tSandbox.UnbondInterval(), stamp1)
		assert.Equal(t, 101, tVal1.UnbondingHeight())

		trx := tx.NewWithdrawTx(stamp, tSandbox.Validator(tVal1.Address()).Sequence()+1, tVal1.Address(), tAcc1.Address(), 4999999000, 1000, "should be able to empty stack")

		assert.NoError(t, exe.Execute(trx, tSandbox))
		assert.Zero(t, tVal1.Stake())
	})

	t.Run("Should fail, can't withdraw empty stack", func(t *testing.T) {
		assert.Zero(t, tVal1.Stake())
		trx := tx.NewWithdrawTx(stamp, tSandbox.Validator(tVal1.Address()).Sequence()+1, tVal1.Address(), tAcc1.Address(), 4999999000, 1000, "should be able to empty stack")
		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	assert.Equal(t, tSandbox.Validator(tVal1.Address()).Stake(), int64(0))
	assert.Equal(t, tSandbox.Validator(tVal1.Address()).Power(), int64(0)) //it shouldn't return 1 but it does
	assert.Equal(t, tSandbox.Account(tAcc1.Address()).Balance(), int64(14999999000))
	assert.Equal(t, tSandbox.Validator(tVal1.Address()).UnbondingHeight(), 101)
	assert.Equal(t, exe.Fee(), int64(1000))

	checkTotalCoin(t, 1000)
}
