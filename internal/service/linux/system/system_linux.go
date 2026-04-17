//go:build linux

package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// Stats holds a snapshot of system resource usage.
type Stats struct {
	CPUUsed   string // e.g. "3.1% user + 1.2% sys"
	CPUIdle   string // e.g. "95.4%"
	MemUsed   string // e.g. "8192.0M"
	MemUnused string // e.g. "1234.5M"
	ActiveApp string // foreground window title
}

// GetStats returns CPU, memory, and active window info in a single snapshot.
func GetStats() (*Stats, error) {
	s := &Stats{}

	// top -bn1: one batch sample, no interactive mode.
	out, err := exec.Command("top", "-bn1").Output()
	if err != nil {
		return nil, fmt.Errorf("top: %w", err)
	}
	s.parseTop(string(out))

	if app, err := ActiveApp(); err == nil {
		s.ActiveApp = app
	}
	return s, nil
}

// ActiveApp returns the title of the currently active window.
func ActiveApp() (string, error) {
	out, err := exec.Command("xdotool", "getactivewindow", "getwindowname").Output()
	if err != nil {
		return "", fmt.Errorf("active app: %w (is xdotool installed? sudo apt install xdotool)", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// parseTop extracts CPU and memory lines from Linux `top -bn1` output.
// Expected formats:
//
//	%Cpu(s):  3.1 us,  1.2 sy,  0.0 ni, 95.4 id, ...
//	MiB Mem :  16384.0 total,   1234.5 free,   8192.0 used,   6957.5 buff/cache
//	KiB Mem :  16384.0 total, ...  (older kernels)
func (s *Stats) parseTop(output string) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "%Cpu"):
			s.parseCPU(line)
		case strings.HasPrefix(line, "MiB Mem") || strings.HasPrefix(line, "KiB Mem"):
			s.parseMem(line)
		}
	}
}

// parseCPU extracts user+sys and idle percentages from a Linux top CPU line.
// e.g. "%Cpu(s):  3.1 us,  1.2 sy,  0.0 ni, 95.4 id, ..."
func (s *Stats) parseCPU(line string) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return
	}
	line = line[idx+1:]

	var user, sys, idle string
	for _, segment := range strings.Split(line, ",") {
		fields := strings.Fields(strings.TrimSpace(segment))
		if len(fields) < 2 {
			continue
		}
		val := fields[0]
		label := fields[1]
		switch label {
		case "us":
			user = val + "%"
		case "sy":
			sys = val + "%"
		case "id":
			idle = val + "%"
		}
	}
	if user != "" && sys != "" {
		s.CPUUsed = fmt.Sprintf("%s user + %s sys", user, sys)
	}
	s.CPUIdle = idle
}

// parseMem extracts used and free memory from a Linux top memory line.
// e.g. "MiB Mem :  16384.0 total,   1234.5 free,   8192.0 used,   6957.5 buff/cache"
func (s *Stats) parseMem(line string) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return
	}
	unit := "M"
	if strings.HasPrefix(line, "KiB") {
		unit = "K"
	}
	line = line[idx+1:]

	for _, segment := range strings.Split(line, ",") {
		fields := strings.Fields(strings.TrimSpace(segment))
		if len(fields) < 2 {
			continue
		}
		val := fields[0]
		label := fields[1]
		switch label {
		case "used":
			s.MemUsed = val + unit
		case "free":
			s.MemUnused = val + unit
		}
	}
}
