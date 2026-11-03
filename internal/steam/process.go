package steam

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Running reports whether the Steam client is running, by looking for a process named
// "steam" in /proc (Flatpak Steam is visible there too).
func Running() bool { return runningIn("/proc") }

func runningIn(procDir string) bool {
	entries, err := os.ReadDir(procDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() || e.Name()[0] < '0' || e.Name()[0] > '9' {
			continue
		}
		comm, err := os.ReadFile(filepath.Join(procDir, e.Name(), "comm"))
		if err == nil && strings.TrimSpace(string(comm)) == "steam" {
			return true
		}
	}
	return false
}

// Command is the argv that runs this install's Steam client, to which arguments such as
// "-shutdown" or a "steam://" URL can be appended.
func (in Install) Command() []string {
	if in.Flatpak {
		return []string{"flatpak", "run", "com.valvesoftware.Steam"}
	}
	return []string{"steam"}
}

// Shutdown asks Steam to quit and waits until it has, up to timeout.
func (in Install) Shutdown(timeout time.Duration) error {
	if !Running() {
		return nil
	}
	argv := append(in.Command(), "-shutdown")
	if out, err := exec.Command(argv[0], argv[1:]...).CombinedOutput(); err != nil {
		return fmt.Errorf("steam -shutdown: %w: %s", err, strings.TrimSpace(string(out)))
	}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if !Running() {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("steam did not quit within %s", timeout)
}

// Start launches Steam detached from this process, with extra arguments.
func (in Install) Start(args ...string) error {
	argv := append(in.Command(), args...)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.SysProcAttr = detachAttr()
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
