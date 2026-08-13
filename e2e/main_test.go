//go:build !windows

package e2e_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

// TestMain fails fast if another e2e process already holds the docker daemon.
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

	return m.Run()
}
