package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/sortition"
	"github.com/zarbchain/zarb-go/tx"
)

func TestExecuteSortitionTx(t *testing.T) {
	setup(t)
	exe := NewSortitionExecutor(true)

	stamp40 := crypto.GenerateTestHash()
	tSandbox.AppendStampAndUpdateHeight(40, stamp40)
	proof1 := sortition.GenerateRandomProof()

	t.Run("Should fail, Bonding period", func(t *testing.T) {
		trx := tx.NewSortitionTx(stamp40, 1, tValSigner.Address(), proof1)

		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	stamp41 := crypto.GenerateTestHash()
	tSandbox.AppendStampAndUpdateHeight(41, stamp41)

	t.Run("Should fail, Invalid address", func(t *testing.T) {
		addr, _, _ := crypto.GenerateTestKeyPair()
		trx := tx.NewSortitionTx(stamp41, 1, addr, proof1)

		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	t.Run("Should fail, Invalid sequence", func(t *testing.T) {
		trx := tx.NewSortitionTx(stamp41, 2, tValSigner.Address(), proof1)

		assert.Error(t, exe.Execute(trx, tSandbox))
	})

	t.Run("Should be ok", func(t *testing.T) {
		trx := tx.NewSortitionTx(stamp41, 1, tValSigner.Address(), proof1)

		// Check if can't join to committee
		tSandbox.AcceptSortition = true
		tSandbox.WelcomeToCommittee = false
		assert.Error(t, exe.Execute(trx, tSandbox))

		// Check if proof is wrong
		tSandbox.AcceptSortition = false
		tSandbox.WelcomeToCommittee = true
		assert.Error(t, exe.Execute(trx, tSandbox))

		// Check if power is 0
		tSandbox.AcceptSortition = true
		tSandbox.WelcomeToCommittee = true
		tSandbox.Validator(tValSigner.Address()).UpdateLastBondingHeight(3)
		assert.Error(t, exe.Execute(trx, tSandbox))

		// Sounds good
		tSandbox.AcceptSortition = true
		tSandbox.WelcomeToCommittee = true
		tSandbox.Validator(tValSigner.Address()).UpdateLastBondingHeight(0)
		assert.NoError(t, exe.Execute(trx, tSandbox))

		// Check unbond state
		tSandbox.AcceptSortition = true
		tSandbox.WelcomeToCommittee = true
		tSandbox.Validator(tValSigner.Address()).UpdateUnbondingHeight(1)
		assert.Error(t, exe.Execute(trx, tSandbox))

		// replay
		assert.Error(t, exe.Execute(trx, tSandbox))

	})

	val := tSandbox.Validator(tValSigner.Address())
	assert.Equal(t, val.Sequence(), 1)
	assert.Equal(t, val.LastJoinedHeight(), 42)
	assert.Zero(t, exe.Fee())

	checkTotalCoin(t, 0)
}

func TestSortitionNonStrictMode(t *testing.T) {
	setup(t)
	exe1 := NewSortitionExecutor(false)

	stamp100 := crypto.GenerateTestHash()
	stamp101 := crypto.GenerateTestHash()
	tSandbox.AppendStampAndUpdateHeight(100, stamp100)
	tSandbox.AppendStampAndUpdateHeight(101, stamp101)
	proof1 := sortition.GenerateRandomProof()
	proof2 := sortition.GenerateRandomProof()

	tSandbox.AcceptSortition = true
	tSandbox.WelcomeToCommittee = false

	sortition1 := tx.NewSortitionTx(stamp100, 102, tValSigner.Address(), proof1)
	sortition2 := tx.NewSortitionTx(stamp101, 102, tValSigner.Address(), proof2)

	assert.NoError(t, exe1.Execute(sortition1, tSandbox))
	assert.NoError(t, exe1.Execute(sortition2, tSandbox))
}
