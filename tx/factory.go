package tx

import (
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/sortition"
	"github.com/zarbchain/zarb-go/tx/payload"
)

func NewMintbaseTx(stamp crypto.Hash, seq int, receiver crypto.Address, amount int64, memo string) *Tx {
	return NewSendTx(
		stamp,
		seq,
		crypto.TreasuryAddress,
		receiver,
		amount,
		0,
		memo)
}

func NewSendTx(stamp crypto.Hash,
	seq int,
	sender, receiver crypto.Address,
	amount, fee int64, memo string) *Tx {
	return &Tx{
		data: txData{
			Stamp:    stamp,
			Sequence: seq,
			Version:  1,
			Type:     payload.PayloadTypeSend,
			Payload: &payload.SendPayload{
				Sender:   sender,
				Receiver: receiver,
				Amount:   amount,
			},
			Fee:  fee,
			Memo: memo,
		},
	}
}

func NewBondTx(stamp crypto.Hash,
	seq int,
	bonder crypto.Address,
	val crypto.PublicKey,
	stake, fee int64, memo string) *Tx {
	return &Tx{
		data: txData{
			Stamp:    stamp,
			Sequence: seq,
			Version:  1,
			Type:     payload.PayloadTypeBond,
			Payload: &payload.BondPayload{
				Bonder:    bonder,
				Validator: val,
				Stake:     stake,
			},
			Fee:  fee,
			Memo: memo,
		},
	}
}

func NewUnbondTx(stamp crypto.Hash,
	seq int,
	val crypto.Address,
	memo string) *Tx {
	return &Tx{
		data: txData{
			Stamp:    stamp,
			Sequence: seq,
			Version:  1,
			Type:     payload.PayloadTypeUnbond,
			Payload: &payload.UnbondPayload{
				Validator: val,
			},
			Fee:  0,
			Memo: memo,
		},
	}
}

func NewWithdrawTx(stamp crypto.Hash,
	seq int,
	val crypto.Address,
	acc crypto.Address,
	amount, fee int64,
	memo string) *Tx {
	return &Tx{
		data: txData{
			Stamp:    stamp,
			Sequence: seq,
			Version:  1,
			Type:     payload.PayloadTypeWithdraw,
			Payload: &payload.WithdrawPayload{
				From:   val,
				To:     acc,
				Amount: amount,
			},
			Fee:  fee,
			Memo: memo,
		},
	}
}

func NewSortitionTx(stamp crypto.Hash,
	seq int,
	addr crypto.Address,
	proof sortition.Proof) *Tx {
	return &Tx{
		data: txData{
			Stamp:    stamp,
			Sequence: seq,
			Version:  1,
			Type:     payload.PayloadTypeSortition,
			Payload: &payload.SortitionPayload{
				Address: addr,
				Proof:   proof,
			},
			Fee: 0,
		},
	}
}
