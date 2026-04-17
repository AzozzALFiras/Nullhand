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
	return captureWithArgs()
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

	// Write to temp, resize with convert (ImageMagick), read back.
	tmp, err := os.CreateTemp("", "nullhand-resize-*.png")
	if err != nil {
		return data, nil // fallback: return full-size
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	if err := os.WriteFile(tmp.Name(), data, 0644); err != nil {
		return data, nil
	}

	// convert input.png -resize WIDTHx output.png  (only constrain width)
	convertArgs := []string{tmp.Name(), "-resize", strconv.Itoa(targetWidth) + "x", tmp.Name()}
	if out, err := exec.Command("convert", convertArgs...).CombinedOutput(); err != nil {
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
	wid := frontWindowID()
	if wid == "" {
		return captureWithArgs()
	}
	return captureWithArgs("--window", wid)
}

// captureWithArgs runs scrot with the given flags and a temp file.
func captureWithArgs(args ...string) ([]byte, error) {
	tmp, err := os.CreateTemp("", "nullhand-*.png")
	if err != nil {
		return nil, fmt.Errorf("screen: create temp file: %w", err)
	}
	tmp.Close()
	defer os.Remove(tmp.Name())

	// scrot [flags] filename
	cmdArgs := append(args, tmp.Name())
	out, err := exec.Command("scrot", cmdArgs...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("scrot: %w — %s", err, strings.TrimSpace(string(out)))
	}

	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		return nil, fmt.Errorf("screen: read captured file: %w", err)
	}
	return data, nil
}

// Size returns the primary display resolution as (width, height).
func Size() (int, int, error) {
	out, err := exec.Command("xrandr").Output()
	if err != nil {
		// Fallback: use xrandr with verbose flag
		return sizeFromProfiler()
	}

	// Look for a line like: "   1920x1080     60.00*+"
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		// Primary connected line: "eDP-1 connected primary 1920x1080+0+0 ..."
		if strings.Contains(line, " connected") && strings.Contains(line, "primary") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.Contains(p, "x") && strings.Contains(p, "+") {
					// e.g. "1920x1080+0+0"
					res := strings.Split(p, "+")[0]
					dims := strings.Split(res, "x")
					if len(dims) == 2 {
						w, err1 := strconv.Atoi(dims[0])
						h, err2 := strconv.Atoi(dims[1])
						if err1 == nil && err2 == nil && w > 0 && h > 0 {
							return w, h, nil
						}
					}
				}
			}
		}
	}
	return sizeFromProfiler()
}

// sizeFromProfiler is a fallback that reads display info via xrandr.
func sizeFromProfiler() (int, int, error) {
	out, err := exec.Command("xrandr").Output()
	if err != nil {
		return 0, 0, fmt.Errorf("screen size: %w", err)
	}
	// Look for the current mode line after a "connected" line.
	// Lines like: "   1920x1080     60.00*+"
	for _, line := range strings.Split(string(out), "\n") {
		trimmed := strings.TrimSpace(line)
		// Current mode has an asterisk
		if strings.Contains(trimmed, "*") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 1 {
				dims := strings.Split(parts[0], "x")
				if len(dims) == 2 {
					w, err1 := strconv.Atoi(dims[0])
					h, err2 := strconv.Atoi(dims[1])
					if err1 == nil && err2 == nil && w > 0 && h > 0 {
						return w, h, nil
					}
				}
			}
		}
	}
	return 0, 0, fmt.Errorf("screen size: could not parse display info")
}

// frontWindowID returns the window ID of the frontmost window as a string.
// Returns empty string on failure (scrot will then grab full screen).
func frontWindowID() string {
	out, err := exec.Command("xdotool", "getactivewindow").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
