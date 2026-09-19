package shell

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"
)

// DefaultTimeout bounds how long a single command may run. Manual commands
// execute on the Telegram polling goroutine, so a command that never exits on
// its own (`ping host`, `top`, `tail -f`, `journalctl -f`) would otherwise
// freeze the whole bot — including /stop.
const DefaultTimeout = 30 * time.Second

// MaxOutputBytes caps how much combined stdout+stderr is kept per command.
// Output past the cap is discarded (the command keeps running) and a
// truncation note is appended.
const MaxOutputBytes = 64 * 1024

// ErrTimeout is returned (wrapped) when a command is killed for exceeding
// the time limit. Any output produced before the kill is still returned.
var ErrTimeout = errors.New("command timed out")

// timeout holds the configured limit in nanoseconds; zero means DefaultTimeout.
var timeout atomic.Int64

// SetTimeout changes the per-command time limit. d <= 0 restores DefaultTimeout.
func SetTimeout(d time.Duration) {
	if d < 0 {
		d = 0
	}
	timeout.Store(int64(d))
}

// Timeout returns the per-command time limit currently in effect.
func Timeout() time.Duration {
	if d := time.Duration(timeout.Load()); d > 0 {
		return d
	}
	return DefaultTimeout
}

// allowedCommands is the set of executable names permitted to run.
// Only the base command name is checked (not the full path).
var allowedCommands = map[string]bool{
	// Basic file operations
	"ls":      true,
	"cat":     true,
	"echo":    true,
	"pwd":     true,
	"find":    true,
	"grep":    true,
	"awk":     true,
	"sed":     true,
	"wc":      true,
	"sort":    true,
	"uniq":    true,
	"head":    true,
	"tail":    true,
	"mkdir":   true,
	"touch":   true,
	"cp":      true,
	"mv":      true,
	"rm":      true,
	"ln":      true,
	"chmod":   true,
	"chown":   true,
	// Version control
	"git": true,
	// Runtimes / build tools
	"go":      true,
	"python3": true,
	"python":  true,
	"node":    true,
	"npm":     true,
	"yarn":    true,
	"make":    true,
	"cargo":   true,
	// Package management (read-only / info only — apt install requires sudo)
	"apt":       true,
	"apt-cache": true,
	"dpkg":      true,
	"snap":      true,
	// Network
	"curl":    true,
	"wget":    true,
	"ping":    true,
	"ssh":     true,
	"scp":     true,
	"netstat": true,
	"ss":      true,
	// System info
	"df":       true,
	"du":       true,
	"top":      true,
	"htop":     true,
	"ps":       true,
	"whoami":   true,
	"hostname": true,
	"date":     true,
	"uname":    true,
	"uptime":   true,
	"free":     true,
	"lscpu":    true,
	"lsblk":    true,
	"lsusb":    true,
	"lspci":    true,
	"lsof":     true,
	"env":      true,
	"printenv": true,
	"which":    true,
	// Process management
	"kill":    true,
	"killall": true,
	"pkill":   true,
	// Archive
	"tar":   true,
	"zip":   true,
	"unzip": true,
	"gzip":  true,
	"gunzip": true,
	// Dev tools
	"pip":   true,
	"pip3":  true,
	"ruby":  true,
	"gem":   true,
	"java":  true,
	"javac": true,
	// Open file/URL
	"xdg-open": true,
	// Systemd (status / show — not start/stop to limit blast radius)
	"systemctl": true,
	"journalctl": true,
}

// Run executes a whitelisted shell command and returns combined stdout+stderr.
// The first token of cmdLine is the executable name; it must be in allowedCommands.
// The command is killed if it runs longer than Timeout().
func Run(cmdLine string) (string, error) {
	parts := strings.Fields(cmdLine)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	base := parts[0]
	if idx := strings.LastIndex(base, "/"); idx >= 0 {
		base = base[idx+1:]
	}

	if !allowedCommands[base] {
		return "", fmt.Errorf("command %q is not in the allowed list", parts[0])
	}

	return execute(parts, Timeout(), MaxOutputBytes)
}

// execute runs argv with a hard deadline and bounded output capture. On
// timeout the whole process group is killed, so children spawned by the
// command (e.g. `make` → compiler) don't linger holding the output pipe.
func execute(argv []string, limit time.Duration, maxOutput int) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), limit)
	defer cancel()

	out := &cappedBuffer{max: maxOutput}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdout = out
	cmd.Stderr = out
	killProcessGroupOnCancel(cmd)
	// A grandchild that escaped the group kill may still hold the output
	// pipe open; stop waiting for it shortly after the deadline.
	cmd.WaitDelay = time.Second

	err := cmd.Run()
	output := strings.TrimSpace(out.String())
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return output, fmt.Errorf("%w after %s and was stopped", ErrTimeout, limit)
	}
	if err != nil {
		return output, fmt.Errorf("command failed: %w", err)
	}
	return output, nil
}

// cappedBuffer is an io.Writer that keeps the first max bytes written to it
// and silently drops the rest. It always reports a full write so the command
// never sees EPIPE just because we stopped listening.
type cappedBuffer struct {
	buf       []byte
	max       int
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := b.max - len(b.buf); room > 0 {
		if len(p) > room {
			b.buf = append(b.buf, p[:room]...)
			b.truncated = true
		} else {
			b.buf = append(b.buf, p...)
		}
	} else if len(p) > 0 {
		b.truncated = true
	}
	return len(p), nil
}

// String returns the captured output. The cut may land inside a multi-byte
// character (Arabic output, box-drawing glyphs), so partial runes are dropped.
func (b *cappedBuffer) String() string {
	s := strings.ToValidUTF8(string(b.buf), "")
	if b.truncated {
		s += fmt.Sprintf("\n… [output truncated after %d bytes]", b.max)
	}
	return s
}

// IsAllowed reports whether the base command name is whitelisted.
func IsAllowed(cmdName string) bool {
	return allowedCommands[cmdName]
}
