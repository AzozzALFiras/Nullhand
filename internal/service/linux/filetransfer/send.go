//go:build linux

package filetransfer

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// TelegramBot is the interface the Telegram client must satisfy for both send and receive operations.
type TelegramBot interface {
	SendMessage(chatID int64, text string) error
	SendPhoto(chatID int64, imageData []byte, caption string) error
	SendDocument(chatID int64, data []byte, filename string) error
	DownloadFile(fileID string) ([]byte, string, error)
}

// SendFile sends a file or directory from the local machine to the given Telegram chat.
// - Direct send if < 50MB
// - Auto-zips directories or files > 50MB
// - Chooses Photo/Video/Audio/Document based on extension for best Telegram rendering
// - Runs asynchronously so it doesn't block the message handler
// - Always cleans up temporary zip files
func SendFile(bot TelegramBot, chatID int64, path string) error {
	// Resolve ~ and make absolute
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not expand ~: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}
	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			_ = bot.SendMessage(chatID, fmt.Sprintf("❌ File not found: %s", path))
			return fmt.Errorf("file not found: %s", path)
		}
		_ = bot.SendMessage(chatID, fmt.Sprintf("❌ Cannot access %s: %v", path, err))
		return err
	}

	// Reply immediately so user knows we're working
	filename := filepath.Base(path)
	if info.IsDir() {
		filename += ".zip"
	}
	_ = bot.SendMessage(chatID, fmt.Sprintf("📤 Sending %s...", filename))

	// Run the actual send in a goroutine so we don't block the main handler
	go func() {
		data, finalName, tempPath, sendErr := prepareFile(path, info)
		if sendErr != nil {
			_ = bot.SendMessage(chatID, fmt.Sprintf("❌ Failed to prepare %s: %v", filename, sendErr))
			return
		}
		if tempPath != "" {
			defer os.Remove(tempPath)
		}

		sizeStr := formatSize(len(data))
		if sendErr = sendWithType(bot, chatID, data, finalName); sendErr != nil {
			_ = bot.SendMessage(chatID, fmt.Sprintf("❌ Failed to send %s: %v", finalName, sendErr))
			return
		}

		_ = bot.SendMessage(chatID, fmt.Sprintf("✅ Sent %s (%s)", finalName, sizeStr))
	}()

	return nil
}

// prepareFile returns the bytes to send, the display filename, and the temporary file path (if created).
// If the input is a directory or >50MB, it creates a temporary zip.
func prepareFile(path string, info os.FileInfo) ([]byte, string, string, error) {
	if info.IsDir() || info.Size() > 50*1024*1024 {
		data, displayName, tempPath, err := zipPath(path)
		return data, displayName, tempPath, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", "", err
	}
	return data, filepath.Base(path), "", nil
}

// zipPath creates a zip of a file or entire directory and returns the bytes, display name, and temp file path.
func zipPath(path string) ([]byte, string, string, error) {
	tmp, err := os.CreateTemp("", "nullhand-send-*.zip")
	if err != nil {
		return nil, "", "", err
	}
	defer tmp.Close()
	zipPath := tmp.Name()

	zw := zip.NewWriter(tmp)
	defer zw.Close()

	base := filepath.Base(path)
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", "", err
	}

	if info.IsDir() {
		err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(path, filePath)
			if err != nil {
				return err
			}
			f, err := zw.Create(rel)
			if err != nil {
				return err
			}
			data, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer data.Close()
			_, err = io.Copy(f, data)
			return err
		})
	} else {
		f, err := zw.Create(base)
		if err != nil {
			return nil, "", "", err
		}
		data, err := os.Open(path)
		if err != nil {
			return nil, "", "", err
		}
		defer data.Close()
		_, err = io.Copy(f, data)
		if err != nil {
			return nil, "", "", err
		}
	}

	if err != nil {
		os.Remove(zipPath)
		return nil, "", "", err
	}

	if err := zw.Close(); err != nil {
		os.Remove(zipPath)
		return nil, "", "", err
	}

	data, err := os.ReadFile(zipPath)
	if err != nil {
		os.Remove(zipPath)
		return nil, "", "", err
	}

	return data, base + ".zip", zipPath, nil
}

// sendWithType sends the data using the appropriate Telegram method based on extension.
func sendWithType(bot TelegramBot, chatID int64, data []byte, filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return bot.SendPhoto(chatID, data, filename)
	case ".mp4", ".mov", ".avi", ".mkv":
		// For video we could add SendVideo, but SendDocument works and shows preview.
		// Using SendDocument for simplicity as per current client capabilities.
		return bot.SendDocument(chatID, data, filename)
	case ".mp3", ".wav", ".ogg", ".flac":
		return bot.SendDocument(chatID, data, filename)
	default:
		return bot.SendDocument(chatID, data, filename)
	}
}

// formatSize returns a human readable size (KB or MB).
func formatSize(bytes int) string {
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
}
