package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateCommitment(t *testing.T) {
	require := require.New(t)

	value := "0x0000000001"
	salt := "0x0000000002"
	commitment := CalculateCommitment(value, salt)
	expected := sha256.Sum256([]byte(value + ":" + salt))
	require.Equal(commitment, hex.EncodeToString(expected[:]))
}

func TestSalt(t *testing.T) {
	require := require.New(t)

	salt1, err := Salt(1)
	require.NoError(err)
	require.NotNil(salt1)

	salt2, err := Salt(0)
	require.NoError(err)
	require.NotNil(salt2)
	require.NotEqual(salt1, salt2)

	// salt3, err := Salt(1)
	// require.NoError(err)
	// require.NotNil(salt3)
	// require.NotEqual(salt1, salt3)
}
