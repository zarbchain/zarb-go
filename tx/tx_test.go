package tx

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/sortition"
	"github.com/zarbchain/zarb-go/tx/payload"
)

func TestJSONMarshaling(t *testing.T) {
	tx, _ := GenerateTestSendTx()
	_, err := tx.MarshalJSON()
	require.NoError(t, err)
}

func TestSendEncodingTx(t *testing.T) {
	tx, _ := GenerateTestSendTx()
	bz, err := tx.Encode()
	require.NoError(t, err)
	var tx2 Tx
	require.NoError(t, tx2.Decode(bz))
	require.Equal(t, tx.ID(), tx2.ID())
}

func TestBondEncodingTx(t *testing.T) {
	tx, _ := GenerateTestBondTx()
	bz, err := tx.MarshalCBOR()
	require.NoError(t, err)
	var tx2 Tx
	require.NoError(t, tx2.UnmarshalCBOR(bz))
	require.Equal(t, tx.ID(), tx2.ID())
}

func TestEncodingTxNoSig(t *testing.T) {
	tx, _ := GenerateTestSendTx()
	tx.data.Signature = nil
	tx.data.PublicKey = nil
	bz, err := tx.MarshalCBOR()
	require.NoError(t, err)
	var tx2 Tx
	require.NoError(t, tx2.UnmarshalCBOR(bz))
	require.Equal(t, tx.ID(), tx2.ID())
}

func TestTxSanityCheck(t *testing.T) {
	t.Run("Invalid sequence", func(t *testing.T) {
		tx, signer := GenerateTestSendTx()
		tx.data.Sequence = -1
		signer.SignMsg(tx)
		assert.Error(t, tx.SanityCheck())
	})

	t.Run("Invalid payload type", func(t *testing.T) {
		tx, signer := GenerateTestSendTx()
		tx.data.Type = 2
		signer.SignMsg(tx)
		assert.Error(t, tx.SanityCheck())
	})

	t.Run("Transaction ID should be same for signed and unsigned transactions", func(t *testing.T) {
		tx, _ := GenerateTestSendTx()
		id1 := tx.ID()
		sb1 := tx.SignBytes()
		tx.data.PublicKey = nil
		tx.data.Signature = nil
		id2 := tx.ID()
		sb2 := tx.SignBytes()
		assert.Equal(t, id1, id2)
		assert.Equal(t, sb1, sb2)
	})
}

func TestSubsidyTx(t *testing.T) {
	a, pub, priv := crypto.GenerateTestKeyPair()
	t.Run("Invalid fee", func(t *testing.T) {
		trx := NewMintbaseTx(crypto.GenerateTestHash(), 88, a, 2500, "subsidy")
		assert.True(t, trx.IsMintbaseTx())
		trx.data.Fee = 1
		assert.Error(t, trx.SanityCheck())
	})

	t.Run("Has signature", func(t *testing.T) {
		trx := NewMintbaseTx(crypto.GenerateTestHash(), 88, a, 2500, "subsidy")
		sig := priv.Sign(trx.SignBytes())
		trx.SetSignature(sig)
		assert.Error(t, trx.SanityCheck())
	})

	t.Run("Has public key", func(t *testing.T) {
		trx := NewMintbaseTx(crypto.GenerateTestHash(), 88, a, 2500, "subsidy")
		trx.SetPublicKey(pub)
		assert.Error(t, trx.SanityCheck())
	})
}

func TestInvalidSignature(t *testing.T) {
	t.Run("Good", func(t *testing.T) {
		tx, _ := GenerateTestSendTx()
		assert.NoError(t, tx.SanityCheck())
	})

	t.Run("No signature", func(t *testing.T) {
		tx, _ := GenerateTestSendTx()
		tx.data.Signature = nil
		assert.Error(t, tx.SanityCheck())
	})

	t.Run("No public key", func(t *testing.T) {
		tx, _ := GenerateTestSendTx()
		tx.data.PublicKey = nil
		assert.Error(t, tx.SanityCheck())
	})

	_, pbInv, pvInv := crypto.GenerateTestKeyPair()
	t.Run("Invalid signature", func(t *testing.T) {
		tx, _ := GenerateTestSendTx()
		sig := pvInv.Sign(tx.SignBytes())
		tx.SetSignature(sig)
		assert.Error(t, tx.SanityCheck())
	})

	t.Run("Invalid public key", func(t *testing.T) {
		tx, _ := GenerateTestSendTx()
		tx.SetPublicKey(pbInv)
		assert.Error(t, tx.SanityCheck())

	})

	t.Run("Invalid sign Bytes", func(t *testing.T) {
		tx, signer := GenerateTestSendTx()
		var tx2 = new(Tx)
		tx2.data.Memo = "Hello"
		tx.SetSignature(signer.SignData(tx2.SignBytes()))
		assert.Error(t, tx.SanityCheck())
	})
}

