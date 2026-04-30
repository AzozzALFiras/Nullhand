package filetransfer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PendingDownload holds the file metadata while waiting for the user to choose a destination.
type PendingDownload struct {
	FileID            string
	Filename          string
	MimeType          string
	AwaitingCustomPath bool // true only after the user tapped "Custom path"
}

// DownloadAndSave downloads the file from Telegram using the fileID and saves it to destDir.
// - Creates the directory if it doesn't exist.
// - Appends timestamp to filename if a conflict exists.
// - Returns the final saved path.
func DownloadAndSave(bot TelegramBot, chatID int64, fileID string, filename string, destDir string) error {
	data, suggestedName, err := bot.DownloadFile(fileID)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	if suggestedName != "" && filename == "" {
		filename = suggestedName
	}
	if filename == "" {
		filename = "downloaded_file"
	}

	// Sanitize filename to prevent path traversal
	filename = filepath.Base(filename)
	filename = strings.ReplaceAll(filename, "..", "")
	if filename == "" || filename == "." {
		filename = "downloaded_file"
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	fullPath := filepath.Join(destDir, filename)

	// If file exists, append timestamp
	if _, err := os.Stat(fullPath); err == nil {
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)
		timestamp := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("%s_%s%s", name, timestamp, ext)
		fullPath = filepath.Join(destDir, filename)
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
