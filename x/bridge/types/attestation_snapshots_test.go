package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAttestationSnapshots_NewAttestationSnapshots(t *testing.T) {
	attestationSnapshots := NewAttestationSnapshots()
	fmt.Println(attestationSnapshots)
	require.Equal(t, len(attestationSnapshots.Snapshots), 0)
	require.Equal(t, attestationSnapshots.String(), "")
	require.Equal(t, attestationSnapshots, &AttestationSnapshots{
		Snapshots: [][]byte{},
	})
}

func TestAttestationSnapshots_SetSnapshot(t *testing.T) {
	attestationSnapshots := NewAttestationSnapshots()
	attestationSnapshots.SetSnapshot([]byte("snapshot"))
	require.Equal(t, attestationSnapshots, &AttestationSnapshots{
		Snapshots: [][]byte{[]byte("snapshot")},
	})
}
