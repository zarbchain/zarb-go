package genesis

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.com/zarb-chain/zarb-go/account"
	"gitlab.com/zarb-chain/zarb-go/crypto"
	"gitlab.com/zarb-chain/zarb-go/validator"
)

func TestMarshaling(t *testing.T) {
	pb, _ := crypto.GenerateRandomKey()
	addr := pb.Address()
	acc := account.NewAccount(addr)
	val := validator.NewValidator(pb, 1)
	gen1 := MakeGenesis("test", time.Now().Truncate(0), []*account.Account{acc}, []*validator.Validator{val})
	gen2 := new(Genesis)

	bz, err := json.Marshal(gen1)
	require.NoError(t, err)
	err = json.Unmarshal(bz, gen2)
	require.NoError(t, err)
	require.Equal(t, gen1.Accounts(), gen2.Accounts())
	require.True(t, gen1.Validators()[0].PublicKey().EqualsTo(gen2.Validators()[0].PublicKey()))
}
