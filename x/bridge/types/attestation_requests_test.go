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
