//go:build linux

package apps

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// desktopIDMap maps common user-facing app names to their .desktop file IDs.
// gtk-launch uses these IDs (without the .desktop extension).
var desktopIDMap = map[string]string{
	"visual studio code":  "code",
	"vscode":              "code",
	"vs code":             "code",
	"google chrome":       "google-chrome",
	"chrome":              "google-chrome",
	"chromium":            "chromium-browser",
	"firefox":             "firefox",
	"thunderbird":         "thunderbird",
	"nautilus":            "org.gnome.Nautilus",
	"files":               "org.gnome.Nautilus",
	"terminal":            "org.gnome.Terminal",
	"gnome terminal":      "org.gnome.Terminal",
	"gedit":               "org.gnome.gedit",
	"text editor":         "org.gnome.TextEditor",
	"gnome text editor":   "org.gnome.TextEditor",
	"slack":               "slack",
	"discord":             "discord",
	"telegram":            "org.telegram.desktop",
	"spotify":             "spotify",
	"vlc":                 "vlc",
	"libreoffice writer":  "libreoffice-writer",
	"libreoffice calc":    "libreoffice-calc",
	"libreoffice impress": "libreoffice-impress",
	"evince":              "org.gnome.Evince",
	"calculator":          "org.gnome.Calculator",
	"gnome calculator":    "org.gnome.Calculator",
	"calendar":            "org.gnome.Calendar",
	"settings":            "gnome-control-center",
	"system settings":     "gnome-control-center",
	"system monitor":      "gnome-system-monitor",
}

// Open launches an application by name and brings it to the foreground.
func Open(appName string) error {
	normalised := strings.ToLower(strings.TrimSpace(appName))

	// 1. Try the .desktop ID map first (most reliable).
	if id, ok := desktopIDMap[normalised]; ok {
		if err := gtkLaunch(id); err == nil {
			time.Sleep(120 * time.Millisecond)
			_ = Focus(appName)
			time.Sleep(180 * time.Millisecond)
			return nil
		}
	}

	// 2. Try gtk-launch with the raw app name as the desktop ID.
	if err := gtkLaunch(normalised); err == nil {
		time.Sleep(120 * time.Millisecond)
		_ = Focus(appName)
		time.Sleep(180 * time.Millisecond)
		return nil
	}

	// 3. Last resort: run as a binary directly.
	cmd := exec.Command(appName)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open %q: %w (install the app or check its .desktop file)", appName, err)
	}
	time.Sleep(120 * time.Millisecond)
	_ = Focus(appName)
	time.Sleep(180 * time.Millisecond)
	return nil
}

// List returns window titles of all currently open windows via wmctrl -l -x.
// We return WM_CLASS instance names (e.g. "Navigator", "code", "slack") rather
// than raw titles, which are more stable across locales and window states.
func List() ([]string, error) {
	// -l -x: list windows + WM_CLASS column.
	// Format: <wid>  <desktop>  <wm_class.instance.WM_CLASS>  <host>  <title>
	out, err := exec.Command("wmctrl", "-l", "-x").Output()
	if err != nil {
		return nil, fmt.Errorf("list apps: %w (is wmctrl installed? sudo apt install wmctrl)", err)
	}

	seen := map[string]struct{}{}
	var apps []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		// wmctrl -l -x: field[2] = WM_CLASS (e.g. "code.Code")
		if len(fields) < 5 {
			continue
		}
		wmClass := fields[2]
		// WM_CLASS is "instance.ClassName" — take the instance part.
		if dot := strings.Index(wmClass, "."); dot > 0 {
			wmClass = wmClass[:dot]
		}
		wmClass = strings.ToLower(wmClass)
		if wmClass == "" || wmClass == "n/a" {
			continue
		}
		if _, dup := seen[wmClass]; !dup {
			seen[wmClass] = struct{}{}
			apps = append(apps, wmClass)
		}
	}
	return apps, nil
}

// Focus brings the given application to the foreground using wmctrl.
// Tries matching against window titles as a fallback if WM_CLASS fails.
func Focus(appName string) error {
	// Try WM_CLASS match first (-x flag with -a).
	normalised := strings.ToLower(strings.TrimSpace(appName))
	if id, ok := desktopIDMap[normalised]; ok {
		// wmctrl -x -a matches against WM_CLASS.
		if err := exec.Command("wmctrl", "-x", "-a", id).Run(); err == nil {
			return nil
		}
	}
	// Fall back to title match.
	out, err := exec.Command("wmctrl", "-a", appName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("focus %q: %w — %s (is wmctrl installed? sudo apt install wmctrl)", appName, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// gtkLaunch launches an application via gtk-launch with the given desktop ID.
func gtkLaunch(desktopID string) error {
	cmd := exec.Command("gtk-launch", desktopID)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("gtk-launch %q: %w", desktopID, err)
	}
	return nil
}
