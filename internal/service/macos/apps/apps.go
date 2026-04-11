package apps

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Open launches an application by name AND brings it to the foreground.
// This is a single "make app ready to receive input" operation — anything
// that types or clicks after this call will land in the correct app.
func Open(appName string) error {
	out, err := exec.Command("open", "-a", appName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("open %q: %w — %s", appName, err, strings.TrimSpace(string(out)))
	}
	// `open -a` may return before the app is fully frontmost, especially if
	// it was already running in the background. Explicitly activate it.
	time.Sleep(120 * time.Millisecond)
	_ = Focus(appName)
	// Give the WindowServer a beat to finish the app switch before callers
	// start sending keystrokes.
	time.Sleep(180 * time.Millisecond)
	return nil
}

// List returns the names of all currently running applications.
func List() ([]string, error) {
	script := `
tell application "System Events"
  set appList to name of every process whose background only is false
  set output to ""
  repeat with appName in appList
    set output to output & appName & linefeed
  end repeat
  return output
end tell`

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return nil, fmt.Errorf("list apps: %w", err)
	}

	var apps []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			apps = append(apps, line)
		}
	}
	return apps, nil
}

// Focus brings the given application to the foreground.
func Focus(appName string) error {
	script := fmt.Sprintf(`tell application "%s" to activate`, appName)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("focus %q: %w — %s", appName, err, strings.TrimSpace(string(out)))
	}
	return nil
}
