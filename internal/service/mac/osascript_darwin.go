//go:build darwin

package mac

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Available reports whether this build can run the macOS features.
func Available() bool { return true }

// defaultTimeout bounds every helper process. osascript in particular can
// hang indefinitely when the app it talks to puts up a modal dialog, and the
// Telegram handler is waiting on it.
const defaultTimeout = 15 * time.Second

// run executes a command with a timeout and returns its trimmed stdout. On
// failure the error carries stderr, which is where osascript, pmset and the
// shortcuts CLI write their diagnostics.
func run(timeout time.Duration, name string, args ...string) (string, error) {
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
		if msg := firstLine(stderr.String()); msg != "" {
			return out, fmt.Errorf("%s: %s", name, msg)
		}
		return out, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

// osa runs an AppleScript snippet and returns its result.
func osa(script string) (string, error) {
	return run(defaultTimeout, "osascript", "-e", script)
}

// appRunning reports whether a GUI app is already running. Media control uses
// it so that asking what is playing never launches Music by itself.
//
// pgrep instead of AppleScript's `exists process`: System Events needs about
// 3.5 seconds per query on a current macOS, which made "what's playing" feel
// broken, and pgrep needs no Automation permission at all.
func appRunning(app string) bool {
	_, err := run(5*time.Second, "pgrep", "-x", app)
	return err == nil
}

// firstLine trims a multi-line tool error down to something a chat reply can
// show. AppleScript errors in particular come with a stack-ish second line.
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return s
}
