package state

import (
	"fmt"
	"testing"
	"time"

	"github.com/zarbchain/zarb-go/validator"

	"github.com/stretchr/testify/assert"
	"github.com/zarbchain/zarb-go/block"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/tx"
	"github.com/zarbchain/zarb-go/util"
)

func TestTransactionLost(t *testing.T) {
	setup(t)

	b1, _ := tState1.ProposeBlock(0)
	assert.NoError(t, tState2.ValidateBlock(*b1))

	b2, _ := tState1.ProposeBlock(0)
	tCommonTxPool.Txs = make([]*tx.Tx, 0)
	assert.Error(t, tState2.ValidateBlock(*b2))
}

func TestCommitValidation(t *testing.T) {
	setup(t)

	b1, c1 := makeBlockAndCommit(t, 0, tValSigner1, tValSigner2, tValSigner4)
	applyBlockAndCommitForAllStates(t, b1, c1)

	val5, key5 := validator.GenerateTestValidator(4)
	tState1.store.UpdateValidator(val5)
	tState2.store.UpdateValidator(val5)

	b2, _ := tState2.ProposeBlock(0)

	invBlockHash := crypto.GenerateTestHash()
	round := 0
	valSig1 := tValSigner1.Sign(block.CommitSignBytes(b2.Hash(), round))
	valSig2 := tValSigner2.Sign(block.CommitSignBytes(b2.Hash(), round))
	valSig3 := tValSigner3.Sign(block.CommitSignBytes(b2.Hash(), round))
	valSig4 := tValSigner4.Sign(block.CommitSignBytes(b2.Hash(), round))
	invSig1 := tValSigner1.Sign(block.CommitSignBytes(invBlockHash, round))
	invSig2 := tValSigner2.Sign(block.CommitSignBytes(invBlockHash, round))
	invSig3 := tValSigner3.Sign(block.CommitSignBytes(invBlockHash, round))
	invSig5 := key5.Sign(block.CommitSignBytes(b2.Hash(), round))

	validSig := crypto.Aggregate([]*crypto.Signature{valSig1, valSig2, valSig3})
	invalidSig := crypto.Aggregate([]*crypto.Signature{invSig1, invSig2, invSig3})

	t.Run("Invalid blockhahs, should return error", func(t *testing.T) {
		c := block.NewCommit(invBlockHash, 0, []int{0, 1, 2}, []int{3}, validSig)

		assert.Error(t, tState1.ApplyBlock(2, *b2, *c))
	})

	t.Run("Invalid signature, should return error", func(t *testing.T) {
		invSig := tValSigner1.Sign([]byte("abc"))
		c := block.NewCommit(b2.Hash(), 0, []int{0, 1, 2}, []int{3}, *invSig)

		assert.Error(t, tState1.ApplyBlock(2, *b2, *c))
	})

	t.Run("Invalid signer, should return error", func(t *testing.T) {
		c := block.NewCommit(b2.Hash(), 0, []int{0, 1, 4}, []int{3}, validSig)
		assert.Error(t, tState1.ApplyBlock(2, *b2, *c))

		c = block.NewCommit(b2.Hash(), 0, []int{0, 1, 3}, []int{4}, validSig)
		assert.Error(t, tState1.ApplyBlock(2, *b2, *c))
	})

	t.Run("Unexpected signature", func(t *testing.T) {
		sig1 := crypto.Aggregate([]*crypto.Signature{valSig1, valSig2, invSig3, valSig4})
		c1 := block.NewCommit(b2.Hash(), 0, []int{0, 1, 2, 3}, []int{}, sig1)
		assert.Error(t, tState1.ApplyBlock(2, *b2, *c1))

		sig2 := crypto.Aggregate([]*crypto.Signature{valSig1, valSig2, valSig3, invSig5})
		c2 := block.NewCommit(b2.Hash(), 0, []int{0, 1, 2, 4}, []int{}, sig2)
		assert.Error(t, tState1.ApplyBlock(2, *b2, *c2))
	})

	t.Run("duplicated or missed number, should return error", func(t *testing.T) {
		c := block.NewCommit(b2.Hash(), 0, []int{0, 1, 2}, []int{2}, validSig)
		assert.Error(t, tState1.ApplyBlock(2, *b2, *c))

		c = block.NewCommit(b2.Hash(), 0, []int{0, 1, 2}, []int{}, validSig)
		assert.Error(t, tState1.ApplyBlock(2, *b2, *c))
	})

	t.Run("unexpected block hash", func(t *testing.T) {
		c := block.NewCommit(invBlockHash, 0, []int{0, 1, 2}, []int{3}, invalidSig)
		assert.Error(t, tState1.ApplyBlock(2, *b2, *c))

	})

	t.Run("Invalid round", func(t *testing.T) {
		c := block.NewCommit(b2.Hash(), 1, []int{0, 1, 2}, []int{3}, validSig)
		assert.Error(t, tState1.ApplyBlock(2, *b2, *c))
	})

	t.Run("Doesn't have 2/3 majority, should return no error", func(t *testing.T) {
		sig := crypto.Aggregate([]*crypto.Signature{valSig1, valSig2})

		c := block.NewCommit(b2.Hash(), 0, []int{0, 1}, []int{2, 3}, sig)
		assert.Error(t, tState1.ApplyBlock(2, *b2, *c))
	})

	t.Run("Update last commit- Not in the set, should return no error", func(t *testing.T) {
		sig := crypto.Aggregate([]*crypto.Signature{valSig1, valSig2, valSig3, invSig5})

		c := block.NewCommit(b2.Hash(), 0, []int{0, 1, 2, 4}, []int{2}, sig)
		assert.Error(t, tState1.ApplyBlock(2, *b2, *c))
	})

	t.Run("Valid signature, should return no error", func(t *testing.T) {
		c := block.NewCommit(b2.Hash(), 0, []int{0, 1, 2}, []int{3}, validSig)
		assert.NoError(t, tState1.ApplyBlock(2, *b2, *c))
	})

	t.Run("Update last commit- Invalid signer", func(t *testing.T) {
		sig := crypto.Aggregate([]*crypto.Signature{valSig1, valSig2, valSig3, invSig5})

		c := block.NewCommit(b2.Hash(), 0, []int{0, 1, 2, 4}, []int{}, sig)
		assert.Error(t, tState1.UpdateLastCommit(c))
	})

	t.Run("Update last commit- valid signature, should return no error", func(t *testing.T) {
		sig := crypto.Aggregate([]*crypto.Signature{valSig1, valSig2, valSig4})

		c := block.NewCommit(b2.Hash(), 0, []int{0, 1, 3}, []int{2}, sig)
		assert.NoError(t, tState1.UpdateLastCommit(c))
		// Commit didn't change
		assert.NotEqual(t, tState1.lastCommit.Hash(), c.Hash())
	})

	t.Run("Update last commit- Valid signature, should return no error", func(t *testing.T) {
		sig := crypto.Aggregate([]*crypto.Signature{valSig1, valSig2, valSig3, valSig4})

		c := block.NewCommit(b2.Hash(), 0, []int{0, 1, 2, 3}, []int{}, sig)
		assert.NoError(t, tState1.UpdateLastCommit(c))
		// Commit updated
		assert.Equal(t, tState1.lastCommit.Hash(), c.Hash())
	})

}