func TestSendSanityCheck(t *testing.T) {
	invAddr, _, _ := crypto.GenerateTestKeyPair()
	t.Run("Ok", func(t *testing.T) {
		trx, _ := GenerateTestSendTx()
		assert.NoError(t, trx.SanityCheck())
	})

	t.Run("Invalid amount", func(t *testing.T) {
		trx, signer := GenerateTestSendTx()
		pld := trx.data.Payload.(*payload.SendPayload)
		pld.Amount = -1
		signer.SignMsg(trx)
		assert.Error(t, trx.SanityCheck())
	})

	t.Run("Invalid fee", func(t *testing.T) {
		trx, signer := GenerateTestSendTx()
		trx.data.Fee = 0
		signer.SignMsg(trx)
		assert.Error(t, trx.SanityCheck())
	})

	t.Run("Invalid sender", func(t *testing.T) {
		trx, signer := GenerateTestSendTx()
		pld := trx.data.Payload.(*payload.SendPayload)
		pld.Sender = invAddr
		signer.SignMsg(trx)
		assert.Error(t, trx.SanityCheck())
	})

	t.Run("Invalid receiver", func(t *testing.T) {
		trx, signer := GenerateTestSendTx()
		pld := trx.data.Payload.(*payload.SendPayload)
		pld.Receiver = crypto.TreasuryAddress
		signer.SignMsg(trx)
		assert.Error(t, trx.SanityCheck())
	})
}

func TestBondSanityCheck(t *testing.T) {
	invAddr, _, _ := crypto.GenerateTestKeyPair()
	t.Run("Ok", func(t *testing.T) {
		trx, _ := GenerateTestBondTx()
		assert.NoError(t, trx.SanityCheck())
	})

	t.Run("Invalid stake", func(t *testing.T) {
		trx, signer := GenerateTestBondTx()
		pld := trx.data.Payload.(*payload.BondPayload)
		pld.Stake = -1
		signer.SignMsg(trx)
		assert.Error(t, trx.SanityCheck())
	})

	t.Run("Invalid bonder", func(t *testing.T) {
		trx, signer := GenerateTestBondTx()
		pld := trx.data.Payload.(*payload.BondPayload)
		pld.Bonder = invAddr
		signer.SignMsg(trx)
		assert.Error(t, trx.SanityCheck())
	})
}

func TestSortitionSanityCheck(t *testing.T) {
	invAddr, _, _ := crypto.GenerateTestKeyPair()
	t.Run("Ok", func(t *testing.T) {
		trx, _ := GenerateTestSortitionTx()
		assert.NoError(t, trx.SanityCheck())
	})

	t.Run("Invalid address", func(t *testing.T) {
		trx, signer := GenerateTestSortitionTx()
		pld := trx.data.Payload.(*payload.SortitionPayload)
		pld.Address = invAddr
		signer.SignMsg(trx)
		assert.Error(t, trx.SanityCheck())
	})

	t.Run("Invalid fee", func(t *testing.T) {
		trx, signer := GenerateTestSortitionTx()
		trx.data.Fee = 1
		signer.SignMsg(trx)
		assert.Error(t, trx.SanityCheck())
	})

}

