//go:build darwin

package system

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// GetStats returns CPU, memory, and active window info on macOS.
// Uses `top -l 1` (one sample) plus AppleScript for the active window.
func GetStats() (*Stats, error) {
	s := &Stats{}

	// `top -l 1 -n 0` produces one snapshot without process listing.
	// -F: skip detailed process info; -R: register without delay.
	out, err := exec.Command("top", "-l", "1", "-n", "0", "-F", "-R").Output()
	if err != nil {
		// Fall back to less restrictive flags if older macOS rejects -F/-R.
		out, err = exec.Command("top", "-l", "1").Output()
		if err != nil {
			return nil, fmt.Errorf("top: %w", err)
		}
	}
	s.parseTop(string(out))

	if app, err := ActiveApp(); err == nil {
		s.ActiveApp = app
	}
	return s, nil
}

// ActiveApp returns the title of the frontmost application's frontmost window.
func ActiveApp() (string, error) {
	script := `tell application "System Events"
	set frontApp to first application process whose frontmost is true
	try
		return name of front window of frontApp
	on error
		return name of frontApp
	end try
end tell`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", fmt.Errorf("active app: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// parseTop extracts CPU and memory lines from macOS `top -l 1` output.
// Expected formats (slightly different from Linux):
//
//	CPU usage: 3.12% user, 1.41% sys, 95.46% idle
//	PhysMem: 8192M used (3072M wired), 1234M unused.
func (s *Stats) parseTop(output string) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "CPU usage:"):
			s.parseCPUDarwin(line)
		case strings.HasPrefix(line, "PhysMem:"):
			s.parseMemDarwin(line)
		}
	}
}

// parseCPUDarwin extracts user/sys/idle from a macOS top CPU line.
//
//	CPU usage: 3.12% user, 1.41% sys, 95.46% idle
func (s *Stats) parseCPUDarwin(line string) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return
	}
	rest := line[idx+1:]
	var user, sys, idle string
	for _, segment := range strings.Split(rest, ",") {
		segment = strings.TrimSpace(segment)
		fields := strings.Fields(segment)
		if len(fields) < 2 {
			continue
		}
		val := strings.TrimSuffix(fields[0], "%")
		if _, err := strconv.ParseFloat(val, 64); err != nil {
			continue
		}
		label := fields[1]
		switch label {
		case "user":
			user = val + "%"
		case "sys":
			sys = val + "%"
		case "idle":
			idle = val + "%"
		}
	}
	if user != "" && sys != "" {
		s.CPUUsed = fmt.Sprintf("%s user + %s sys", user, sys)
	}
	s.CPUIdle = idle
}

// parseMemDarwin extracts used/unused from a macOS top memory line.
//
//	PhysMem: 8192M used (3072M wired), 1234M unused.
func (s *Stats) parseMemDarwin(line string) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return
	}
	rest := line[idx+1:]
	for _, segment := range strings.Split(rest, ",") {
		segment = strings.TrimSpace(segment)
		fields := strings.Fields(segment)
		if len(fields) < 2 {
			continue
		}
		val := fields[0]
		// Strip parenthetical detail like "(3072M wired)" — already split by ','.
		if strings.Contains(segment, "used") {
			s.MemUsed = val
		}
		if strings.Contains(segment, "unused") {
			s.MemUnused = strings.TrimRight(val, ".")
		}
	}
}
