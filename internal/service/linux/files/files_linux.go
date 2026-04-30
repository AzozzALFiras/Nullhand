//go:build linux

package files

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetClipboard returns the current X11 clipboard text content via xclip.
func GetClipboard() (string, error) {
	cmd := exec.Command("xclip", "-selection", "clipboard", "-o")
	cmd.Env = append(os.Environ(), "DISPLAY=:0")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("xclip: %w (is xclip installed? sudo apt install xclip)", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// SetClipboard copies text to the X11 clipboard via xclip.
func SetClipboard(text string) error {
	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Env = append(os.Environ(), "DISPLAY=:0")
	cmd.Stdin = strings.NewReader(text)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("xclip: %w — %s (is xclip installed? sudo apt install xclip)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