func TestSendDecodingAndHash(t *testing.T) {
	d, _ := hex.DecodeString("a9010102582008f7d9c21fdaa4a4147e60a0f3933c850b0c0d9af6b2a308c0a7b5639a7e49d603186e040a050106a301548dfaf698d3889b13251529ff971277305fbf1f440254bdd1540a13d82c38e5b4dfbd7b5b2bcab5fb5f290318640767746573742074781458603846ed5d519e51f6dd63e552ac410c531d5436c726475f6f8fb51c1133b07e32bd3bc4c674359546a1145cb1935a3c0621fc5329c6039707445e472a73d857d8eff832b971838b21c53baa090d90b02f6d2c5a1e9358e46f4f4955ff737c08991558309e6b99c60cf5ccb551efc793ec2bedd66070bde8fbddeb8305c5f670a21532304775637f9e622d5212f34c9479d12d11")
	s, _ := hex.DecodeString("a7010102582008f7d9c21fdaa4a4147e60a0f3933c850b0c0d9af6b2a308c0a7b5639a7e49d603186e040a050106a301548dfaf698d3889b13251529ff971277305fbf1f440254bdd1540a13d82c38e5b4dfbd7b5b2bcab5fb5f29031864076774657374207478")
	h, _ := crypto.HashFromString("38ea1ad335bbfd84641a34b6af3332810aef8e52da08897273f187fd6059c50a")
	var trx Tx
	err := trx.UnmarshalCBOR(d)
	assert.NoError(t, err)
	d2, _ := trx.MarshalCBOR()
	assert.Equal(t, d, d2)
	assert.Equal(t, trx.SignBytes(), s)
	assert.Equal(t, trx.ID(), h)
}

func TestBondDecodingAndHash(t *testing.T) {
	d, _ := hex.DecodeString("a90101025820401a78337d2715db2a69916bbedbbb0a44336915fa1e5938b5b9cf350c340bb603186e0400050206a30154ebd6d0afdbe2ac5968224763f1c2dbb99fedda53025860c0ac961b453008e5cf521afab81fe303dc957de4b385bf472108e27b054f14e2da340f288928472b443a716eb795800cd8b885791827debda0a7909c741286cebd324db9546bab8af03830a9f91792389303b3746d76fc57148830184c3e0506031864076c7465737420626f6e642d74781458603b42c95589d73085882cf972c80b8c652de6e89ae4268a37a6c2d3206019bf9b706e048a7555e77f926895f6dccdb10f42efcaf0008006a0a15b8c58d704a9a0f277c66d7c6f6e8be1efb50b509c32b3b4364b2b7feb7eda106164c7ad18408a1558305a53c9e788896943587b594a2c2b930f2f9945abdf12e1c16fae8d235f74663a2f28d1b8fa442af0781f13f7de5a1c99")
	s, _ := hex.DecodeString("a70101025820401a78337d2715db2a69916bbedbbb0a44336915fa1e5938b5b9cf350c340bb603186e0400050206a30154ebd6d0afdbe2ac5968224763f1c2dbb99fedda53025860c0ac961b453008e5cf521afab81fe303dc957de4b385bf472108e27b054f14e2da340f288928472b443a716eb795800cd8b885791827debda0a7909c741286cebd324db9546bab8af03830a9f91792389303b3746d76fc57148830184c3e0506031864076c7465737420626f6e642d7478")
	h, _ := crypto.HashFromString("0beb43223b39e21e8c29afa70afa44bf7e6320c148e9c3cdd6feb4e00d6c460a")
	var trx Tx
	err := trx.UnmarshalCBOR(d)
	assert.NoError(t, err)
	d2, _ := trx.MarshalCBOR()
	assert.Equal(t, d, d2)
	assert.Equal(t, trx.SignBytes(), s)
	assert.Equal(t, trx.ID(), h)
}

func TestSortitionDecodingAndHash(t *testing.T) {
	d, _ := hex.DecodeString("a901010258205c89caaff437620b0f8c3b02db9e42cf2cd7668d1b9ea6f97cf0a8c15b4443f803186e0400050306a20154564404dd49c9f8ef137e33fde04b8733317c23c802583000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007717465737420736f72746974696f6e2d7478145860e0c7d9197d7e8dee6f8a279fe0b3c9a51069aa15ca513cca339fd92d4d2510145e70c604cfa263672001ba561e4a090816589b0bd35365399493f8180c0b06693f264fe892c26ccbc9c1d78de8a579d37122eb81c6f4792236ace58138666c1415583062a837ef943e068ba7c87b350e867b9200ceb821a72c122c59cf93f6e55cc74abb7110875de2e8b8bb4002923433448c")
	s, _ := hex.DecodeString("a701010258205c89caaff437620b0f8c3b02db9e42cf2cd7668d1b9ea6f97cf0a8c15b4443f803186e0400050306a20154564404dd49c9f8ef137e33fde04b8733317c23c802583000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007717465737420736f72746974696f6e2d7478")
	h, _ := crypto.HashFromString("656a4cb4ae6dc60e53196818e035147762b738500bc002a30a29820ac9d8f433")
	var trx Tx
	err := trx.UnmarshalCBOR(d)
	assert.NoError(t, err)
	d2, _ := trx.MarshalCBOR()
	assert.Equal(t, d, d2)
	assert.Equal(t, trx.SignBytes(), s)
	assert.Equal(t, trx.ID(), h)
}

