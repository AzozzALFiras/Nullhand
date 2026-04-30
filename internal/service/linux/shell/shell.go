package shell

import (
	"fmt"
	"os/exec"
	"strings"
)

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

	cmd := exec.Command(parts[0], parts[1:]...)
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		return output, fmt.Errorf("command failed: %w", err)
	}
	return output, nil
}

// IsAllowed reports whether the base command name is whitelisted.
func IsAllowed(cmdName string) bool {
	return allowedCommands[cmdName]
}
