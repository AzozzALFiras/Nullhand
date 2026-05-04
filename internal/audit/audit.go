package audit

import (
	"bufio"
	"fmt"
	"io"
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

// Path returns the absolute path of the audit log file. It is the canonical
// answer to "where do I look?" and is also used by tests + the /log command
// so they don't hardcode the home-relative path.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".nullhand", "audit.log"), nil
}

// Tail returns up to `n` of the most recent log lines, oldest-first. It reads
// from the end of the file in 16KB chunks so a multi-megabyte log doesn't
// have to be loaded into memory just to show the last few entries.
//
// The logger's own write mutex is intentionally NOT taken: appends are
// atomic at the OS level for short lines, so a concurrent write can at worst
// produce a partial trailing line, which we drop.
func Tail(path string, n int) ([]string, error) {
	if n <= 0 {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()
	if size == 0 {
		return nil, nil
	}

	const chunk int64 = 16 * 1024
	var buf []byte
	pos := size
	lines := 0

	for pos > 0 && lines <= n {
		readSize := chunk
		if pos < readSize {
			readSize = pos
		}
		pos -= readSize
		piece := make([]byte, readSize)
		if _, err := f.ReadAt(piece, pos); err != nil && err != io.EOF {
			return nil, err
		}
		buf = append(piece, buf...)
		// We over-count by one when the file ends with a newline (the final
		// "split" produces an empty trailing field) — that's fine, the slice
		// trim below handles it.
		lines = strings.Count(string(buf), "\n")
	}

	all := strings.Split(strings.TrimRight(string(buf), "\n"), "\n")
	// If we read past the start mid-line on the first chunk, drop that
	// partial line so callers never see truncated entries.
	if pos > 0 && len(all) > 0 {
		all = all[1:]
	}
	if len(all) > n {
		all = all[len(all)-n:]
	}
	return all, nil
}

// Search returns up to `limit` lines from the last `scan` entries that
// contain `query` (case-insensitive substring). Empty query returns the
// tail unfiltered. limit=0 is treated as scan.
func Search(path, query string, scan, limit int) ([]string, error) {
	tail, err := Tail(path, scan)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = scan
	}
	if query == "" {
		if len(tail) > limit {
			return tail[len(tail)-limit:], nil
		}
		return tail, nil
	}
	q := strings.ToLower(query)
	var out []string
	for _, line := range tail {
		if strings.Contains(strings.ToLower(line), q) {
			out = append(out, line)
		}
	}
	if len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

// readAllLines is a helper used only by tests for parity checks. It loads the
// entire file into a slice via bufio so tests can compare Tail's output
// against a known baseline.
func readAllLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		out = append(out, sc.Text())
	}
	return out, sc.Err()
}