func TestSendSignBytes(t *testing.T) {
	h := crypto.GenerateTestHash()
	signer := crypto.GenerateTestSigner()
	addr, _, _ := crypto.GenerateTestKeyPair()

	trx1 := NewSendTx(h, 1, signer.Address(), addr, 100, 10, "test send-tx")
	signer.SignMsg(trx1)

	trx2 := NewSendTx(h, 1, signer.Address(), addr, 100, 10, "test send-tx")
	trx3 := NewSendTx(h, 2, signer.Address(), addr, 100, 10, "test send-tx")

	assert.Equal(t, trx1.SignBytes(), trx2.SignBytes())
	assert.NotEqual(t, trx1.SignBytes(), trx3.SignBytes())
}

func TestBondSignBytes(t *testing.T) {
	h := crypto.GenerateTestHash()
	signer := crypto.GenerateTestSigner()
	_, pub, _ := crypto.GenerateTestKeyPair()

	trx1 := NewBondTx(h, 1, signer.Address(), pub, 100, 100, "test bond-tx")
	signer.SignMsg(trx1)

	trx2 := NewBondTx(h, 1, signer.Address(), pub, 100, 100, "test bond-tx")
	trx3 := NewBondTx(h, 2, signer.Address(), pub, 100, 100, "test bond-tx")

	assert.Equal(t, trx1.SignBytes(), trx2.SignBytes())
	assert.NotEqual(t, trx1.SignBytes(), trx3.SignBytes())
	assert.True(t, trx1.IsBondTx())
}

func TestUnbondSignBytes(t *testing.T) {
	h := crypto.GenerateTestHash()
	signer := crypto.GenerateTestSigner()

	trx1 := NewUnbondTx(h, 1, signer.Address(), "test unbond-tx")
	signer.SignMsg(trx1)

	trx2 := NewUnbondTx(h, 1, signer.Address(), "test unbond-tx")
	trx3 := NewUnbondTx(h, 2, signer.Address(), "test unbond-tx")

	assert.Equal(t, trx1.SignBytes(), trx2.SignBytes())
	assert.NotEqual(t, trx1.SignBytes(), trx3.SignBytes())
	assert.True(t, trx1.IsUnbondTx())

}
func TestWithdrawSignBytes(t *testing.T) {
	h := crypto.GenerateTestHash()
	signer := crypto.GenerateTestSigner()
	addr, _, _ := crypto.GenerateTestKeyPair()

	trx1 := NewWithdrawTx(h, 1, signer.Address(), addr, 1000, 1000, "test unbond-tx")
	signer.SignMsg(trx1)

	trx2 := NewWithdrawTx(h, 1, signer.Address(), addr, 1000, 1000, "test unbond-tx")
	trx3 := NewWithdrawTx(h, 2, signer.Address(), addr, 1000, 1000, "test unbond-tx")

	assert.Equal(t, trx1.SignBytes(), trx2.SignBytes())
	assert.NotEqual(t, trx1.SignBytes(), trx3.SignBytes())
	assert.True(t, trx1.IsWithdrawTx())

}

func TestSortitionSignBytes(t *testing.T) {
	h := crypto.GenerateTestHash()
	signer := crypto.GenerateTestSigner()
	proof := sortition.GenerateRandomProof()

	trx1 := NewSortitionTx(h, 1, signer.Address(), proof)
	signer.SignMsg(trx1)

	trx2 := NewSortitionTx(h, 1, signer.Address(), proof)
	trx3 := NewSortitionTx(h, 2, signer.Address(), proof)

	assert.Equal(t, trx1.SignBytes(), trx2.SignBytes())
	assert.NotEqual(t, trx1.SignBytes(), trx3.SignBytes())
	assert.True(t, trx1.IsSortitionTx())
}
