package grpc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zarbchain/zarb-go/account"
	"github.com/zarbchain/zarb-go/util"
	zarb "github.com/zarbchain/zarb-go/www/grpc/proto"
)

func TestGetAccount(t *testing.T) {
	conn, client := callServer(t)
	acc1, _ := account.GenerateTestAccount(util.RandInt(10000))
	t.Run("Should return nil for non existing account ", func(t *testing.T) {
		res, err := client.GetAccount(tCtx, &zarb.AccountRequest{Address: acc1.Address().String()})
		assert.Error(t, err)
		assert.Nil(t, res)
	})
	conn.Close()

	tMockState.Store.UpdateAccount(acc1)
	t.Run("Should return account details", func(t *testing.T) {
		res, err := client.GetAccount(tCtx, &zarb.AccountRequest{Address: acc1.Address().String()})
		fmt.Println("the vlaue in", res)
		assert.Error(t, err)
		assert.Nil(t, res)
	})
	conn.Close()

}
