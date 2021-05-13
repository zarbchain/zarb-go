package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/tx"
)

func TestExecuteBondTx(t *testing.T) {
	setup(t)
	exe := NewBondExecutor(true)

	bonder := tAcc1.Address()
	addr, pub, _ := crypto.GenerateTestKeyPair()
	stamp := crypto.GenerateTestHash()
	tSandbox.AppendStampAndUpdateHeight(100, stamp)

	t.Run("Should fail, Invalid bonder", func(t *testing.T) {
		trx := tx.NewBondTx(stamp, 1, addr, pub, 1000, 1000, "invalid bonder")
		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	t.Run("Should fail, Invalid sequence", func(t *testing.T) {
		trx := tx.NewBondTx(stamp, tSandbox.AccSeq(bonder)+2, bonder, pub, 1000, 1000, "invalid sequence")

		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	t.Run("Should fail, Insufficient balance", func(t *testing.T) {
		trx := tx.NewBondTx(stamp, tSandbox.AccSeq(bonder)+1, bonder, pub, 10000000000, 10000000, "insufficient balance")

		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	t.Run("Should fail, Inside committee", func(t *testing.T) {
		tSandbox.InCommittee = true
		trx := tx.NewBondTx(stamp, tSandbox.AccSeq(bonder)+1, bonder, pub, 1000, 1000, "inside committee")

		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	t.Run("Ok", func(t *testing.T) {
		tSandbox.InCommittee = false
		trx := tx.NewBondTx(stamp, tSandbox.AccSeq(bonder)+1, bonder, pub, 1000, 1000, "ok")

		assert.NoError(t, exe.Execute(trx, tSandbox))

		// Replay
		assert.Error(t, exe.Execute(trx, tSandbox))
	})
	assert.Equal(t, tSandbox.Account(bonder).Balance(), int64(10000000000-2000))
	assert.Equal(t, tSandbox.Validator(addr).Stake(), int64(1000))
	assert.Equal(t, tSandbox.Validator(addr).BondingHeight(), 101)
	assert.Equal(t, exe.Fee(), int64(1000))

	checkTotalCoin(t, 1000)
}

func TestBondNonStrictMode(t *testing.T) {
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
