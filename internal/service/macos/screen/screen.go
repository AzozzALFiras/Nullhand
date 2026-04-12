package screen

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Capture takes a full-screen screenshot and returns the PNG bytes.
func Capture() ([]byte, error) {
	return captureWithArgs("-x") // -x = no sound
}

// CaptureForVision takes a full-screen screenshot and resizes it to match
// the logical screen resolution. On Retina displays, screencapture produces
// images at physical resolution (e.g. 2880x1800) but click coordinates use
// logical resolution (e.g. 1440x900). Resizing to logical width ensures
// pixel coordinates in the image map 1:1 to click coordinates.
//
// If maxWidth is provided and smaller than the logical width, the image is
// resized to maxWidth instead (capping token cost).
func CaptureResized(maxWidth int) ([]byte, error) {
	data, err := Capture()
	if err != nil {
		return nil, err
	}

	// Determine target width: logical screen width, capped by maxWidth.
	targetWidth := maxWidth
	if logicalW, _, err := Size(); err == nil && logicalW > 0 {
		if logicalW < targetWidth || targetWidth <= 0 {
			targetWidth = logicalW
		}
	}
	if targetWidth <= 0 {
		return data, nil // can't determine size, return full
	}

	// Write to temp, resize with sips, read back.
	tmp, err := os.CreateTemp("", "nullhand-resize-*.png")
	if err != nil {
		return data, nil // fallback: return full-size
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	if err := os.WriteFile(tmp.Name(), data, 0644); err != nil {
		return data, nil
	}

	sipsArgs := []string{"--resampleWidth", strconv.Itoa(targetWidth), tmp.Name()}
	if out, err := exec.Command("sips", sipsArgs...).CombinedOutput(); err != nil {
		_ = out
		return data, nil // fallback: return full-size
	}

	resized, err := os.ReadFile(tmp.Name())
	if err != nil {
		return data, nil
	}
	return resized, nil
}

// CaptureActive takes a screenshot of the active (frontmost) window.
func CaptureActive() ([]byte, error) {
	return captureWithArgs("-x", "-l", frontWindowID())
}

// captureWithArgs runs screencapture with the given flags and a temp file.
func captureWithArgs(args ...string) ([]byte, error) {
	tmp, err := os.CreateTemp("", "nullhand-*.png")
	if err != nil {
		return nil, fmt.Errorf("screen: create temp file: %w", err)
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	cmdArgs := append(args, tmp.Name())
	out, err := exec.Command("screencapture", cmdArgs...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("screencapture: %w — %s", err, strings.TrimSpace(string(out)))
	}

	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		return nil, fmt.Errorf("screen: read captured file: %w", err)
	}
	return data, nil
}

// Size returns the primary display resolution as (width, height).
func Size() (int, int, error) {
	script := `tell application "Finder" to get bounds of window of desktop`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		// Fallback: use system_profiler
		return sizeFromProfiler()
	}

	// Output: "0, 0, 1920, 1080"
	parts := strings.Split(strings.TrimSpace(string(out)), ", ")
	if len(parts) != 4 {
		return sizeFromProfiler()
	}
	w, err1 := strconv.Atoi(parts[2])
	h, err2 := strconv.Atoi(parts[3])
	if err1 != nil || err2 != nil {
		return sizeFromProfiler()
	}
	return w, h, nil
}

// sizeFromProfiler is a fallback that reads display info via system_profiler.
func sizeFromProfiler() (int, int, error) {
	out, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		return 0, 0, fmt.Errorf("screen size: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Resolution:") {
			// e.g. "Resolution: 2560 x 1600"
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				w, _ := strconv.Atoi(parts[1])
				h, _ := strconv.Atoi(parts[3])
				if w > 0 && h > 0 {
					return w, h, nil
				}
			}
		}
	}
	return 0, 0, fmt.Errorf("screen size: could not parse display info")
}

// frontWindowID returns the window ID of the frontmost window as a string.
// Returns empty string on failure (screencapture will then grab full screen).
func frontWindowID() string {
	script := `tell application "System Events" to get id of first window of (first process whose frontmost is true)`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
