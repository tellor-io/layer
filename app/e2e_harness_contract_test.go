package app_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), ".."))
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	require.NoError(t, err)
	return string(body)
}

func TestSequentialRunner_DoesNotUseGNUXargsR(t *testing.T) {
	body := readRepoFile(t, "e2e/run-all-sequential.sh")
	require.NotContains(t, body, "xargs -r", "BSD xargs (darwin) rejects -r; labeled container cleanup becomes a no-op")
}

func TestSequentialRunner_DoesNotGloballyPruneDocker(t *testing.T) {
	body := readRepoFile(t, "e2e/run-all-sequential.sh")
	require.NotContains(t, body, "docker volume prune", "volume prune is host-global and deletes unrelated unused volumes")
	require.NotContains(t, body, "docker network prune", "network prune is host-global and deletes unrelated unused networks")
}

func TestSequentialRunner_UsesGoTestList(t *testing.T) {
	body := readRepoFile(t, "e2e/run-all-sequential.sh")
	require.Contains(t, body, "go test -list", "runner must use the same discovery as CI prepare")
	require.NotContains(t, body, "grep -h '^func Test'", "grep discovery is not compilation-aware and needs a TestMain special case")
}
