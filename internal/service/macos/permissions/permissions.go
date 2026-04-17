package permissions

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

// Status describes whether each required Linux capability is available.
type Status struct {
	ScreenRecording bool
	Accessibility   bool
}

// AllGranted reports whether every required permission is granted.
func (s Status) AllGranted() bool {
	return s.ScreenRecording && s.Accessibility
}

// Check probes both required Linux capabilities and returns their status.
func Check() Status {
	return Status{
		ScreenRecording: checkScreenRecording(),
		Accessibility:   checkAccessibility(),
	}
}

// checkScreenRecording attempts a small screenshot with scrot.
// Without a running X display or sufficient permissions, scrot will fail.
func checkScreenRecording() bool {
	f, err := os.CreateTemp("", "nullhand-perm-*.png")
	if err != nil {
		return false
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)

	var stderr bytes.Buffer
	cmd := exec.Command("scrot", path)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return false
	}

	// Any error message in stderr indicates a problem.
	se := strings.ToLower(stderr.String())
	if strings.Contains(se, "error") || strings.Contains(se, "permission") {
		return false
	}

	info, err := os.Stat(path)
	if err != nil || info.Size() < 500 {
		return false
	}
	return true
}

// checkAccessibility verifies that xdotool can query the active window,
// which requires a working X11 display and XTEST extension access.
func checkAccessibility() bool {
	var stderr bytes.Buffer
	cmd := exec.Command("xdotool", "getactivewindow")
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return false
	}
	se := strings.ToLower(stderr.String())
	if strings.Contains(se, "error") || strings.Contains(se, "unable") {
		return false
	}
	return true
}

// OpenScreenRecordingPane opens the system settings panel relevant to display
// / privacy on Linux. Falls back to the GNOME privacy panel.
func OpenScreenRecordingPane() error {
	return exec.Command("xdg-open", "settings://privacy").Run()
}

// OpenAccessibilityPane opens the accessibility settings panel.
func OpenAccessibilityPane() error {
	return exec.Command("xdg-open", "settings://universal-access").Run()
}

// OpenAutomationPane opens the general system settings panel (Linux has no
// direct Automation privacy pane equivalent).
func OpenAutomationPane() error {
	return exec.Command("xdg-open", "settings://").Run()
}
