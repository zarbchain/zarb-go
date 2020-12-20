package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAggregation(t *testing.T) {
	_, pk1, pv1 := GenerateTestKeyPair()
	_, pk2, pv2 := GenerateTestKeyPair()
	_, pk3, pv3 := GenerateTestKeyPair()
	_, pk4, pv4 := GenerateTestKeyPair()
	msg1 := []byte("zarb")
	msg2 := []byte("zarb0")

	sig1 := pv1.Sign(msg1)
	sig11 := pv1.Sign(msg2)
	sig2 := pv2.Sign(msg1)
	sig3 := pv3.Sign(msg1)
	sig4 := pv4.Sign(msg1)

	agg1 := Aggregate([]*Signature{sig1, sig2, sig3})
	agg2 := Aggregate([]*Signature{sig1, sig2, sig4})
	agg3 := Aggregate([]*Signature{sig11, sig2, sig3})
	agg4 := Aggregate([]*Signature{sig1, sig2})
	agg5 := Aggregate([]*Signature{sig3, sig2, sig1})

	pks1 := []PublicKey{pk1, pk2, pk3}
	pks2 := []PublicKey{pk1, pk2, pk4}
	pks3 := []PublicKey{pk1, pk2}
	pks4 := []PublicKey{pk3, pk2, pk1}

	assert.True(t, VerifyAggregated(agg1, pks1, msg1))
	assert.False(t, VerifyAggregated(agg1, pks1, msg2))
	assert.False(t, VerifyAggregated(agg2, pks1, msg1))
	assert.False(t, VerifyAggregated(agg1, pks2, msg1))
	assert.True(t, VerifyAggregated(agg2, pks2, msg1))
	assert.False(t, VerifyAggregated(agg2, pks2, msg2))
	assert.False(t, VerifyAggregated(agg3, pks1, msg1))
	assert.False(t, VerifyAggregated(agg3, pks1, msg2))
	assert.False(t, VerifyAggregated(agg4, pks1, msg1))
	assert.False(t, VerifyAggregated(agg1, pks3, msg1))
	assert.True(t, VerifyAggregated(agg5, pks1, msg1))
	assert.True(t, VerifyAggregated(agg1, pks4, msg1))

}
