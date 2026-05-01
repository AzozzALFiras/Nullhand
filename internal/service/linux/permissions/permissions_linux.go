//go:build linux

package permissions

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Status describes whether each required Linux capability is available.
type Status struct {
	ScreenRecording bool
	Accessibility   bool
}

// AllGranted reports whether every required capability is available.
func (s Status) AllGranted() bool {
	return s.ScreenRecording && s.Accessibility
}

// Check probes the two capabilities required at runtime and returns their
// status. It also validates the X11 session and key dependencies.
func Check() Status {
	return Status{
		ScreenRecording: checkScreenRecording(),
		Accessibility:   checkAccessibility(),
	}
}

// IsX11 reports whether the current desktop session is an X11 session.
// Wayland sessions are not supported in v1; callers should warn and exit.
func IsX11() bool {
	sessionType := strings.ToLower(strings.TrimSpace(os.Getenv("XDG_SESSION_TYPE")))
	return sessionType == "x11" || sessionType == ""
}

// CheckDependencies verifies that all required command-line tools are present.
// Returns a non-nil error listing every missing tool and the apt install command.
func CheckDependencies() error {
	required := []struct{ bin, pkg string }{
		{"xdotool", "xdotool"},
		{"scrot", "scrot"},
		{"wmctrl", "wmctrl"},
		{"xclip", "xclip"},
		{"convert", "imagemagick"},
		{"python3", "python3"},
		{"xrandr", "x11-xserver-utils"},
		{"gtk-launch", "libgtk-3-bin"},
	}

	var missing []string
	var pkgs []string
	for _, r := range required {
		if err := exec.Command("which", r.bin).Run(); err != nil {
			missing = append(missing, r.bin)
			pkgs = append(pkgs, r.pkg)
		}
	}

	// Check python3-pyatspi separately (it's a Python package, not a binary).
	if err := exec.Command("python3", "-c", "import pyatspi").Run(); err != nil {
		missing = append(missing, "python3-pyatspi")
		pkgs = append(pkgs, "python3-pyatspi at-spi2-core")
	}

	if len(missing) == 0 {
		// Optional: Arabic OCR pack. Don't block, just print a one-liner.
		warnArabicOCR()
		return nil
	}

	return fmt.Errorf(
		"missing tools: %s\n\nInstall with:\n  sudo apt install %s",
		strings.Join(missing, ", "),
		strings.Join(unique(pkgs), " "),
	)
}

// warnArabicOCR prints a one-line hint when tesseract is installed but the
// Arabic language pack is not. Bot still works in English-only OCR mode.
func warnArabicOCR() {
	if _, err := exec.LookPath("tesseract"); err != nil {
		return
	}
	out, err := exec.Command("tesseract", "--list-langs").CombinedOutput()
	if err != nil {
		return
	}
	if strings.Contains(string(out), "\nara") || strings.HasPrefix(string(out), "ara") {
		return
	}
	fmt.Println("ℹ️  Tesseract Arabic pack missing. For OCR of Arabic UI text run: sudo apt install tesseract-ocr-ara")
}

// checkScreenRecording checks that the scrot binary is present on PATH.
// We do not attempt an actual capture here — on Lubuntu/LXQt and other
// lightweight desktops scrot may fail during the permissions probe even
// though it works fine at runtime (e.g. no compositor yet, locked screen).
func checkScreenRecording() bool {
	_, err := exec.LookPath("scrot")
	return err == nil
}

// checkAccessibility verifies that xdotool can query the active window.
// This requires a working X11 display and the XTEST extension.
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

// OpenScreenRecordingPane opens the GNOME privacy settings panel.
// On Ubuntu GNOME this reveals display/privacy options.
func OpenScreenRecordingPane() error {
	// gnome-control-center privacy is the closest Linux equivalent.
	out, err := exec.Command("gnome-control-center", "privacy").CombinedOutput()
	if err != nil {
		// Fallback: xdg-open
		return exec.Command("xdg-open", "settings://privacy").Run()
	}
	_ = out
	return nil
}

// OpenAccessibilityPane opens the GNOME universal access panel.
func OpenAccessibilityPane() error {
	out, err := exec.Command("gnome-control-center", "universal-access").CombinedOutput()
	if err != nil {
		return exec.Command("xdg-open", "settings://universal-access").Run()
	}
	_ = out
	return nil
}

// OpenAutomationPane opens the general GNOME settings panel.
// Linux has no direct equivalent to macOS Automation privacy.
func OpenAutomationPane() error {
	out, err := exec.Command("gnome-control-center").CombinedOutput()
	if err != nil {
		return exec.Command("xdg-open", "settings://").Run()
	}
	_ = out
	return nil
}

// CheckX11Session verifies that the current session is X11, not Wayland or SSH.
// Returns an error if the environment is not suitable for running nullhand.
func CheckX11Session() error {
	if os.Getenv("DISPLAY") == "" {
		return fmt.Errorf("$DISPLAY not set — run nullhand in an X11 session, not SSH or Wayland")
	}
	if os.Getenv("WAYLAND_DISPLAY") != "" {
		return fmt.Errorf("Wayland detected — please log in with 'Ubuntu on Xorg' for full compatibility")
	}
	return nil
}

// unique deduplicates a slice while preserving order.
func unique(ss []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range ss {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
