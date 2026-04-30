//go:build darwin

package keyboard

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Type pastes text into the focused element via the clipboard, preserving the
// previous clipboard contents. Uses pbcopy + Cmd+V — Unicode-safe and
// keyboard-layout independent (handles Arabic, emoji, Chinese, etc.).
func Type(text string) error {
	prev, _, err := TypeAndHold(text)
	if err != nil {
		return err
	}
	RestoreClipboard(prev)
	return nil
}

// TypeAndHold pastes text without restoring the clipboard.
// Returns: (previousClipboard, hadPrevious, error).
func TypeAndHold(text string) (string, bool, error) {
	if text == "" {
		return "", false, nil
	}
	prev, hadPrev := readClipboard()

	if err := writeClipboard(text); err != nil {
		return prev, hadPrev, fmt.Errorf("keyboard: set clipboard: %w", err)
	}
	time.Sleep(60 * time.Millisecond)

	// Cmd+V — keystroke "v" using {command down}.
	if err := osascriptKeystroke("v", []string{"command"}); err != nil {
		return prev, hadPrev, fmt.Errorf("keyboard: paste: %w", err)
	}
	time.Sleep(180 * time.Millisecond)
	return prev, hadPrev, nil
}

// RestoreClipboard writes prev back to the clipboard. No-op if empty.
func RestoreClipboard(prev string) {
	if prev == "" {
		return
	}
	_ = writeClipboard(prev)
}

// ReadClipboard returns the current pasteboard text (trailing newline trimmed).
func ReadClipboard() string {
	s, _ := readClipboard()
	return strings.TrimRight(s, "\n")
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

// PressKey presses a key or modifier+key shortcut on macOS.
// Examples: "return", "cmd+t", "cmd+shift+5", "escape", "tab", "f5".
// Modifier mapping: ctrl/control → control; cmd/command → command; option/alt → option.
func PressKey(shortcut string) error {
	parts := strings.Split(strings.ToLower(shortcut), "+")
	key := parts[len(parts)-1]
	modifiers := parts[:len(parts)-1]

	// Map of friendly key names to macOS key codes (System Events).
	// Letters/digits use `keystroke` directly; non-typing keys use `key code`.
	if code, ok := keyCodeMap[key]; ok {
		return osascriptKeyCode(code, normalizeModifiers(modifiers))
	}
	// Single character → keystroke. Single special punctuation also goes via
	// keystroke in most cases.
	return osascriptKeystroke(key, normalizeModifiers(modifiers))
}

// normalizeModifiers maps Linux/macOS modifier aliases to AppleScript names.
func normalizeModifiers(mods []string) []string {
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
		case "fn":
			// AppleScript can't directly synthesize fn. Skip silently.
		}
	}
	return out
}

// osascriptKeystroke sends a keystroke for printable characters.
func osascriptKeystroke(char string, mods []string) error {
	modClause := ""
	if len(mods) > 0 {
		mc := make([]string, 0, len(mods))
		for _, m := range mods {
			mc = append(mc, m+" down")
		}
		modClause = " using {" + strings.Join(mc, ", ") + "}"
	}
	// Quote the character for AppleScript.
	escaped := strings.ReplaceAll(char, "\"", "\\\"")
	script := fmt.Sprintf(`tell application "System Events" to keystroke "%s"%s`, escaped, modClause)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript keystroke %q: %w — %s", char, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// osascriptKeyCode sends a key code event for non-printable keys (F-keys,
// arrows, return, escape, etc.).
func osascriptKeyCode(code int, mods []string) error {
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
		return fmt.Errorf("osascript key code %d: %w — %s", code, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ClearField selects all (Cmd+A) and deletes the contents of the focused field.
func ClearField() error {
	if err := osascriptKeystroke("a", []string{"command"}); err != nil {
		return fmt.Errorf("clear field (select all): %w", err)
	}
	time.Sleep(60 * time.Millisecond)
	if err := osascriptKeyCode(51, nil); err != nil { // 51 = Delete
		return fmt.Errorf("clear field (delete): %w", err)
	}
	time.Sleep(60 * time.Millisecond)
	return nil
}

// SelectAllAndType clears the focused field then types text.
func SelectAllAndType(text string) error {
	if err := ClearField(); err != nil {
		return err
	}
	return Type(text)
}

// keyCodeMap maps friendly key names to macOS virtual key codes.
// Reference: https://eastmanreference.com/complete-list-of-applescript-key-codes
var keyCodeMap = map[string]int{
	"return": 36, "enter": 36,
	"tab": 48, "space": 49,
	"escape": 53, "esc": 53,
	"delete": 51, "backspace": 51,
	"forward_delete": 117, "del": 117,
	"left": 123, "right": 124, "down": 125, "up": 126,
	"home": 115, "end": 119,
	"pageup": 116, "pagedown": 121,
	"f1": 122, "f2": 120, "f3": 99, "f4": 118,
	"f5": 96, "f6": 97, "f7": 98, "f8": 100,
	"f9": 101, "f10": 109, "f11": 103, "f12": 111,
	"f13": 105, "f14": 107, "f15": 113,
}
