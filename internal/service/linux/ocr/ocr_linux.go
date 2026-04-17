//go:build linux

package ocr

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	screensvc "github.com/AzozzALFiras/nullhand/internal/service/linux/screen"
)

// ErrNotInstalled is returned when tesseract is not found on the system.
var ErrNotInstalled = errors.New("tesseract is not installed — run: sudo apt install tesseract-ocr")

// ReadScreen captures the current screen and extracts visible text via
// Tesseract OCR. Returns the extracted text (trimmed), or ErrNotInstalled if
// tesseract is not available, or another error if capture/OCR fails.
// Extracted text is truncated to 4096 characters (Telegram message limit).
func ReadScreen() (string, error) {
	// 1. Take screenshot
	data, err := screensvc.Capture()
	if err != nil {
		return "", fmt.Errorf("ocr: screenshot: %w", err)
	}

	// 2. Write to temp file
	tmp, err := os.CreateTemp("", "nullhand-ocr-*.png")
	if err != nil {
		return "", fmt.Errorf("ocr: create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", fmt.Errorf("ocr: write temp file: %w", err)
	}
	tmp.Close()

	// 3. Run tesseract
	cmd := exec.Command("tesseract", tmpPath, "stdout", "-l", "eng")
	out, err := cmd.Output()
	if err != nil {
		// Check if tesseract is missing entirely
		if _, lookErr := exec.LookPath("tesseract"); lookErr != nil {
			return "", ErrNotInstalled
		}
		// Tesseract exits non-zero on some images even when it produces output.
		// Return whatever stdout we got; if empty the caller handles it.
		text := strings.TrimSpace(string(out))
		if text == "" {
			return "", fmt.Errorf("ocr: tesseract failed: %w", err)
		}
		return truncate(text, 4096), nil
	}

	text := strings.TrimSpace(string(out))
	return truncate(text, 4096), nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
