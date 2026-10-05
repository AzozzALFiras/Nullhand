//go:build darwin

package mac

import (
	"time"

	"github.com/AzozzALFiras/Nullhand/internal/service/proc"
)

// Available reports whether this build can run the macOS features.
func Available() bool { return true }

// defaultTimeout bounds every helper process. osascript in particular can
// hang indefinitely when the app it talks to puts up a modal dialog, and the
// Telegram handler is waiting on it.
const defaultTimeout = 15 * time.Second

// osa runs an AppleScript snippet and returns its result.
func osa(script string) (string, error) {
	return proc.Run(defaultTimeout, "osascript", "-e", script)
}

// appRunning reports whether a GUI app is already running. Media control uses
// it so that asking what is playing never launches Music by itself.
//
// pgrep instead of AppleScript's `exists process`: System Events needs about
// 3.5 seconds per query on a current macOS, which made "what's playing" feel
// broken, and pgrep needs no Automation permission at all.
func appRunning(app string) bool {
	_, err := proc.Run(5*time.Second, "pgrep", "-x", app)
	return err == nil
}
