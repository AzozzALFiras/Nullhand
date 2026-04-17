//go:build linux

package mouse

import (
	"fmt"
	"os/exec"
	"strings"
)

// Click performs a left mouse click at (x, y).
func Click(x, y int) error {
	// Single xdotool invocation: move then click — no inter-process race.
	return xdotool("mousemove", "--sync",
		fmt.Sprintf("%d", x), fmt.Sprintf("%d", y),
		"click", "1")
}

// RightClick performs a right mouse click at (x, y).
func RightClick(x, y int) error {
	return xdotool("mousemove", "--sync",
		fmt.Sprintf("%d", x), fmt.Sprintf("%d", y),
		"click", "3")
}

// DoubleClick performs a double left click at (x, y).
func DoubleClick(x, y int) error {
	return xdotool("mousemove", "--sync",
		fmt.Sprintf("%d", x), fmt.Sprintf("%d", y),
		"click", "--repeat", "2", "--delay", "100", "1")
}

// Move moves the cursor to (x, y) without clicking.
func Move(x, y int) error {
	return xdotool("mousemove", fmt.Sprintf("%d", x), fmt.Sprintf("%d", y))
}

// Drag drags from (x1,y1) to (x2,y2).
// Uses a single xdotool invocation to avoid inter-process race conditions
// between mousedown and mousemove.
func Drag(x1, y1, x2, y2 int) error {
	return xdotool(
		"mousemove", "--sync", fmt.Sprintf("%d", x1), fmt.Sprintf("%d", y1),
		"mousedown", "1",
		"mousemove", "--sync", fmt.Sprintf("%d", x2), fmt.Sprintf("%d", y2),
		"mouseup", "1",
	)
}

// Scroll scrolls in direction ("up" | "down" | "left" | "right") by steps.
// xdotool button numbers: 4=up, 5=down, 6=left, 7=right.
func Scroll(direction string, steps int) error {
	var button string
	switch strings.ToLower(direction) {
	case "up":
		button = "4"
	case "down":
		button = "5"
	case "left":
		button = "6"
	case "right":
		button = "7"
	default:
		return fmt.Errorf("unknown scroll direction: %s", direction)
	}

	count := steps
	if count < 1 {
		count = 1
	}
	return xdotool("click", "--repeat", fmt.Sprintf("%d", count), "--delay", "50", button)
}

// xdotool executes xdotool with the given arguments.
// All mouse operations use this helper so the error message is uniform.
func xdotool(args ...string) error {
	out, err := exec.Command("xdotool", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("xdotool error: %w — %s (is xdotool installed? sudo apt install xdotool)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
