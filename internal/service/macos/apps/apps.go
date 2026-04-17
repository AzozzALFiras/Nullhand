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
	// gtk-launch uses .desktop file names; try the app name directly as a
	// binary first, falling back to xdg-open for known file/URI handlers.
	out, err := exec.Command("bash", "-c",
		fmt.Sprintf("nohup %s >/dev/null 2>&1 & disown", shellQuote(appName))).CombinedOutput()
	if err != nil {
		// Fallback: try gtk-launch (requires a matching .desktop entry).
		out2, err2 := exec.Command("gtk-launch", appName).CombinedOutput()
		if err2 != nil {
			return fmt.Errorf("open %q: %w — %s %s",
				appName, err, strings.TrimSpace(string(out)), strings.TrimSpace(string(out2)))
		}
	}
	// Give the app time to start / come to the foreground.
	time.Sleep(120 * time.Millisecond)
	_ = Focus(appName)
	// Give the window manager a beat to finish the app switch.
	time.Sleep(180 * time.Millisecond)
	return nil
}

// List returns the names of all currently running applications.
func List() ([]string, error) {
	// wmctrl -l lists all open windows; -p adds PIDs.
	// We extract unique window titles as a proxy for running apps.
	out, err := exec.Command("wmctrl", "-l").Output()
	if err != nil {
		return nil, fmt.Errorf("list apps: %w", err)
	}

	seen := map[string]struct{}{}
	var apps []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		// wmctrl -l format: <wid> <desktop> <host> <title…>
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		title := strings.Join(fields[3:], " ")
		if title == "" {
			continue
		}
		if _, dup := seen[title]; !dup {
			seen[title] = struct{}{}
			apps = append(apps, title)
		}
	}
	return apps, nil
}

// Focus brings the given application to the foreground.
func Focus(appName string) error {
	out, err := exec.Command("wmctrl", "-a", appName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("focus %q: %w — %s", appName, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// shellQuote wraps s in single quotes, escaping any existing single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
