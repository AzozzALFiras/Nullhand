//go:build linux

package keyboard

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Type inserts text into the frontmost application via clipboard paste.
// It saves the existing clipboard first and restores it before returning.
//
// We use clipboard + Ctrl+V rather than xdotool type because xdotool type
// can mishandle non-Latin input sources (Arabic, Chinese, emoji). The
// clipboard approach inserts exact Unicode regardless of active layout.
func Type(text string) error {
	prev, _, err := TypeAndHold(text)
	if err != nil {
		return err
	}
	RestoreClipboard(prev)
	return nil
}

// TypeAndHold pastes text without restoring the clipboard afterwards.
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

	if err := xdotoolKey("ctrl+v"); err != nil {
		return prev, hadPrev, fmt.Errorf("keyboard: paste: %w", err)
	}
	time.Sleep(180 * time.Millisecond)
	return prev, hadPrev, nil
}

// RestoreClipboard writes text back to the clipboard. No-op if text is empty.
func RestoreClipboard(prev string) {
	if prev == "" {
		return
	}
	_ = writeClipboard(prev)
}

// ReadClipboard returns the current clipboard text (trailing newline trimmed).
func ReadClipboard() string {
	s, _ := readClipboard()
	return strings.TrimRight(s, "\n")
}

func readClipboard() (string, bool) {
	cmd := exec.Command("xclip", "-selection", "clipboard", "-o")
	cmd.Env = append(os.Environ(), "DISPLAY=:0")
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}

func writeClipboard(text string) error {
	cmd := exec.Command("xclip", "-selection", "clipboard")
	cmd.Env = append(os.Environ(), "DISPLAY=:0")
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("xclip: %w (is xclip installed? sudo apt install xclip)", err)
	}
	return nil
}

// PressKey presses a key or modifier+key shortcut.
// Examples: "enter", "ctrl+t", "ctrl+shift+5", "escape", "tab", "f5".
// Modifier mapping: cmd/command → ctrl, option → alt.
func PressKey(shortcut string) error {
	parts := strings.Split(strings.ToLower(shortcut), "+")
	key := parts[len(parts)-1]
	modifiers := parts[:len(parts)-1]

	xkey, known := keyNameMap[key]
	if !known {
		xkey = key // pass through letters, numbers, bare punctuation
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

// buildModifiers converts ["ctrl","shift"] → "ctrl+shift", mapping macOS
// modifier names to xdotool equivalents.
func buildModifiers(mods []string) string {
	var out []string
	for _, m := range mods {
		switch m {
		case "cmd", "command":
			out = append(out, "ctrl")
		case "shift":
			out = append(out, "shift")
		case "ctrl", "control":
			out = append(out, "ctrl")
		case "alt", "option":
			out = append(out, "alt")
		case "super", "win":
			out = append(out, "super")
		}
	}
	return strings.Join(out, "+")
}

// keyNameMap maps readable key names to X11/xdotool key names.
// Letters, numbers, and simple punctuation are passed through unchanged.
var keyNameMap = map[string]string{
	// Navigation & control
	"enter": "Return", "return": "Return",
	"escape": "Escape", "esc": "Escape",
	"tab": "Tab", "space": "space",
	"delete": "BackSpace", "backspace": "BackSpace",
	"del": "Delete", "forward_delete": "Delete",
	"up": "Up", "down": "Down", "left": "Left", "right": "Right",
	"home": "Home", "end": "End",
	"pageup": "Prior", "pagedown": "Next",
	// Function keys
	"f1": "F1", "f2": "F2", "f3": "F3", "f4": "F4",
	"f5": "F5", "f6": "F6", "f7": "F7", "f8": "F8",
	"f9": "F9", "f10": "F10", "f11": "F11", "f12": "F12",
	// Punctuation that xdotool names explicitly
	"-": "minus", "=": "equal",
	"[": "bracketleft", "]": "bracketright",
	";": "semicolon", "'": "apostrophe",
	",": "comma", ".": "period", "/": "slash",
	"`": "grave", "\\": "backslash",
}

// xdotoolKey sends a key combo via xdotool key (e.g. "ctrl+v", "Return").
func xdotoolKey(combo string) error {
	cmd := exec.Command("xdotool", "key", combo)
	cmd.Env = append(os.Environ(), "DISPLAY=:0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("xdotool key error: %w — %s (is xdotool installed? sudo apt install xdotool)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
