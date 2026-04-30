//go:build darwin

package apps

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// macAppNameMap maps common user-facing names to the canonical macOS app name
// (matching the .app bundle name without extension). `open -a` is fairly
// forgiving but normalising helps.
var macAppNameMap = map[string]string{
	"vscode":             "Visual Studio Code",
	"vs code":            "Visual Studio Code",
	"code":               "Visual Studio Code",
	"chrome":             "Google Chrome",
	"firefox":            "Firefox",
	"safari":             "Safari",
	"slack":              "Slack",
	"discord":            "Discord",
	"telegram":           "Telegram",
	"whatsapp":           "WhatsApp",
	"messages":           "Messages",
	"imessage":           "Messages",
	"mail":               "Mail",
	"finder":             "Finder",
	"terminal":           "Terminal",
	"iterm":              "iTerm",
	"notes":              "Notes",
	"settings":           "System Settings",
	"system settings":    "System Settings",
	"system preferences": "System Preferences",
	"calculator":         "Calculator",
	"app store":          "App Store",
}

// Open launches an application by name and brings it to the foreground.
func Open(appName string) error {
	resolved := resolveMacAppName(appName)
	cmd := exec.Command("open", "-a", resolved)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("open %q: %w — %s", appName, err, strings.TrimSpace(string(out)))
	}
	// Give the app a moment to come to the foreground.
	time.Sleep(400 * time.Millisecond)
	_ = Focus(resolved)
	return nil
}

// Focus brings a running application to the foreground.
func Focus(appName string) error {
	resolved := resolveMacAppName(appName)
	script := fmt.Sprintf(`tell application %q to activate`, resolved)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("focus %q: %w — %s", appName, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// CloseApp quits an application gracefully.
func CloseApp(name string) error {
	resolved := resolveMacAppName(name)
	script := fmt.Sprintf(`tell application %q to quit`, resolved)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		// Fallback: try pkill on the app's process name.
		if e2 := exec.Command("pkill", "-f", resolved).Run(); e2 == nil {
			return nil
		}
		return fmt.Errorf("close %q: %w — %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// List returns the bundle/process names of all currently running apps with a
// visible window.
func List() ([]string, error) {
	script := `tell application "System Events" to get name of every application process whose visible is true`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return nil, fmt.Errorf("list apps: %w", err)
	}
	// Output is a comma-separated list.
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ", ")
	out2 := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		name := strings.TrimSpace(p)
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out2 = append(out2, name)
	}
	return out2, nil
}

// resolveMacAppName normalises the input via macAppNameMap, falling back to
// the input unchanged.
func resolveMacAppName(name string) string {
	low := strings.ToLower(strings.TrimSpace(name))
	if mapped, ok := macAppNameMap[low]; ok {
		return mapped
	}
	return strings.TrimSpace(name)
}
