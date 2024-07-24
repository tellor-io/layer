package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupportedValueTypes(t *testing.T) {
	require := require.New(t)

	validTypes := []string{
		"int8",
		"int16",
		"int32",
		"int64",
		"int128",
		"int256",
		"int[]",
		"int8[]",
		"int16[]",
		"int32[]",
		"int64[]",
		"int128[]",
		"int256[]",
		"uint8",
		"uint16",
		"uint32",
		"uint64",
		"uint128",
		"uint256",
		"uint[]",
		"uint8[]",
		"uint16[]",
		"uint32[]",
		"uint64[]",
		"uint128[]",
		"uint256[]",
		"bytes",
		"string",
		"bool",
		"address",
		"bytes[]",
		"string[]",
		"bool[]",
		"address[]",
	}
	for _, tt := range validTypes {
		require.True(SupportedValueTypes[tt])
	}

	invalidTypes := []string{
		"a",
		"",
		"0",
	}

	for _, tt := range invalidTypes {
		require.False(SupportedValueTypes[tt])
	}
}
