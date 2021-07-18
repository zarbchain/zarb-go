package grpc

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/tx"
	"github.com/zarbchain/zarb-go/tx/payload"
	zarb "github.com/zarbchain/zarb-go/www/grpc/proto"
)

func TestGetTransaction(t *testing.T) {
	conn, client := callServer(t)
	tx1, _ := tx.GenerateTestSendTx()

	tMockState.Store.SaveTransaction(tx1)

	t.Run("Should return transaction", func(t *testing.T) {
		res, err := client.GetTransaction(tCtx, &zarb.TransactionRequest{Id: tx1.ID().String()})
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.NotEmpty(t, res.Tranaction)
		assert.Equal(t, tx1.ID().String(), res.Tranaction.Id)
		assert.Equal(t, tx1.Stamp().String(), res.Tranaction.Stamp)
		assert.Equal(t, tx1.Fee(), res.Tranaction.Fee)
		assert.Equal(t, tx1.Memo(), res.Tranaction.Memo)
		assert.Equal(t, int64(tx1.Sequence()), res.Tranaction.Sequence)
		assert.Equal(t, tx1.Signature().String(), res.Tranaction.Signature)
		assert.Equal(t, tx1.Payload().(*payload.SendPayload).Amount, res.Tranaction.Payload.(*zarb.TransactionInfo_Send).Send.Amount)
		assert.Equal(t, tx1.Payload().(*payload.SendPayload).Sender.String(), res.Tranaction.Payload.(*zarb.TransactionInfo_Send).Send.Sender)
		assert.Equal(t, tx1.Payload().(*payload.SendPayload).Receiver.String(), res.Tranaction.Payload.(*zarb.TransactionInfo_Send).Send.Receiver)
	})

	t.Run("Should return nil value because transcation id is invalid", func(t *testing.T) {
		res, err := client.GetTransaction(tCtx, &zarb.TransactionRequest{Id: "invalid_id"})
		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("Should return nil value because transcation doesn't exist", func(t *testing.T) {
		id := crypto.GenerateTestHash()
		res, err := client.GetTransaction(tCtx, &zarb.TransactionRequest{Id: id.String()})
		assert.Error(t, err)
		assert.Nil(t, res)
	})
	conn.Close()

}

func TestSendRawTransaction(t *testing.T) {
	conn, client := callServer(t)

	t.Run("Should fail, Non decodable hex", func(t *testing.T) {
		res, err := client.SendRawTransaction(tCtx, &zarb.SendRawTransactionRequest{Data: "None Decodable hex"})
		assert.Error(t, err)
		assert.Nil(t, res)
	})
	t.Run("Should fail, Non decodable data", func(t *testing.T) {
		res, err := client.SendRawTransaction(tCtx, &zarb.SendRawTransactionRequest{Data: hex.EncodeToString([]byte("None Decodable data"))})
		assert.Error(t, err)
		assert.Nil(t, res)
	})
	t.Run("Should fail, Non verifable trx", func(t *testing.T) {
		trx, _ := tx.GenerateTestSendTx()
		_, signer := tx.GenerateTestSendTx()
		trx.SetSignature(signer.SignData(trx.SignBytes()))
		data, _ := trx.Encode()
		res, err := client.SendRawTransaction(tCtx, &zarb.SendRawTransactionRequest{Data: hex.EncodeToString(data)})
		assert.Error(t, err)
		assert.Nil(t, res)
	})
	trx, _ := tx.GenerateTestSendTx()
	data, _ := trx.Encode()
	t.Run("Should pass", func(t *testing.T) {
		res, err := client.SendRawTransaction(tCtx, &zarb.SendRawTransactionRequest{Data: hex.EncodeToString(data)})
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})
	t.Run("Should fail, Not Broadcasted", func(t *testing.T) {
		res, err := client.SendRawTransaction(tCtx, &zarb.SendRawTransactionRequest{Data: hex.EncodeToString(data)})
		assert.Error(t, err)
		assert.Nil(t, res)
	})
	assert.NoError(t, conn.Close())
}
