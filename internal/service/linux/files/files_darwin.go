//go:build darwin

package files

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetClipboard returns the current macOS pasteboard text via pbpaste.
func GetClipboard() (string, error) {
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		return "", fmt.Errorf("pbpaste: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// SetClipboard copies text to the macOS pasteboard via pbcopy.
func SetClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pbcopy: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
