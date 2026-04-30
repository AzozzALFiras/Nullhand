//go:build linux

package screen

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Capture takes a full-screen screenshot and returns the PNG bytes.
func Capture() ([]byte, error) {
	return captureWithArgs()
}

// CaptureResized takes a full-screen screenshot and resizes it to the logical
// screen resolution (or maxWidth if smaller). On HiDPI/fractional-scaling
// desktops scrot captures at native resolution; resizing here ensures pixel
// coordinates in the image map 1:1 to click coordinates.
func CaptureResized(maxWidth int) ([]byte, error) {
	data, err := Capture()
	if err != nil {
		return nil, err
	}

	targetWidth := maxWidth
	if logicalW, _, err := Size(); err == nil && logicalW > 0 {
		if logicalW < targetWidth || targetWidth <= 0 {
			targetWidth = logicalW
		}
	}
	if targetWidth <= 0 {
		return data, nil
	}

	// Write to temp, resize with convert (ImageMagick), read back.
	tmp, err := os.CreateTemp("", "nullhand-resize-*.png")
	if err != nil {
		return data, nil
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	if err := os.WriteFile(tmp.Name(), data, 0644); err != nil {
		return data, nil
	}

	// -resize Wx — constrain width only, preserve aspect ratio.
	args := []string{tmp.Name(), "-resize", strconv.Itoa(targetWidth) + "x", tmp.Name()}
	if out, err := exec.Command("convert", args...).CombinedOutput(); err != nil {
		_ = out
		return data, nil
	}

	resized, err := os.ReadFile(tmp.Name())
	if err != nil {
		return data, nil
	}
	return resized, nil
}

// CaptureActive takes a screenshot of the active (frontmost) window.
func CaptureActive() ([]byte, error) {
	wid := frontWindowID()
	if wid == "" {
		return captureWithArgs()
	}
	// scrot --window <wid> captures only that window.
	return captureWithArgs("--window", wid)
}

// captureWithArgs runs scrot with optional extra flags and saves to a temp file.
func captureWithArgs(args ...string) ([]byte, error) {
	tmp, err := os.CreateTemp("", "nullhand-*.png")
	if err != nil {
		return nil, fmt.Errorf("screen: create temp file: %w", err)
	}
	tmpName := tmp.Name()
	tmp.Close()
	os.Remove(tmpName) // delete so scrot can create it fresh
	defer os.Remove(tmpName)

	// scrot [flags] filename
	cmdArgs := append(args, tmpName)
	cmd := exec.Command("scrot", cmdArgs...)
	cmd.Env = append(os.Environ(), "DISPLAY=:0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("scrot: %w — %s (is scrot installed? sudo apt install scrot)", err, strings.TrimSpace(string(out)))
	}

	data, err := os.ReadFile(tmpName)
	if err != nil {
		return nil, fmt.Errorf("screen: read captured file: %w", err)
	}
	return data, nil
}

// Size returns the primary display resolution as (width, height).
// Parses xrandr output looking for the connected primary display line.
func Size() (int, int, error) {
	xrandrCmd := exec.Command("xrandr")
	xrandrCmd.Env = append(os.Environ(), "DISPLAY=:0")
	out, err := xrandrCmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("xrandr: %w (is xrandr installed? sudo apt install x11-xserver-utils)", err)
	}

	// First pass: look for the "connected primary WxH+x+y" pattern on the
	// connected line itself (present when a monitor is set as primary).
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, " connected") && strings.Contains(line, "primary") {
			if w, h, ok := parseResolutionGeometry(line); ok {
				return w, h, nil
			}
		}
	}

	// Second pass: find the current mode line (contains '*') that follows
	// any connected display line — handles the case where no primary is set.
	inConnected := false
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, " connected") {
			inConnected = true
			// Try geometry on the same line first.
			if w, h, ok := parseResolutionGeometry(line); ok {
				return w, h, nil
			}
			continue
		}
		if inConnected && strings.Contains(line, "*") {
			// Mode line, e.g. "   1920x1080     60.00*+"
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				dims := strings.Split(fields[0], "x")
				if len(dims) == 2 {
					w, err1 := strconv.Atoi(dims[0])
					h, err2 := strconv.Atoi(dims[1])
					if err1 == nil && err2 == nil && w > 0 && h > 0 {
						return w, h, nil
					}
				}
			}
		}
		if strings.Contains(line, " connected") || strings.Contains(line, " disconnected") {
			inConnected = strings.Contains(line, " connected")
		}
	}

	return 0, 0, fmt.Errorf("screen size: could not parse xrandr output")
}

// parseResolutionGeometry extracts WxH from a string like "eDP-1 connected primary 1920x1080+0+0".
func parseResolutionGeometry(line string) (int, int, bool) {
	for _, field := range strings.Fields(line) {
		if strings.Contains(field, "x") && strings.Contains(field, "+") {
			res := strings.Split(field, "+")[0]
			dims := strings.Split(res, "x")
			if len(dims) == 2 {
				w, err1 := strconv.Atoi(dims[0])
				h, err2 := strconv.Atoi(dims[1])
				if err1 == nil && err2 == nil && w > 0 && h > 0 {
					return w, h, true
				}
			}
		}
	}
	return 0, 0, false
}

// frontWindowID returns the X window ID of the active window as a string.
// Returns empty string on failure (Capture falls back to full screen).
func frontWindowID() string {
	cmd := exec.Command("xdotool", "getactivewindow")
	cmd.Env = append(os.Environ(), "DISPLAY=:0")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// WindowTitle returns the title of the active (frontmost) window via xdotool.
// Empty string if no window is active or xdotool fails.
func WindowTitle() (string, error) {
	cmd := exec.Command("xdotool", "getactivewindow", "getwindowname")
	cmd.Env = append(os.Environ(), "DISPLAY=:0")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("xdotool getwindowname: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// AnyWindowTitle searches all open windows for one whose title contains
// titleSubstr (case-insensitive). Returns the matching window's title or
// empty string if none. Uses wmctrl -l for the listing.
func AnyWindowTitle(titleSubstr string) (string, error) {
	if titleSubstr == "" {
		return WindowTitle()
	}
	cmd := exec.Command("wmctrl", "-l")
	cmd.Env = append(os.Environ(), "DISPLAY=:0")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("wmctrl -l: %w", err)
	}
	needle := strings.ToLower(titleSubstr)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.SplitN(line, " ", 5)
		if len(fields) < 5 {
			continue
		}
		title := fields[4]
		if strings.Contains(strings.ToLower(title), needle) {
			return strings.TrimSpace(title), nil
		}
	}
	return "", nil
}

// WaitForWindow polls every 200ms until either the active window or any open
// window has a title containing titleSubstr (case-insensitive), or until the
// timeout expires. Returns the matched title on success.
func WaitForWindow(titleSubstr string, timeoutMs int) (string, error) {
	if titleSubstr == "" {
		return "", fmt.Errorf("WaitForWindow: titleSubstr is required")
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	needle := strings.ToLower(titleSubstr)
	for {
		// Check active window first (cheaper).
		if title, err := WindowTitle(); err == nil && strings.Contains(strings.ToLower(title), needle) {
			return title, nil
		}
		// Then any open window.
		if title, err := AnyWindowTitle(titleSubstr); err == nil && title != "" {
			return title, nil
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("WaitForWindow: timeout after %dms waiting for window %q", timeoutMs, titleSubstr)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
