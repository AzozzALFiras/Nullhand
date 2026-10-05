// Package proc runs short-lived helper processes with a deadline.
//
// Every platform service in Nullhand shells out to something — osascript,
// pmset, mdfind, find, qlmanage — and all of them need the same handling: a
// timeout, because the Telegram handler is waiting, and stderr in the error,
// because that is where those tools explain themselves.
package proc

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Run executes a command with a timeout and returns its trimmed stdout. On
// failure the error carries the first line of stderr.
func Run(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var stdout, stderr strings.Builder
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := strings.TrimSpace(stdout.String())
	if ctx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("%s timed out after %s", name, timeout)
	}
	if err != nil {
		if msg := FirstLine(stderr.String()); msg != "" {
			return out, fmt.Errorf("%s: %s", name, msg)
		}
		return out, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

// FirstLine trims a multi-line tool error down to something a chat reply can
// show. AppleScript errors in particular come with a stack-ish second line.
func FirstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}

// Lines splits tool output into trimmed, non-empty lines.
func Lines(out string) []string {
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}
