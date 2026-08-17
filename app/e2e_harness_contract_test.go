package app_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
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

func TestMakefile_E2ETargetUsesSequentialRunner(t *testing.T) {
	body := readRepoFile(t, "Makefile")
	m := regexp.MustCompile(`(?m)^e2e:[^\n]*\n((?:\t[^\n]*\n?)*)`).FindStringSubmatch(body)
	require.NotNil(t, m, "Makefile must define an e2e target")
	require.Contains(t, m[1], "run-all-sequential.sh",
		"make e2e must invoke the serial runner; concurrent e2e go test processes destroy each other's containers")
}

func TestE2EWorkflow_PrepareDoesNotNeedImageBuilds(t *testing.T) {
	body := readRepoFile(t, ".github/workflows/e2e.yml")
	require.Regexp(t, `(?m)^  prepare:\n    runs-on: ubuntu-latest\n    timeout-minutes: 20\n    outputs:`, body,
		"prepare only lists tests; it must not wait on docker image builds")
}

func TestE2EWorkflow_OnlyIBCJobNeedsIBCImage(t *testing.T) {
	body := readRepoFile(t, ".github/workflows/e2e.yml")
	require.Contains(t, body, "\n  test-ibc:\n")
	require.Equal(t, 1, strings.Count(body, "- build-ibc"),
		"only the IBC test job should wait on the IBC image")
}
