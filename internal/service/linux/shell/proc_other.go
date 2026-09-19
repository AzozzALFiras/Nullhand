//go:build !unix

package shell

import "os/exec"

// killProcessGroupOnCancel is a no-op where process groups are unavailable;
// exec.CommandContext still kills the direct child on timeout.
func killProcessGroupOnCancel(cmd *exec.Cmd) {}
