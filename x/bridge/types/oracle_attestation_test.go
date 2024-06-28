package types

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOracleAttestation_NewOracleAttestations(t *testing.T) {
	valsetSize := int(10)
	res := NewOracleAttestations(valsetSize)
	require.Equal(t, len(res.Attestations), valsetSize)
}

func TestOracleAttestation_SetAttestation(t *testing.T) {
	res := NewOracleAttestations(10)
	for i := 0; i < 10; i++ {
		attestation := []byte("attestation" + strconv.Itoa(i))
		res.SetAttestation(i, attestation)
		require.Equal(t, res.Attestations[i], attestation)
	}
}