package app_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

type e2eWorkflow struct {
	Jobs map[string]struct {
		Needs any `yaml:"needs"`
	} `yaml:"jobs"`
}

func loadE2EWorkflow(t *testing.T) e2eWorkflow {
	t.Helper()
	var wf e2eWorkflow
	require.NoError(t, yaml.Unmarshal([]byte(readRepoFile(t, ".github/workflows/e2e.yml")), &wf))
	require.NotEmpty(t, wf.Jobs)
	return wf
}

func needsList(v any) []string {
	switch n := v.(type) {
	case nil:
		return nil
	case string:
		return []string{n}
	case []any:
		out := make([]string, 0, len(n))
		for _, item := range n {
			s, ok := item.(string)
			if !ok {
				continue
			}
			out = append(out, s)
		}
		return out
	default:
		return nil
	}
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
	wf := loadE2EWorkflow(t)
	prepare, ok := wf.Jobs["prepare"]
	require.True(t, ok, "prepare job missing")
	require.Empty(t, needsList(prepare.Needs), "prepare only lists tests; it must not wait on docker image builds")
}

func TestE2EWorkflow_OnlyIBCJobNeedsIBCImage(t *testing.T) {
	wf := loadE2EWorkflow(t)
	var ibcWaiters []string
	for name, job := range wf.Jobs {
		if slices.Contains(needsList(job.Needs), "build-ibc") {
			ibcWaiters = append(ibcWaiters, name)
		}
	}
	slices.Sort(ibcWaiters)
	require.Equal(t, []string{"test-ibc"}, ibcWaiters, "only the IBC test job should wait on the IBC image")
}
