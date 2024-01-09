package e2e

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const seed = 42

var latestVersion = "latest"

func TestE2ESimple(t *testing.T) {
	if os.Getenv("E2E") != "true" {
		t.Skip("skipping e2e test")
	}

	t.Log("Running simple e2e test", "version", latestVersion)

	testnet, err := New(t.Name(), seed)
	require.NoError(t, err)
	t.Cleanup(testnet.Cleanup)
	require.NoError(t, testnet.CreateGenesisNodes(2, latestVersion, 10000000, 0))

	kr, err := testnet.CreateAccount("alice", 1e12)
	require.NotNil(t, kr)
	require.NoError(t, err)

	require.NoError(t, testnet.Setup())
	require.NoError(t, testnet.Start())
}
