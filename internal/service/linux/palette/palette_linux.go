//go:build linux

package palette

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Run opens the command palette of the frontmost application by pressing the
// given keyboard shortcut, pastes the command name via the clipboard
// (Unicode-safe), and presses Return to execute it.
//
// Examples:
//
//	Run("ctrl+shift+p", "Claude Code: Focus Chat")   // VS Code / Cursor
//	Run("ctrl+k", "jump to channel #general")        // Slack / Discord
//	Run("ctrl+p", "quick find")                      // Notion / Obsidian
//
// Requires xdotool and xclip: sudo apt install xdotool xclip
func Run(shortcut, command string) error {
	if shortcut == "" {
		return fmt.Errorf("palette: shortcut is required")
	}
	if command == "" {
		return fmt.Errorf("palette: command is required")
	}

	time.Sleep(150 * time.Millisecond)

	if err := pressShortcut(shortcut); err != nil {
		return fmt.Errorf("palette: open: %w", err)
	}

	// Give Electron apps time to render the palette input.
	time.Sleep(250 * time.Millisecond)

	prev, hadPrev := readClipboard()

	if err := writeClipboard(command); err != nil {
		return fmt.Errorf("palette: set clipboard: %w", err)
	}
	time.Sleep(60 * time.Millisecond)

	// Ctrl+V paste — layout-independent, works regardless of keyboard locale.
	if err := xdotoolKey("ctrl+v"); err != nil {
		return fmt.Errorf("palette: paste command: %w", err)
	}
	time.Sleep(150 * time.Millisecond)

	if err := xdotoolKey("Return"); err != nil {
		return fmt.Errorf("palette: press return: %w", err)
	}

	time.Sleep(250 * time.Millisecond)

	if hadPrev {
		_ = writeClipboard(prev)
	}
	return nil
}

// pressShortcut parses a "ctrl+shift+p" style string and sends it via xdotool.
func pressShortcut(shortcut string) error {
	parts := strings.Split(strings.ToLower(shortcut), "+")
	if len(parts) == 0 {
		return fmt.Errorf("empty shortcut")
	}
	key := strings.TrimSpace(parts[len(parts)-1])
	modifiers := parts[:len(parts)-1]

	xkey, ok := specialKeyNames[key]
	if !ok {
		if mapped, ok2 := letterKeyNames[key]; ok2 {
			xkey = mapped
		} else {
			// Unknown key — pass through; xdotool will reject it clearly.
			return fmt.Errorf("unsupported palette key %q", key)
		}
	}

	modStr := buildModifiers(modifiers)
	var combo string
	if modStr != "" {
		combo = modStr + "+" + xkey
	} else {
		combo = xkey
	}
	return xdotoolKey(combo)
}

// buildModifiers converts modifier slice to xdotool format (e.g. "ctrl+shift").
// cmd/command → ctrl, option → alt (macOS shortcut strings are accepted).
func buildModifiers(mods []string) string {
	var out []string
	for _, m := range mods {
		switch strings.TrimSpace(m) {
		case "cmd", "command":
			out = append(out, "ctrl")
		case "shift":
			out = append(out, "shift")
		case "ctrl", "control":
			out = append(out, "ctrl")
		case "alt", "option", "opt":
			out = append(out, "alt")
		case "super", "win":
			out = append(out, "super")
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "+")
}

// specialKeyNames maps human-readable names to X11/xdotool key symbols.
var specialKeyNames = map[string]string{
	"return": "Return", "enter": "Return",
	"tab": "Tab", "space": "space", "escape": "Escape", "esc": "Escape",
	"delete": "BackSpace", "backspace": "BackSpace",
	"/": "slash", "\\": "backslash",
	"up": "Up", "down": "Down", "left": "Left", "right": "Right",
	"-": "minus", "=": "equal",
	"[": "bracketleft", "]": "bracketright",
	";": "semicolon", "'": "apostrophe",
	",": "comma", ".": "period", "`": "grave",
}

// letterKeyNames maps letters and digits to themselves (xdotool accepts them).
var letterKeyNames = map[string]string{
	"a": "a", "b": "b", "c": "c", "d": "d", "e": "e", "f": "f",
	"g": "g", "h": "h", "i": "i", "j": "j", "k": "k", "l": "l",
	"m": "m", "n": "n", "o": "o", "p": "p", "q": "q", "r": "r",
	"s": "s", "t": "t", "u": "u", "v": "v", "w": "w", "x": "x",
	"y": "y", "z": "z",
	"0": "0", "1": "1", "2": "2", "3": "3", "4": "4",
	"5": "5", "6": "6", "7": "7", "8": "8", "9": "9",
}

func xdotoolKey(combo string) error {
	out, err := exec.Command("xdotool", "key", combo).CombinedOutput()
	if err != nil {
		return fmt.Errorf("xdotool: %w — %s (is xdotool installed? sudo apt install xdotool)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func readClipboard() (string, bool) {
	out, err := exec.Command("xclip", "-selection", "clipboard", "-o").Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}

func writeClipboard(text string) error {
	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
