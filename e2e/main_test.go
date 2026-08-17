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

// flock serializes e2e processes on this machine. The ibc-test label sweep
// matches interchaintest cleanup; KEEP_CONTAINERS or ICTEST_SKIP_FAILURE_CLEANUP
// skip it. -list runs still take the lock.
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

	sweep := !listOnly && os.Getenv("KEEP_CONTAINERS") == "" && os.Getenv("ICTEST_SKIP_FAILURE_CLEANUP") == ""
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
	rm := func(list, remove []string) {
		out, _ := exec.Command(docker, list...).Output()
		ids := strings.Fields(string(out))
		if len(ids) == 0 {
			return
		}
		_ = exec.Command(docker, append(remove, ids...)...).Run()
	}
	rm([]string{"ps", "-aq", "--filter", "label=ibc-test"}, []string{"rm", "-f"})
	rm([]string{"volume", "ls", "-q", "--filter", "label=ibc-test"}, []string{"volume", "rm"})
	rm([]string{"network", "ls", "-q", "--filter", "label=ibc-test"}, []string{"network", "rm"})
}
