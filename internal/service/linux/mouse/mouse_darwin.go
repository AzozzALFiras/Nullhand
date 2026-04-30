//go:build darwin

package mouse

import (
	"fmt"
	"os/exec"
	"strings"
)

// hasCliclick caches the result of looking up cliclick in PATH.
// cliclick is preferred (atomic move+click, supports drag and scroll
// reliably). osascript is the fallback for users who haven't installed it.
var cliclickAvailable = func() bool {
	_, err := exec.LookPath("cliclick")
	return err == nil
}()

// Click performs a left mouse click at (x, y).
func Click(x, y int) error {
	if cliclickAvailable {
		return cliclick(fmt.Sprintf("c:%d,%d", x, y))
	}
	return osaClick(x, y, "click")
}

// RightClick performs a right (secondary) mouse click at (x, y).
func RightClick(x, y int) error {
	if cliclickAvailable {
		return cliclick(fmt.Sprintf("rc:%d,%d", x, y))
	}
	return osaClick(x, y, "right click")
}

// DoubleClick performs a double left click at (x, y).
func DoubleClick(x, y int) error {
	if cliclickAvailable {
		return cliclick(fmt.Sprintf("dc:%d,%d", x, y))
	}
	if err := osaClick(x, y, "click"); err != nil {
		return err
	}
	return osaClick(x, y, "click")
}

// Move moves the cursor to (x, y) without clicking.
func Move(x, y int) error {
	if cliclickAvailable {
		return cliclick(fmt.Sprintf("m:%d,%d", x, y))
	}
	// osascript can position the mouse via System Events on macOS 10.14+.
	script := fmt.Sprintf(`tell application "System Events" to set mouse location to {%d, %d}`, x, y)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("move mouse via osascript: %w — %s (install cliclick: brew install cliclick)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Drag drags from (x1,y1) to (x2,y2).
func Drag(x1, y1, x2, y2 int) error {
	if !cliclickAvailable {
		return fmt.Errorf("drag requires cliclick on macOS — install with: brew install cliclick")
	}
	// cliclick: dd: drag down (press), dm: drag move, du: drag up (release)
	return cliclick(
		fmt.Sprintf("dd:%d,%d", x1, y1),
		fmt.Sprintf("dm:%d,%d", x2, y2),
		fmt.Sprintf("du:%d,%d", x2, y2),
	)
}

// Scroll scrolls the active region in a direction by N steps.
// Each "step" is roughly equivalent to one notch on a physical mouse wheel.
func Scroll(direction string, steps int) error {
	if steps < 1 {
		steps = 1
	}
	dir := strings.ToLower(direction)

	// Vertical scroll via Page Up / Page Down (works without cliclick).
	// Horizontal scroll via shift+arrow keys as a best-effort fallback.
	switch dir {
	case "up", "down":
		key := "page up"
		if dir == "down" {
			key = "page down"
		}
		// Each press = approximately 1 step.
		for i := 0; i < steps; i++ {
			script := fmt.Sprintf(`tell application "System Events" to key code %d`, pageKeyCode(key))
			out, err := exec.Command("osascript", "-e", script).CombinedOutput()
			if err != nil {
				return fmt.Errorf("scroll %s: %w — %s", dir, err, strings.TrimSpace(string(out)))
			}
		}
		return nil
	case "left", "right":
		// Best-effort — some apps don't support keyboard horizontal scroll.
		key := 123 // left arrow
		if dir == "right" {
			key = 124
		}
		for i := 0; i < steps; i++ {
			script := fmt.Sprintf(`tell application "System Events" to key code %d`, key)
			out, err := exec.Command("osascript", "-e", script).CombinedOutput()
			if err != nil {
				return fmt.Errorf("scroll %s: %w — %s", dir, err, strings.TrimSpace(string(out)))
			}
		}
		return nil
	}
	return fmt.Errorf("unknown scroll direction: %s", direction)
}

func pageKeyCode(name string) int {
	switch name {
	case "page up":
		return 116
	case "page down":
		return 121
	}
	return 0
}

// cliclick runs cliclick with the given commands chained together.
func cliclick(cmds ...string) error {
	args := append([]string{"-e", "0"}, cmds...) // -e 0 = no easing/delay
	cmd := exec.Command("cliclick", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cliclick: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// osaClick clicks at (x, y) via AppleScript. Slower than cliclick and only
// works on macOS 10.14+ with Accessibility permission granted to the bot.
func osaClick(x, y int, action string) error {
	// `click at {x, y}` works inside `tell application "System Events"`.
	script := fmt.Sprintf(`tell application "System Events" to %s at {%d, %d}`, action, x, y)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript click: %w — %s (install cliclick for reliable clicks: brew install cliclick)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
