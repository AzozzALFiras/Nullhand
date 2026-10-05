//go:build darwin

package mac

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AzozzALFiras/Nullhand/internal/service/proc"
)

// shortcutTimeout is generous: a Shortcut may talk to HomeKit, wait on a
// network call, or drive another app.
const shortcutTimeout = 2 * time.Minute

// ListShortcuts returns the names of the Shortcuts on this Mac.
func ListShortcuts() ([]string, error) {
	out, err := proc.Run(defaultTimeout, "shortcuts", "list")
	if err != nil {
		return nil, err
	}
	return parseShortcutNames(out), nil
}

// RunShortcut runs a Shortcut by name, optionally passing text as its input,
// and returns whatever text the Shortcut hands back.
//
// The CLI exchanges input and output as files, so text input is written to a
// temp file and output is read back from one. A Shortcut that returns nothing
// simply leaves the output file missing.
func RunShortcut(name, input string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("shortcut name is required")
	}

	dir, err := os.MkdirTemp("", "nullhand-shortcut-")
	if err != nil {
		return "", fmt.Errorf("shortcut: temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	args := []string{"run", name}
	if strings.TrimSpace(input) != "" {
		inputPath := filepath.Join(dir, "input.txt")
		if err := os.WriteFile(inputPath, []byte(input), 0600); err != nil {
			return "", fmt.Errorf("shortcut: write input: %w", err)
		}
		args = append(args, "--input-path", inputPath)
	}
	outputPath := filepath.Join(dir, "output.txt")
	args = append(args, "--output-path", outputPath, "--output-type", "public.plain-text")

	if _, err := proc.Run(shortcutTimeout, "shortcuts", args...); err != nil {
		return "", err
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return "", nil // ran fine, returned nothing
	}
	return strings.TrimSpace(string(data)), nil
}
