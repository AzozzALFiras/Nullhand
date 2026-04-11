package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// Stats holds a snapshot of system resource usage.
type Stats struct {
	CPUUsed   string // e.g. "14.1%"
	CPUIdle   string // e.g. "85.9%"
	MemUsed   string // e.g. "23G"
	MemUnused string // e.g. "1234M"
	ActiveApp string // foreground application name
}

// GetStats returns CPU, memory, and active app info in a single snapshot.
func GetStats() (*Stats, error) {
	s := &Stats{}

	// top -l 1 -n 0 → one sample, no process list (fast).
	out, err := exec.Command("top", "-l", "1", "-n", "0").Output()
	if err != nil {
		return nil, fmt.Errorf("top: %w", err)
	}
	s.parseTop(string(out))

	if app, err := ActiveApp(); err == nil {
		s.ActiveApp = app
	}
	return s, nil
}

// ActiveApp returns the name of the frontmost application.
func ActiveApp() (string, error) {
	script := `tell application "System Events" to get name of first process whose frontmost is true`
	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return "", fmt.Errorf("active app: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// parseTop extracts CPU and memory lines from `top` output.
// Expected format:
//
//	CPU usage: 7.38% user, 6.72% sys, 85.88% idle
//	PhysMem: 23G used (4567M wired, 3456M compressor), 1234M unused.
func (s *Stats) parseTop(output string) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "CPU usage:"):
			s.parseCPU(line)
		case strings.HasPrefix(line, "PhysMem:"):
			s.parseMem(line)
		}
	}
}

// parseCPU extracts user+sys and idle percentages.
func (s *Stats) parseCPU(line string) {
	// "CPU usage: 7.38% user, 6.72% sys, 85.88% idle"
	line = strings.TrimPrefix(line, "CPU usage:")
	parts := strings.Split(line, ",")
	var user, sys, idle string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		fields := strings.Fields(p)
		if len(fields) < 2 {
			continue
		}
		switch fields[1] {
		case "user":
			user = fields[0]
		case "sys":
			sys = fields[0]
		case "idle":
			idle = fields[0]
		}
	}
	if user != "" && sys != "" {
		s.CPUUsed = fmt.Sprintf("%s user + %s sys", user, sys)
	}
	s.CPUIdle = idle
}

// parseMem extracts used and unused memory totals.
func (s *Stats) parseMem(line string) {
	// "PhysMem: 23G used (4567M wired, 3456M compressor), 1234M unused."
	line = strings.TrimPrefix(line, "PhysMem:")
	line = strings.TrimSuffix(line, ".")

	parts := strings.Split(line, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.Contains(p, "used") {
			fields := strings.Fields(p)
			if len(fields) >= 1 {
				s.MemUsed = fields[0]
			}
		}
		if strings.Contains(p, "unused") {
			fields := strings.Fields(p)
			if len(fields) >= 1 {
				s.MemUnused = fields[0]
			}
		}
	}
}
