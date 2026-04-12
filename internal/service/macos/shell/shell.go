package shell

import (
	"fmt"
	"os/exec"
	"strings"
)

// allowedCommands is the set of executable names permitted to run.
// Only the base command name is checked (not the full path).
var allowedCommands = map[string]bool{
	"ls":        true,
	"cat":       true,
	"echo":      true,
	"pwd":       true,
	"git":       true,
	"go":        true,
	"python3":   true,
	"python":    true,
	"node":      true,
	"npm":       true,
	"yarn":      true,
	"brew":      true,
	"curl":      true,
	"ping":      true,
	"df":        true,
	"du":        true,
	"top":       true,
	"ps":        true,
	"whoami":    true,
	"hostname":  true,
	"date":      true,
	"uname":     true,
	"find":      true,
	"grep":      true,
	"awk":       true,
	"sed":       true,
	"wc":        true,
	"sort":      true,
	"uniq":      true,
	"head":      true,
	"tail":      true,
	"mkdir":     true,
	"touch":     true,
	"cp":        true,
	"mv":        true,
	"open":      true,
	"xcode-select": true,
	"swift":     true,
	"make":      true,
	"cargo":     true,
	// Dev tools
	"pip":       true,
	"pip3":      true,
	"docker":    true,
	"ruby":      true,
	"gem":       true,
	"java":      true,
	"javac":     true,
	"flutter":   true,
	"pod":       true,
	"xcodebuild": true,
	// System utilities
	"which":     true,
	"env":       true,
	"printenv":  true,
	"killall":   true,
	"lsof":      true,
	"chmod":     true,
	"rm":        true,
	"ln":        true,
	// Archive
	"tar":       true,
	"zip":       true,
	"unzip":     true,
	// Network
	"ssh":       true,
	"scp":       true,
}

// Run executes a whitelisted shell command and returns combined stdout+stderr.
// The first token of cmdLine is the executable name; it must be in allowedCommands.
func Run(cmdLine string) (string, error) {
	parts := strings.Fields(cmdLine)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	base := parts[0]
	// Strip path prefix for the whitelist check.
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
