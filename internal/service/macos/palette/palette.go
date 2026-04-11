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

	if err := runAppleScript(`tell application "System Events" to keystroke "v" using command down`); err != nil {
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

	// Map special keys to key codes; letters go through keystroke.
	if code, ok := specialKeyCodes[key]; ok {
		script := fmt.Sprintf(
			`tell application "System Events" to key code %d using {%s}`,
			code, modStr,
		)
		return runAppleScript(script)
	}

	if len(key) != 1 {
		return fmt.Errorf("unsupported palette key %q", key)
	}
	escaped := strings.ReplaceAll(key, `"`, `\"`)
	script := fmt.Sprintf(
		`tell application "System Events" to keystroke "%s" using {%s}`,
		escaped, modStr,
	)
	return runAppleScript(script)
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
