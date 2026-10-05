package files

import (
	"fmt"
	"os"
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

// expand resolves ~ to the user home directory. Relative paths are left
// relative: Read and List accept them as the user typed them.
func expand(path string) (string, error) {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand path: %w", err)
		}
		return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(path, "~"), "/")), nil
	}
	return path, nil
}

// ExpandPath resolves ~ and returns an absolute path. Services that hand the
// path to another process (find, qlmanage, Finder) need it absolute, since
// those run with their own working directory.
func ExpandPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	expanded, err := expand(path)
	if err != nil {
		return "", err
	}
	return filepath.Abs(expanded)
}

// ExpandDir is ExpandPath with the home directory as the default, which is
// the right starting point for a file search from a phone.
func ExpandDir(dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return os.UserHomeDir()
	}
	return ExpandPath(dir)
}
