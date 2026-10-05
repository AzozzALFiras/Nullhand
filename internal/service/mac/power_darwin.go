//go:build darwin

package mac

import (
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// BatteryStatus reports the battery level and charging state.
func BatteryStatus() (Battery, error) {
	out, err := run(defaultTimeout, "pmset", "-g", "batt")
	if err != nil {
		return Battery{}, err
	}
	return parseBattery(out)
}

// LockScreen locks the screen immediately.
func LockScreen() error {
	// Cmd+Ctrl+Q is the system lock shortcut and needs the Accessibility
	// permission the bot already asks for. Sleeping the display is the
	// fallback; it locks too whenever "require password immediately" is set.
	if _, err := osa(`tell application "System Events" to keystroke "q" using {command down, control down}`); err == nil {
		return nil
	}
	_, err := run(defaultTimeout, "pmset", "displaysleepnow")
	return err
}

// SleepNow puts the Mac to sleep.
func SleepNow() error {
	_, err := run(defaultTimeout, "pmset", "sleepnow")
	return err
}

// The caffeinate process is tracked so a later /awake off (or a new /awake)
// can stop it; without a handle the only way back would be killing every
// caffeinate on the system, including ones the user started.
var (
	awakeMu    sync.Mutex
	awakeCmd   *exec.Cmd
	awakeUntil time.Time
)

// KeepAwake stops the Mac from sleeping for d, replacing any previous
// keep-awake. A zero or negative d means "until cancelled". The returned time
// is the deadline, zero when there is none.
func KeepAwake(d time.Duration) (time.Time, error) {
	StopKeepAwake()

	cmd := exec.Command("caffeinate", keepAwakeArgs(d)...)
	if err := cmd.Start(); err != nil {
		return time.Time{}, fmt.Errorf("caffeinate: %w", err)
	}

	awakeMu.Lock()
	awakeCmd = cmd
	if d > 0 {
		awakeUntil = time.Now().Add(d)
	} else {
		awakeUntil = time.Time{}
	}
	until := awakeUntil
	awakeMu.Unlock()

	// Reap the process when its own -t deadline expires, and forget it so
	// KeepAwakeUntil stops claiming the Mac is being held awake.
	go func() {
		_ = cmd.Wait()
		awakeMu.Lock()
		if awakeCmd == cmd {
			awakeCmd = nil
			awakeUntil = time.Time{}
		}
		awakeMu.Unlock()
	}()

	return until, nil
}

// StopKeepAwake ends a keep-awake and reports whether one was running.
func StopKeepAwake() bool {
	awakeMu.Lock()
	cmd := awakeCmd
	awakeCmd = nil
	awakeUntil = time.Time{}
	awakeMu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return false
	}
	_ = cmd.Process.Kill()
	return true
}

// KeepAwakeUntil returns the current keep-awake deadline. ok is false when the
// Mac is free to sleep; a zero deadline with ok true means "until cancelled".
func KeepAwakeUntil() (time.Time, bool) {
	awakeMu.Lock()
	defer awakeMu.Unlock()
	return awakeUntil, awakeCmd != nil
}

// DarkMode reports whether the system appearance is currently dark.
func DarkMode() (bool, error) {
	out, err := osa(`tell application "System Events" to tell appearance preferences to get dark mode`)
	if err != nil {
		return false, err
	}
	return out == "true", nil
}

// SetDarkMode switches the system appearance.
func SetDarkMode(on bool) error {
	_, err := osa(fmt.Sprintf(
		`tell application "System Events" to tell appearance preferences to set dark mode to %t`, on))
	return err
}

// ToggleDarkMode flips the system appearance and returns the new state.
func ToggleDarkMode() (bool, error) {
	if _, err := osa(
		`tell application "System Events" to tell appearance preferences to set dark mode to not dark mode`); err != nil {
		return false, err
	}
	return DarkMode()
}
