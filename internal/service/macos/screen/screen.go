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
