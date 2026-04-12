package keyboard

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Type inserts text into the frontmost application via clipboard paste.
// It saves the existing clipboard first and restores it before returning.
// Use this for one-shot typing where no verification is needed.
//
// We deliberately avoid osascript's `keystroke` for text entry because it
// translates each character through the current macOS input source. If the
// user's input source is Arabic (or any non-US layout), typing ASCII like
// "hello" produces gibberish like "ش شششش".
//
// Clipboard + Cmd+V inserts the exact Unicode text regardless of layout.
func Type(text string) error {
	prev, _, err := TypeAndHold(text)
	if err != nil {
		return err
	}
	RestoreClipboard(prev)
	return nil
}

// TypeAndHold pastes text without restoring the clipboard afterwards.
// It returns the previous clipboard contents so the caller can restore them
// later (after verification, for example). This is the building block the
// agent verification loop uses: paste → verify_clipboard → restore_clipboard.
//
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

	// Use key code 9 (V) instead of keystroke "v" — keystroke breaks when
	// the macOS keyboard layout is non-Latin (Arabic, Chinese, etc.)
	if err := runAppleScript(`tell application "System Events" to key code 9 using command down`); err != nil {
		return prev, hadPrev, fmt.Errorf("keyboard: paste: %w", err)
	}
	time.Sleep(180 * time.Millisecond)
	return prev, hadPrev, nil
}

// RestoreClipboard writes the given text back to the clipboard. If text is
// empty this is a no-op. Best-effort: errors are swallowed because they
// should not fail the overall operation.
func RestoreClipboard(prev string) {
	if prev == "" {
		return
	}
	_ = writeClipboard(prev)
}

// ReadClipboard returns the current clipboard text (trimmed trailing newline).
// Used by the agent verification loop.
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

// PressKey presses a key or modifier+key shortcut.
// Examples: "enter", "cmd+t", "cmd+shift+5", "escape", "tab", "f5"
func PressKey(shortcut string) error {
	parts := strings.Split(strings.ToLower(shortcut), "+")
	key := parts[len(parts)-1]
	modifiers := parts[:len(parts)-1]

	// Map readable key names to osascript key code or keystroke equivalents.
	keyCode, isCode := keyCodeMap[key]

	modStr := buildModifiers(modifiers)

	var script string
	if isCode {
		if modStr != "" {
			script = fmt.Sprintf(`tell application "System Events" to key code %d using {%s}`, keyCode, modStr)
		} else {
			script = fmt.Sprintf(`tell application "System Events" to key code %d`, keyCode)
		}
	} else {
		escaped := strings.ReplaceAll(key, `"`, `\"`)
		if modStr != "" {
			script = fmt.Sprintf(`tell application "System Events" to keystroke "%s" using {%s}`, escaped, modStr)
		} else {
			script = fmt.Sprintf(`tell application "System Events" to keystroke "%s"`, escaped)
		}
	}

	return runAppleScript(script)
}

// buildModifiers converts ["cmd","shift"] → `command down, shift down`.
func buildModifiers(mods []string) string {
	var out []string
	for _, m := range mods {
		switch m {
		case "cmd", "command":
			out = append(out, "command down")
		case "shift":
			out = append(out, "shift down")
		case "ctrl", "control":
			out = append(out, "control down")
		case "alt", "option":
			out = append(out, "option down")
		}
	}
	return strings.Join(out, ", ")
}

// keyCodeMap maps readable key names to macOS virtual key codes.
var keyCodeMap = map[string]int{
	"enter":     36,
	"return":    36,
	"escape":    53,
	"tab":       48,
	"space":     49,
	"delete":    51,
	"backspace": 51,
	"up":        126,
	"down":      125,
	"left":      123,
	"right":     124,
	"f1":        122, "f2": 120, "f3": 99, "f4": 118,
	"f5": 96, "f6": 97, "f7": 98, "f8": 100,
	"f9": 101, "f10": 109, "f11": 103, "f12": 111,
}

func runAppleScript(script string) error {
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript error: %w — %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
