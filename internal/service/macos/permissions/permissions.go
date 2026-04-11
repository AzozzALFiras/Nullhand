package permissions

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

// Status describes whether each macOS privacy permission is granted.
type Status struct {
	ScreenRecording bool
	Accessibility   bool
}

// AllGranted reports whether every required permission is granted.
func (s Status) AllGranted() bool {
	return s.ScreenRecording && s.Accessibility
}

// Check probes both required macOS permissions and returns their status.
// Calling this also triggers the macOS TCC prompts on first run.
func Check() Status {
	return Status{
		ScreenRecording: checkScreenRecording(),
		Accessibility:   checkAccessibility(),
	}
}

// checkScreenRecording attempts a small screenshot and inspects stderr.
// Without the permission, screencapture writes a warning to stderr.
func checkScreenRecording() bool {
	f, err := os.CreateTemp("", "nullhand-perm-*.png")
	if err != nil {
		return false
	}
	path := f.Name()
	f.Close()
	defer os.Remove(path)

	var stderr bytes.Buffer
	cmd := exec.Command("screencapture", "-x", "-t", "png", path)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return false
	}

	// Any mention of permission in stderr means denied.
	se := strings.ToLower(stderr.String())
	if strings.Contains(se, "not authorized") || strings.Contains(se, "permission") {
		return false
	}

	info, err := os.Stat(path)
	if err != nil || info.Size() < 500 {
		return false
	}
	return true
}

// checkAccessibility tries to query System Events. Without accessibility
// permission, osascript fails with an "not authorized" error.
func checkAccessibility() bool {
	var stderr bytes.Buffer
	cmd := exec.Command("osascript", "-e",
		`tell application "System Events" to get name of first process whose frontmost is true`)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return false
	}
	se := strings.ToLower(stderr.String())
	if strings.Contains(se, "not authorized") || strings.Contains(se, "(-1743)") {
		return false
	}
	return true
}

// OpenScreenRecordingPane opens System Settings > Privacy > Screen Recording.
func OpenScreenRecordingPane() error {
	return exec.Command("open",
		"x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture").Run()
}

// OpenAccessibilityPane opens System Settings > Privacy > Accessibility.
func OpenAccessibilityPane() error {
	return exec.Command("open",
		"x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility").Run()
}

// OpenAutomationPane opens System Settings > Privacy > Automation.
func OpenAutomationPane() error {
	return exec.Command("open",
		"x-apple.systempreferences:com.apple.preference.security?Privacy_Automation").Run()
}
