//go:build !windows

// Package e2e_test provides the e2e test suite for the Layer blockchain.
//
// The lock file serializes e2e test processes on this machine; it does not
// guard the docker daemon itself. The lock fires on -list runs too —
// intended fail-fast. The !windows build tag removes the lock and the sweep
// on Windows. The label "ibc-test" is interchaintest's global cleanup label;
// the sweep can remove resources of an interchaintest run from another repo
// on the same daemon. Set KEEP_CONTAINERS or ICTEST_SKIP_FAILURE_CLEANUP to
// keep resources. On go test -timeout, the process dies by panic and the
// post-run sweep does not run; the next invocation's pre-run sweep removes
// leftovers.
package e2e_test

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	lockFile, err := os.OpenFile(filepath.Join(os.TempDir(), "layer-e2e.lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: cannot open lock file: %v\n", err)
		return 1
	}
	defer lockFile.Close()

	if err := unix.Flock(int(lockFile.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		fmt.Fprintln(os.Stderr, "e2e: another e2e test process is already running against this docker daemon.")
		fmt.Fprintln(os.Stderr, "e2e: tests must run one process at a time — use `make e2e` or e2e/run-all-sequential.sh.")
		return 1
	}

	flag.Parse()

	// test.list holds the -list regexp; any non-empty value means a list-only run.
	listOnly := false
	if fl := flag.Lookup("test.list"); fl != nil {
		listOnly = fl.Value.String() != ""
	}

	preserve := os.Getenv("KEEP_CONTAINERS") != "" || os.Getenv("ICTEST_SKIP_FAILURE_CLEANUP") != ""

	if !listOnly && !preserve {
		cleanupDockerByLabel()
	}

	code := m.Run()

	if !listOnly && !preserve {
		cleanupDockerByLabel()
	}

	return code
}

func cleanupDockerByLabel() {
	docker, err := exec.LookPath("docker")
	if err != nil {
		// docker not installed — nothing to clean.
		return
	}
	// containers
	out, _ := exec.Command(docker, "ps", "-aq", "--filter", "label=ibc-test").Output()
	for _, id := range strings.Fields(string(out)) {
		_ = exec.Command(docker, "rm", "-f", id).Run()
	}
	// volumes
	out, _ = exec.Command(docker, "volume", "ls", "-q", "--filter", "label=ibc-test").Output()
	for _, id := range strings.Fields(string(out)) {
		_ = exec.Command(docker, "volume", "rm", id).Run()
	}
	// networks
	out, _ = exec.Command(docker, "network", "ls", "-q", "--filter", "label=ibc-test").Output()
	for _, id := range strings.Fields(string(out)) {
		_ = exec.Command(docker, "network", "rm", id).Run()
	}
}
