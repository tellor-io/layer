package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBridgeValsetSignatures_NewBridgeValsetSignatures(t *testing.T) {
	bridgeValsetSignatures := NewBridgeValsetSignatures(1)
	require.Equal(t, bridgeValsetSignatures, &BridgeValsetSignatures{
		Signatures: [][]byte{
			[]uint8{},
		},
	})

	bridgeValsetSignatures = NewBridgeValsetSignatures(2)
	require.Equal(t, bridgeValsetSignatures, &BridgeValsetSignatures{
		Signatures: [][]byte{
			[]uint8{},
			[]uint8{},
		},
	})
}