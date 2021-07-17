package validator

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/util"
)

func TestMarshaling(t *testing.T) {
	val1, _ := GenerateTestValidator(util.RandInt(1000))
	assert.Equal(t, val1.PublicKey().Address(), val1.Address())
	val1.AddToStake(1)
	val1.IncSequence()
	val1.UpdateLastJoinedHeight(100)

	bs, err := val1.Encode()
	require.NoError(t, err)
	val2 := new(Validator)
	err = val2.Decode(bs)
	require.NoError(t, err)
	assert.Equal(t, val1.Hash(), val2.Hash())

	val3 := new(Validator)
	err = val3.Decode(bs)
	require.NoError(t, err)
	assert.Equal(t, val2, val3)

	/// test json marshaing/unmarshaling
	js, err := json.Marshal(val1)
	require.NoError(t, err)
	fmt.Println(string(js))
	val4 := new(Validator)
	require.NoError(t, json.Unmarshal(js, val4))

	assert.Equal(t, val3, val4)

	/// should fail
	val5 := new(Validator)
	err = val5.Decode([]byte("asdfghjkl"))
	require.Error(t, err)
}

func TestMarshalingRawData(t *testing.T) {
	bs, _ := hex.DecodeString("a7015860dff46fbce5ae1ba4837de551206176c0a74deb5dfca803228f570f7c9ba093ea109700559b72fe1d385492f0d5a10f17a4cec41eb2e552f51e1f7f48ab311d4e195b1563c1fcba8ee201173e4e6362cabedaccee541f9efc9c4140d9fb268102021901b4031902f7041a2af78f2105140618640700")
	val := new(Validator)
	err := val.Decode(bs)
	require.NoError(t, err)
	assert.Equal(t, val.Stake(), int64(720867105))
	assert.Equal(t, val.Sequence(), 759)
	assert.Equal(t, val.LastBondingHeight(), 20)
	assert.Equal(t, val.UnbondingHeight(), 100)
	assert.Zero(t, val.LastJoinedHeight())
	bs2, _ := val.Encode()
	assert.Equal(t, bs, bs2)
	assert.Equal(t, val.Hash(), crypto.HashH(bs))
	expected, _ := crypto.HashFromString("4ae73f2be07945e814e21b106d4a4cb27982f01179c85231074799745b63b92d")
	assert.Equal(t, val.Hash(), expected)
}

func TestIncSequence(t *testing.T) {
	val, _ := GenerateTestValidator(util.RandInt(1000))
	seq := val.Sequence()
	val.IncSequence()
	assert.Equal(t, val.Sequence(), seq+1)
}

func TestNumber(t *testing.T) {
	val, _ := GenerateTestValidator(5)
	assert.Equal(t, val.Number(), 5)
}

func TestPower(t *testing.T) {
	val, _ := GenerateTestValidator(util.RandInt(1000))
	val.data.Stake = 0
	assert.Equal(t, val.Stake(), int64(0))
	assert.Equal(t, val.Power(), int64(1))
	val.data.Stake = 1
	assert.Equal(t, val.Stake(), int64(1))
	assert.Equal(t, val.Power(), int64(1))
	val.UpdateUnbondingHeight(1)
	assert.Equal(t, val.Stake(), int64(1))
	assert.Equal(t, val.Power(), int64(0))
}
