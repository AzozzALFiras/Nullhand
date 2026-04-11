package files

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Read returns the text content of the file at path.
// The path is expanded (~ → home directory).
func Read(path string) (string, error) {
	path, err := expand(path)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %q: %w", path, err)
	}
	return string(data), nil
}

// Write writes content to the file at path, creating it if necessary.
func Write(path, content string) error {
	path, err := expand(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("write: mkdir: %w", err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// List returns the entries in the directory at path.
func List(path string) ([]string, error) {
	path, err := expand(path)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("ls %q: %w", path, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			name += "/"
		}
		names = append(names, name)
	}
	return names, nil
}

// GetClipboard returns the current macOS clipboard text content.
func GetClipboard() (string, error) {
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		return "", fmt.Errorf("pbpaste: %w", err)
	}
	return strings.TrimRight(string(out), "\n"), nil
}

// SetClipboard copies text to the macOS clipboard.
func SetClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pbcopy: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// expand resolves ~ to the user home directory.
func expand(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand path: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
