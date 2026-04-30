//go:build darwin

package palette

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Run opens the command palette by pressing the given keyboard shortcut,
// pastes the command name via the clipboard (Unicode-safe), and presses
// Return to execute it. macOS analog of palette_linux.go.
//
// Examples:
//
//	Run("cmd+shift+p", "Claude Code: Focus Chat")   // VS Code / Cursor
//	Run("cmd+k", "jump to channel #general")        // Slack / Discord
//	Run("cmd+p", "quick find")                      // Notion / Obsidian
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

	// Cmd+V paste — layout-independent.
	if err := osaKeystrokeWithMod("v", "command"); err != nil {
		return fmt.Errorf("palette: paste command: %w", err)
	}
	time.Sleep(150 * time.Millisecond)

	if err := osaKeyCode(36, nil); err != nil { // 36 = Return
		return fmt.Errorf("palette: press return: %w", err)
	}

	time.Sleep(250 * time.Millisecond)

	if hadPrev {
		_ = writeClipboard(prev)
	}
	return nil
}

// pressShortcut parses "cmd+shift+p" and sends it via osascript System Events.
func pressShortcut(shortcut string) error {
	parts := strings.Split(strings.ToLower(shortcut), "+")
	if len(parts) == 0 {
		return fmt.Errorf("empty shortcut")
	}
	key := strings.TrimSpace(parts[len(parts)-1])
	modifiers := parts[:len(parts)-1]

	mods := normalizeMods(modifiers)
	if code, ok := keyCodeMap[key]; ok {
		return osaKeyCode(code, mods)
	}
	return osaKeystroke(key, mods)
}

func normalizeMods(mods []string) []string {
	var out []string
	for _, m := range mods {
		switch strings.TrimSpace(m) {
		case "cmd", "command":
			out = append(out, "command")
		case "ctrl", "control":
			out = append(out, "control")
		case "alt", "option", "opt":
			out = append(out, "option")
		case "shift":
			out = append(out, "shift")
		}
	}
	return out
}

func osaKeystrokeWithMod(char, mod string) error {
	return osaKeystroke(char, []string{mod})
}

func osaKeystroke(char string, mods []string) error {
	modClause := ""
	if len(mods) > 0 {
		mc := make([]string, 0, len(mods))
		for _, m := range mods {
			mc = append(mc, m+" down")
		}
		modClause = " using {" + strings.Join(mc, ", ") + "}"
	}
	escaped := strings.ReplaceAll(char, "\"", "\\\"")
	script := fmt.Sprintf(`tell application "System Events" to keystroke "%s"%s`, escaped, modClause)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript keystroke: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func osaKeyCode(code int, mods []string) error {
	modClause := ""
	if len(mods) > 0 {
		mc := make([]string, 0, len(mods))
		for _, m := range mods {
			mc = append(mc, m+" down")
		}
		modClause = " using {" + strings.Join(mc, ", ") + "}"
	}
	script := fmt.Sprintf(`tell application "System Events" to key code %d%s`, code, modClause)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript key code: %w — %s", err, strings.TrimSpace(string(out)))
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

// keyCodeMap maps key names to macOS virtual key codes.
var keyCodeMap = map[string]int{
	"return": 36, "enter": 36,
	"tab": 48, "space": 49,
	"escape": 53, "esc": 53,
	"delete": 51, "backspace": 51,
	"left": 123, "right": 124, "down": 125, "up": 126,
	"home": 115, "end": 119,
	"pageup": 116, "pagedown": 121,
	"f1": 122, "f2": 120, "f3": 99, "f4": 118,
	"f5": 96, "f6": 97, "f7": 98, "f8": 100,
	"f9": 101, "f10": 109, "f11": 103, "f12": 111,
}
