//go:build !windows

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

// flock serializes e2e processes on this machine. The docker sweep removes
// every object carrying interchaintest's `ibc-test` label key — broader than
// interchaintest's own cleanup, which scopes to ibc-test=<test name> — so it
// hits leftovers from any interchaintest repo sharing the daemon. It only
// runs when E2E_DOCKER_SWEEP is set (run-all-sequential.sh and CI set it);
// KEEP_CONTAINERS or ICTEST_SKIP_FAILURE_CLEANUP skip it even then. -list
// runs still take the lock.
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

	listOnly := false
	if fl := flag.Lookup("test.list"); fl != nil {
		listOnly = fl.Value.String() != ""
	}

	sweep := !listOnly &&
		os.Getenv("E2E_DOCKER_SWEEP") != "" &&
		os.Getenv("KEEP_CONTAINERS") == "" &&
		os.Getenv("ICTEST_SKIP_FAILURE_CLEANUP") == ""
	if sweep {
		cleanupDockerByLabel()
	}

	code := m.Run()

	if sweep {
		cleanupDockerByLabel()
	}

	return code
}

func cleanupDockerByLabel() {
	docker, err := exec.LookPath("docker")
	if err != nil {
		return
	}
	rm := func(kind string, list, remove []string) {
		out, _ := exec.Command(docker, list...).Output()
		ids := strings.Fields(string(out))
		if len(ids) == 0 {
			return
		}
		// docker rm removes what it can and exits nonzero on any failure.
		if err := exec.Command(docker, append(remove, ids...)...).Run(); err != nil {
			fmt.Fprintf(os.Stderr, "e2e: sweep: %s removal incomplete (%v); targeted: %s\n", kind, err, strings.Join(ids, " "))
			return
		}
		fmt.Fprintf(os.Stderr, "e2e: sweep: removed %ss: %s\n", kind, strings.Join(ids, " "))
	}
	rm("container", []string{"ps", "-aq", "--filter", "label=ibc-test"}, []string{"rm", "-f"})
	rm("volume", []string{"volume", "ls", "-q", "--filter", "label=ibc-test"}, []string{"volume", "rm"})
	rm("network", []string{"network", "ls", "-q", "--filter", "label=ibc-test"}, []string{"network", "rm"})
}
