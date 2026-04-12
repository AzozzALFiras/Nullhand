package palette

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Run opens the command palette of the frontmost application by pressing
// the given keyboard shortcut, pastes the command name via the clipboard
// (Unicode-safe), and presses Return to execute it.
//
// Examples:
//
//	Run("cmd+shift+p", "Claude Code: Focus Chat")   // VS Code / Cursor
//	Run("cmd+k", "jump to channel #general")        // Slack / Discord / Linear
//	Run("cmd+/", "find plugin")                     // Figma
//	Run("cmd+p", "quick find")                      // Notion / Obsidian
//
// This is the vision-free way to reach any named command inside an Electron
// app. The palette input is auto-focused by the host app, and the host's
// fuzzy matcher finds the command — we never query the AX tree.
//
// Requires macOS Accessibility permission (for System Events keystrokes).
func Run(shortcut, command string) error {
	if shortcut == "" {
		return fmt.Errorf("palette: shortcut is required")
	}
	if command == "" {
		return fmt.Errorf("palette: command is required")
	}

	// Give the frontmost app a beat to be ready to receive the shortcut.
	time.Sleep(150 * time.Millisecond)

	if err := pressShortcut(shortcut); err != nil {
		return fmt.Errorf("palette: open: %w", err)
	}

	// Palette render time. Electron apps need a few hundred ms before the
	// palette input is ready to accept keystrokes.
	time.Sleep(250 * time.Millisecond)

	prev, hadPrev := readClipboard()

	if err := writeClipboard(command); err != nil {
		return fmt.Errorf("palette: set clipboard: %w", err)
	}
	time.Sleep(60 * time.Millisecond)

	// Use key code 9 (V) instead of keystroke "v" — keystroke breaks when
	// the macOS keyboard layout is non-Latin (Arabic, Chinese, etc.)
	if err := runAppleScript(`tell application "System Events" to key code 9 using command down`); err != nil {
		return fmt.Errorf("palette: paste command: %w", err)
	}
	time.Sleep(150 * time.Millisecond)

	// Fire Return to execute the palette command.
	if err := runAppleScript(`tell application "System Events" to key code 36`); err != nil {
		return fmt.Errorf("palette: press return: %w", err)
	}

	// Let the command actually run before we touch anything else.
	time.Sleep(250 * time.Millisecond)

	if hadPrev {
		_ = writeClipboard(prev)
	}
	return nil
}

// pressShortcut parses a "cmd+shift+p" style string and dispatches it via
// osascript keystroke with the right modifiers.
func pressShortcut(shortcut string) error {
	parts := strings.Split(strings.ToLower(shortcut), "+")
	if len(parts) == 0 {
		return fmt.Errorf("empty shortcut")
	}
	key := strings.TrimSpace(parts[len(parts)-1])
	modifiers := parts[:len(parts)-1]

	modStr := buildModifiers(modifiers)

	// Use key codes for ALL keys to avoid keyboard layout issues (Arabic, etc.)
	if code, ok := specialKeyCodes[key]; ok {
		var script string
		if modStr != "" {
			script = fmt.Sprintf(`tell application "System Events" to key code %d using {%s}`, code, modStr)
		} else {
			script = fmt.Sprintf(`tell application "System Events" to key code %d`, code)
		}
		return runAppleScript(script)
	}

	// Single letter — use key code map
	if code, ok := letterKeyCodes[key]; ok {
		var script string
		if modStr != "" {
			script = fmt.Sprintf(`tell application "System Events" to key code %d using {%s}`, code, modStr)
		} else {
			script = fmt.Sprintf(`tell application "System Events" to key code %d`, code)
		}
		return runAppleScript(script)
	}

	return fmt.Errorf("unsupported palette key %q", key)
}

func buildModifiers(mods []string) string {
	var out []string
	for _, m := range mods {
		switch strings.TrimSpace(m) {
		case "cmd", "command":
			out = append(out, "command down")
		case "shift":
			out = append(out, "shift down")
		case "ctrl", "control":
			out = append(out, "control down")
		case "alt", "option", "opt":
			out = append(out, "option down")
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, ", ")
}

var specialKeyCodes = map[string]int{
	"return": 36, "enter": 36,
	"tab": 48, "space": 49, "escape": 53, "esc": 53,
	"delete": 51, "backspace": 51,
	"/": 44, "\\": 42,
	"up": 126, "down": 125, "left": 123, "right": 124,
	"-": 27, "=": 24, "[": 33, "]": 30,
	";": 41, "'": 39, ",": 43, ".": 47, "`": 50,
}

// letterKeyCodes maps letters and numbers to macOS virtual key codes.
// Layout-independent — works with Arabic, Chinese, or any input source.
var letterKeyCodes = map[string]int{
	"a": 0, "b": 11, "c": 8, "d": 2, "e": 14, "f": 3,
	"g": 5, "h": 4, "i": 34, "j": 38, "k": 40, "l": 37,
	"m": 46, "n": 45, "o": 31, "p": 35, "q": 12, "r": 15,
	"s": 1, "t": 17, "u": 32, "v": 9, "w": 13, "x": 7,
	"y": 16, "z": 6,
	"0": 29, "1": 18, "2": 19, "3": 20, "4": 21,
	"5": 23, "6": 22, "7": 26, "8": 28, "9": 25,
}

func runAppleScript(script string) error {
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func readClipboard() (string, bool) {
	out, err := exec.Command("pbpaste").Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}

func writeClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
