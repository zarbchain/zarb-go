package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zarbchain/zarb-go/account"
	"github.com/zarbchain/zarb-go/config"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/genesis"
	"github.com/zarbchain/zarb-go/param"
	"github.com/zarbchain/zarb-go/util"
	"github.com/zarbchain/zarb-go/validator"
)

func TestRunningNode(t *testing.T) {
	_, pb, pv := crypto.RandomKeyPair()
	acc := account.NewAccount(crypto.TreasuryAddress, 0)
	acc.AddToBalance(21 * 1e14)
	val := validator.NewValidator(pb, 0)
	gen := genesis.MakeGenesis(util.Now(), []*account.Account{acc}, []*validator.Validator{val}, param.DefaultParams())
	conf := config.DefaultConfig()
	conf.Store.Path = util.TempDirPath()
	conf.Network.NodeKeyFile = util.TempFilePath()

	signer := crypto.NewSigner(pv)
	n, err := NewNode(gen, conf, signer)

	require.NoError(t, err)
	assert.Equal(t, n.state.LastBlockHash(), crypto.UndefHash)

	err = n.Start()
	require.NoError(t, err)
	n.Stop()
}
