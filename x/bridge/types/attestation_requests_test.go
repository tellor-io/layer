package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAttestationRequests_AddRequest(t *testing.T) {
	attestationRequests := AttestationRequests{}
	attestationRequests.AddRequest(&AttestationRequest{
		Snapshot: []byte("snapshot"),
	})
	require.Equal(t, attestationRequests.Requests[0].Snapshot, []byte("snapshot"))
}

func TestAttestationRequests_HasSnapshot(t *testing.T) {
	attestationRequests := AttestationRequests{}
	require.False(t, attestationRequests.HasSnapshot([]byte("snapshot")))

	attestationRequests.AddRequest(&AttestationRequest{
		Snapshot: []byte("snapshot"),
	})
	require.True(t, attestationRequests.HasSnapshot([]byte("snapshot")))
	require.False(t, attestationRequests.HasSnapshot([]byte("other-snapshot")))
}
