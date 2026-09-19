//go:build unix

package shell

import (
	"errors"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A command that backgrounds a child must not leave that child running after
// the timeout: the whole process group is killed, not just the shell.
func TestExecuteKillsBackgroundChildrenOnTimeout(t *testing.T) {
	out, err := execute([]string{"sh", "-c", "sleep 30 & echo $!; wait"}, 300*time.Millisecond, MaxOutputBytes)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", err)
	}
	pid, convErr := strconv.Atoi(strings.TrimSpace(out))
	if convErr != nil {
		t.Fatalf("expected the background PID on stdout, got %q", out)
	}

	// The orphaned child is reaped by init asynchronously; give it a moment.
	deadline := time.Now().Add(2 * time.Second)
	for {
		if err := syscall.Kill(pid, 0); errors.Is(err, syscall.ESRCH) {
			return
		}
		if time.Now().After(deadline) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
			t.Fatalf("background child %d survived the timeout", pid)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
