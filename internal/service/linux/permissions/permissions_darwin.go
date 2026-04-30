//go:build darwin

package permissions

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Status describes whether each required macOS capability is available.
type Status struct {
	ScreenRecording bool
	Accessibility   bool
}

// AllGranted reports whether every required capability is available.
func (s Status) AllGranted() bool {
	return s.ScreenRecording && s.Accessibility
}

// Check probes the two macOS capabilities required at runtime.
func Check() Status {
	return Status{
		ScreenRecording: checkScreenRecording(),
		Accessibility:   checkAccessibility(),
	}
}

// IsX11 always returns true on macOS — there's no equivalent gate, but we
// preserve the symbol for cross-platform callers.
func IsX11() bool {
	return true
}

// CheckDependencies verifies that all required command-line tools are present.
// On macOS we need: osascript (built-in), screencapture (built-in), tesseract
// (optional, Homebrew). cliclick is recommended but optional.
func CheckDependencies() error {
	required := []struct {
		bin string
		// hint shown if missing — usually the brew install command.
		hint string
	}{
		{"osascript", "ships with macOS — should always be present"},
		{"screencapture", "ships with macOS — should always be present"},
		{"sips", "ships with macOS — should always be present"},
	}
	optional := []struct {
		bin, brew string
	}{
		{"tesseract", "brew install tesseract"},
		{"cliclick", "brew install cliclick"},
	}

	var missing []string
	for _, r := range required {
		if _, err := exec.LookPath(r.bin); err != nil {
			missing = append(missing, fmt.Sprintf("  %s — %s", r.bin, r.hint))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required tools:\n%s", strings.Join(missing, "\n"))
	}

	// Optional tools: warn but don't fail.
	var warnings []string
	for _, o := range optional {
		if _, err := exec.LookPath(o.bin); err != nil {
			warnings = append(warnings, fmt.Sprintf("  %s missing (optional) — install with: %s", o.bin, o.brew))
		}
	}
	if len(warnings) > 0 {
		fmt.Println("⚠ Optional tools missing — some features will be limited:")
		for _, w := range warnings {
			fmt.Println(w)
		}
	}
	return nil
}

// checkScreenRecording verifies that the bot can take a screenshot. On macOS
// 10.15+ this requires the user to grant Screen Recording permission to
// whatever process invokes screencapture.
func checkScreenRecording() bool {
	// `screencapture -x -t png /dev/null` is the cheapest probe that exercises
	// the actual permission. We pipe to a tempfile and discard.
	out, err := exec.Command("screencapture", "-x", "-t", "png", "/dev/null").CombinedOutput()
	if err != nil {
		return false
	}
	if strings.Contains(strings.ToLower(string(out)), "permission") {
		return false
	}
	return true
}

// checkAccessibility verifies that osascript can query "System Events".
// This requires the bot's process to have Accessibility permission.
func checkAccessibility() bool {
	var stderr bytes.Buffer
	cmd := exec.Command("osascript", "-e", `tell application "System Events" to name of first application process whose frontmost is true`)
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return false
	}
	se := strings.ToLower(stderr.String())
	if strings.Contains(se, "not allowed") || strings.Contains(se, "1002") {
		return false
	}
	return true
}

// OpenScreenRecordingPane opens System Settings → Privacy → Screen Recording.
func OpenScreenRecordingPane() error {
	return exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_ScreenCapture").Run()
}

// OpenAccessibilityPane opens System Settings → Privacy → Accessibility.
func OpenAccessibilityPane() error {
	return exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Accessibility").Run()
}

// OpenAutomationPane opens System Settings → Privacy → Automation.
func OpenAutomationPane() error {
	return exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation").Run()
}

// CheckX11Session is a no-op on macOS (X11 is not the display server).
// Returns nil to signal "session is OK to run".
func CheckX11Session() error {
	return nil
}
