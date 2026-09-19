//go:build unix

package shell

import (
	"os/exec"
	"syscall"
)

// killProcessGroupOnCancel starts the command in its own process group and,
// when its context is cancelled, kills the entire group instead of only the
// direct child.
func killProcessGroupOnCancel(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
