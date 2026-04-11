package mouse

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Click performs a left mouse click at (x, y).
func Click(x, y int) error {
	return runAppleScript(clickScript("click", x, y))
}

// RightClick performs a right mouse click at (x, y).
func RightClick(x, y int) error {
	return runAppleScript(fmt.Sprintf(`
tell application "System Events"
  do shell script "cliclick rc:%d,%d"
end tell`, x, y))
}

// DoubleClick performs a double left click at (x, y).
func DoubleClick(x, y int) error {
	return runAppleScript(clickScript("doubleclick", x, y))
}

// Move moves the cursor to (x, y) without clicking.
func Move(x, y int) error {
	return runAppleScript(fmt.Sprintf(`
tell application "System Events"
  do shell script "cliclick m:%d,%d"
end tell`, x, y))
}

// Drag drags from (x1,y1) to (x2,y2).
func Drag(x1, y1, x2, y2 int) error {
	script := fmt.Sprintf(`
tell application "System Events"
  set src to {%d, %d}
  set dst to {%d, %d}
  drag from src to dst
end tell`, x1, y1, x2, y2)
	return runAppleScript(script)
}

// Scroll scrolls in direction ("up" | "down" | "left" | "right") by steps.
func Scroll(direction string, steps int) error {
	var deltaY, deltaX int
	switch strings.ToLower(direction) {
	case "up":
		deltaY = steps
	case "down":
		deltaY = -steps
	case "left":
		deltaX = steps
	case "right":
		deltaX = -steps
	default:
		return fmt.Errorf("unknown scroll direction: %s", direction)
	}

	script := fmt.Sprintf(`
tell application "System Events"
  scroll area 1 of window 1 by {%d, %d}
end tell`, deltaX, deltaY)
	_ = script

	// Use osascript-compatible scroll via key simulation
	key := "down"
	if deltaY > 0 {
		key = "up"
	}
	count := steps
	if count < 1 {
		count = 1
	}
	osa := fmt.Sprintf(`
tell application "System Events"
  repeat %d times
    key code 125 -- %s
  end repeat
end tell`, count, key)
	return runAppleScript(osa)
}

// clickScript builds an osascript click command using cliclick.
func clickScript(action string, x, y int) string {
	return fmt.Sprintf(`do shell script "cliclick %s:%s,%s"`,
		action,
		strconv.Itoa(x),
		strconv.Itoa(y),
	)
}

// runAppleScript executes the given AppleScript source via osascript.
func runAppleScript(script string) error {
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript error: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
