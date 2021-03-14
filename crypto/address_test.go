package crypto

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddressMarshaling(t *testing.T) {
	addr1, _, _ := GenerateTestKeyPair()
	addr2 := new(Address)
	addr3 := new(Address)
	addr4 := new(Address)

	js, err := json.Marshal(addr1)
	assert.NoError(t, err)
	require.Error(t, addr2.UnmarshalJSON([]byte("bad")))
	require.NoError(t, json.Unmarshal(js, addr2))

	bs, err := addr2.MarshalCBOR()
	assert.NoError(t, err)
	assert.NoError(t, addr3.UnmarshalCBOR(bs))

	txt, err := addr2.MarshalText()
	assert.NoError(t, err)
	assert.NoError(t, addr4.UnmarshalText(txt))

	require.True(t, addr1.EqualsTo(*addr4))
	require.NoError(t, addr1.SanityCheck())
}

func TestAddressFromBytes(t *testing.T) {
	_, err := addressFromRawBytes(nil)
	assert.Error(t, err)
	addr1, _, _ := GenerateTestKeyPair()
	addr2, err := addressFromRawBytes(addr1.RawBytes())
	assert.NoError(t, err)
	require.True(t, addr1.EqualsTo(addr2))

	inv, _ := hex.DecodeString("0102")
	_, err = addressFromRawBytes(inv)
	assert.Error(t, err)
}

func TestAddressFromString(t *testing.T) {
	addr1, _, _ := GenerateTestKeyPair()
	addr2, err := AddressFromString(addr1.String())
	assert.NoError(t, err)
	require.True(t, addr1.EqualsTo(addr2))

	_, err = AddressFromString("inv")
	assert.Error(t, err)
}

func TestMarshalingEmptyAddress(t *testing.T) {
	addr1 := Address{}

	js, err := json.Marshal(addr1)
	assert.NoError(t, err)
	var addr2 Address
	err = json.Unmarshal(js, &addr2)
	assert.NoError(t, err)
	assert.Equal(t, addr1, addr2)

	assert.Error(t, addr2.SanityCheck())

	bs, err := addr1.MarshalCBOR()
	assert.NoError(t, err)
	var addr3 Address
	err = addr3.UnmarshalCBOR(bs)
	assert.NoError(t, err) /// No error
	assert.Equal(t, addr1, addr3)
}

func TestTreasuryAddress(t *testing.T) {
	expected, err := AddressFromString("0000000000000000000000000000000000000000")
	assert.NoError(t, err)
	assert.Equal(t, TreasuryAddress.RawBytes(), expected.RawBytes())
}
