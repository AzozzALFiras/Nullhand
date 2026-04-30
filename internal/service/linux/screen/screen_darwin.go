//go:build darwin

package screen

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Capture takes a full-screen screenshot via macOS `screencapture` and returns
// the PNG bytes. -x silences the shutter sound.
func Capture() ([]byte, error) {
	return captureWithArgs()
}

// CaptureResized takes a screenshot and resizes it to the logical resolution
// (or maxWidth if smaller). On Retina displays screencapture produces an image
// at native pixel resolution; resizing maps clicks 1:1.
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

	tmp, err := os.CreateTemp("", "nullhand-resize-*.png")
	if err != nil {
		return data, nil
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	if err := os.WriteFile(tmp.Name(), data, 0644); err != nil {
		return data, nil
	}

	// `sips` is built into macOS — no external dependency needed for resize.
	args := []string{"--resampleWidth", strconv.Itoa(targetWidth), tmp.Name()}
	if out, err := exec.Command("sips", args...).CombinedOutput(); err != nil {
		_ = out
		return data, nil
	}

	resized, err := os.ReadFile(tmp.Name())
	if err != nil {
		return data, nil
	}
	return resized, nil
}

// CaptureActive captures only the frontmost window via `screencapture -l <wid>`.
func CaptureActive() ([]byte, error) {
	wid := frontWindowID()
	if wid == "" {
		return captureWithArgs()
	}
	return captureWithArgs("-l", wid)
}

// captureWithArgs runs screencapture with optional extra flags, writing to a
// temp file and returning its PNG bytes.
func captureWithArgs(extra ...string) ([]byte, error) {
	tmp, err := os.CreateTemp("", "nullhand-*.png")
	if err != nil {
		return nil, fmt.Errorf("screen: create temp file: %w", err)
	}
	tmpName := tmp.Name()
	tmp.Close()
	os.Remove(tmpName)
	defer os.Remove(tmpName)

	// -x: silent (no shutter sound). -t png: PNG format.
	args := append([]string{"-x", "-t", "png"}, extra...)
	args = append(args, tmpName)
	cmd := exec.Command("screencapture", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("screencapture: %w — %s", err, strings.TrimSpace(string(out)))
	}
	data, err := os.ReadFile(tmpName)
	if err != nil {
		return nil, fmt.Errorf("screen: read captured file: %w", err)
	}
	return data, nil
}

// Size returns the logical screen resolution as (width, height).
// Uses AppleScript to query the main screen bounds — this gives logical
// (point) dimensions, which is what mouse coordinates use.
func Size() (int, int, error) {
	script := `tell application "Finder" to get bounds of window of desktop`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err == nil {
		// Output: "0, 0, 1920, 1080"
		parts := strings.Split(strings.TrimSpace(string(out)), ",")
		if len(parts) == 4 {
			w, err1 := strconv.Atoi(strings.TrimSpace(parts[2]))
			h, err2 := strconv.Atoi(strings.TrimSpace(parts[3]))
			if err1 == nil && err2 == nil && w > 0 && h > 0 {
				return w, h, nil
			}
		}
	}

	// Fallback: system_profiler SPDisplaysDataType
	out, err = exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		return 0, 0, fmt.Errorf("screen size: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		// Look for "Resolution: 1920 x 1080" or "UI Looks like: 1920 x 1080 @ 60.00Hz"
		if strings.HasPrefix(line, "UI Looks like:") || strings.HasPrefix(line, "Resolution:") {
			// "Resolution: 1920 x 1080 Retina" or "UI Looks like: 1920 x 1080 @ 60.00Hz"
			idx := strings.Index(line, ":")
			if idx < 0 {
				continue
			}
			rest := line[idx+1:]
			fields := strings.Fields(rest)
			if len(fields) < 3 || fields[1] != "x" {
				continue
			}
			w, err1 := strconv.Atoi(fields[0])
			h, err2 := strconv.Atoi(fields[2])
			if err1 == nil && err2 == nil && w > 0 && h > 0 {
				return w, h, nil
			}
		}
	}
	return 0, 0, fmt.Errorf("screen size: could not parse system_profiler output")
}

// frontWindowID returns the window ID of the active window. macOS uses
// CGWindowID (numeric) which screencapture accepts via -l.
func frontWindowID() string {
	// AppleScript to get the window ID of the frontmost window of the frontmost
	// app. Returns empty if not available.
	script := `tell application "System Events"
	set frontApp to first application process whose frontmost is true
	try
		set winID to id of front window of frontApp
		return winID as string
	on error
		return ""
	end try
end tell`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// WindowTitle returns the title of the frontmost window.
func WindowTitle() (string, error) {
	script := `tell application "System Events"
	set frontApp to first application process whose frontmost is true
	try
		return name of front window of frontApp
	on error
		return name of frontApp
	end try
end tell`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", fmt.Errorf("osascript window title: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// AnyWindowTitle returns the title of any open window whose title contains
// titleSubstr (case-insensitive). Empty if none found.
func AnyWindowTitle(titleSubstr string) (string, error) {
	if titleSubstr == "" {
		return WindowTitle()
	}
	// AppleScript: iterate every visible window of every process, return first
	// whose name contains the substring.
	script := fmt.Sprintf(`tell application "System Events"
	set needle to %q
	repeat with proc in (every application process whose visible is true)
		try
			repeat with win in (every window of proc)
				try
					set t to name of win
					if t contains needle then
						return t
					end if
				end try
			end repeat
		end try
	end repeat
	return ""
end tell`, titleSubstr)
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", fmt.Errorf("osascript any window: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// WaitForWindow polls every 200ms until a window with a matching title appears
// or timeoutMs expires. Returns the matched title on success.
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
		if title, err := WindowTitle(); err == nil && strings.Contains(strings.ToLower(title), needle) {
			return title, nil
		}
		if title, err := AnyWindowTitle(titleSubstr); err == nil && title != "" {
			return title, nil
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("WaitForWindow: timeout after %dms waiting for window %q", timeoutMs, titleSubstr)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