func TestUpdateBlockTime(t *testing.T) {
	setup(t)

	// Maipulate last block time
	tState1.lastBlockTime = util.Now().Add(-6 * time.Second)
	b, _ := tState1.ProposeBlock(0)
	fmt.Println(b.Header().Time())
	assert.True(t, b.Header().Time().After(tState1.lastBlockTime))
	assert.Zero(t, b.Header().Time().Second()%10)
}

func TestBlockValidation(t *testing.T) {
	setup(t)

	b1, c1 := makeBlockAndCommit(t, 0, tValSigner1, tValSigner2, tValSigner3, tValSigner4)
	applyBlockAndCommitForAllStates(t, b1, c1)
	assert.False(t, tState1.lastBlockHash.EqualsTo(crypto.UndefHash))

	//
	// Version   			(SanityCheck)
	// UnixTime				(?)
	// LastBlockHash		(OK)
	// StateHash			(OK)
	// TxIDsHash			(?)
	// LastReceiptsHash		(OK)
	// LastCommitHash		(OK)
	// CommittersHash		(OK)
	// ProposerAddress		(OK) -> Check in ApplyBlock
	//
	invAdd, _, _ := crypto.GenerateTestKeyPair()
	invHash := crypto.GenerateTestHash()
	invCommit := block.GenerateTestCommit(tState1.lastBlockHash)
	trx := tState1.createSubsidyTx(0)
	ids := block.NewTxIDs()
	ids.Append(trx.ID())

	b := block.MakeBlock(util.Now(), ids, invHash, tState1.validatorSet.CommittersHash(), tState1.stateHash(), tState1.lastReceiptsHash, tState1.lastCommit, tState1.proposer)
	assert.Error(t, tState1.validateBlock(b))

	b = block.MakeBlock(util.Now(), ids, tState1.lastBlockHash, invHash, tState1.stateHash(), tState1.lastReceiptsHash, tState1.lastCommit, tState1.proposer)
	assert.Error(t, tState1.validateBlock(b))

	b = block.MakeBlock(util.Now(), ids, tState1.lastBlockHash, tState1.validatorSet.CommittersHash(), invHash, tState1.lastReceiptsHash, tState1.lastCommit, tState1.proposer)
	assert.Error(t, tState1.validateBlock(b))

	b = block.MakeBlock(util.Now(), ids, tState1.lastBlockHash, tState1.validatorSet.CommittersHash(), tState1.stateHash(), invHash, tState1.lastCommit, tState1.proposer)
	assert.Error(t, tState1.validateBlock(b))

	b = block.MakeBlock(util.Now(), ids, tState1.lastBlockHash, tState1.validatorSet.CommittersHash(), tState1.stateHash(), tState1.lastReceiptsHash, invCommit, tState1.proposer)
	assert.Error(t, tState1.validateBlock(b))

	b = block.MakeBlock(util.Now(), ids, tState1.lastBlockHash, tState1.validatorSet.CommittersHash(), tState1.stateHash(), tState1.lastReceiptsHash, tState1.lastCommit, invAdd)
	assert.NoError(t, tState1.validateBlock(b))
	c := makeCommitAndSign(t, b.Hash(), 1, tValSigner1, tValSigner2, tValSigner3, tValSigner4)
	assert.Error(t, tState1.ApplyBlock(2, b, c))

	b = block.MakeBlock(util.Now(), ids, tState1.lastBlockHash, tState1.validatorSet.CommittersHash(), tState1.stateHash(), tState1.lastReceiptsHash, tState1.lastCommit, tState1.proposer)
	assert.NoError(t, tState1.validateBlock(b))
}
