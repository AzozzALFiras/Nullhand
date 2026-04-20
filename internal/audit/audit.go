package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Logger appends structured audit lines to ~/.nullhand/audit.log.
// It is safe for concurrent use.
type Logger struct {
	mu   sync.Mutex
	file *os.File
}

// New opens (or creates) the audit log file and returns a Logger.
// The directory ~/.nullhand/ is created if it does not exist.
func New() (*Logger, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("audit: resolve home dir: %w", err)
	}

	dir := filepath.Join(home, ".nullhand")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("audit: create log dir: %w", err)
	}

	logPath := filepath.Join(dir, "audit.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("audit: open log file: %w", err)
	}

	return &Logger{file: f}, nil
}

// Log writes one audit line in the format:
//
//	[2026-04-14 22:31:05] user=123456789 action=screenshot [key="value" ...]
//
// extras are appended verbatim after the action field, e.g. `cmd="git status"`.
// Errors are silently swallowed so a log failure never crashes the bot.
func (l *Logger) Log(userID int64, action string, extras ...string) error {
	ts := time.Now().Format("2006-01-02 15:04:05")
	var sb strings.Builder
	fmt.Fprintf(&sb, "[%s] user=%d action=%s", ts, userID, action)
	for _, e := range extras {
		sb.WriteByte(' ')
		sb.WriteString(e)
	}
	sb.WriteByte('\n')

	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := l.file.WriteString(sb.String())
	return err
}

// Close flushes and closes the log file.
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		_ = l.file.Close()
		l.file = nil
	}
}
