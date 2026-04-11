package accessibility

import (
	"fmt"
	"os/exec"
	"strings"

	appsvc "github.com/AzozzALFiras/nullhand/internal/service/macos/apps"
)

// FocusField searches the FRONTMOST application's frontmost window for a
// text input field (text field, text area, or combo box) whose placeholder,
// help, description, or title contains the labelHint substring, and clicks
// it to give it keyboard focus. Case-insensitive substring match.
//
// appName is only used for error messages — the actual search always targets
// whatever process is currently frontmost. This is intentional: macOS process
// names often differ from the user-facing app name (e.g. "Visual Studio Code"
// runs as process "Code"), and callers are expected to have already called
// open_app to bring the target app to the front.
//
// Requires macOS Accessibility permission.
func FocusField(appName, labelHint string) error {
	if labelHint == "" {
		return fmt.Errorf("accessibility: label is required")
	}

	// Guard: focus_text_field only works for native Cocoa apps. Electron apps
	// expose an empty AX tree and will always return "not found", wasting
	// 10+ seconds per attempt. Self-correct immediately.
	if appName != "" && !appsvc.IsNativeAX(appName) {
		return fmt.Errorf("accessibility: %q is not a native AX app — use focus_via_palette or run_recipe instead", appName)
	}

	script := fmt.Sprintf(`
tell application "System Events"
    set frontApp to first process whose frontmost is true
    set frontName to name of frontApp
    tell frontApp
        try
            set allElements to entire contents of window 1
        on error
            return "no window:" & frontName
        end try
        set needle to %q
        set needleLC to my toLower(needle)
        repeat with el in allElements
            try
                set cls to (class of el) as string
            on error
                set cls to ""
            end try
            if cls is "text field" or cls is "text area" or cls is "combo box" then
                set haystacks to {}
                try
                    set end of haystacks to ((value of attribute "AXPlaceholderValue" of el) as string)
                end try
                try
                    set end of haystacks to ((description of el) as string)
                end try
                try
                    set end of haystacks to ((title of el) as string)
                end try
                try
                    set end of haystacks to ((name of el) as string)
                end try
                try
                    set end of haystacks to ((help of el) as string)
                end try
                repeat with h in haystacks
                    try
                        if my toLower(h as string) contains needleLC then
                            click el
                            return "ok:" & frontName
                        end if
                    end try
                end repeat
            end if
        end repeat
        return "not found:" & frontName
    end tell
end tell

on toLower(s)
    set out to ""
    repeat with c in (s as string)
        set code to id of c
        if code >= 65 and code <= 90 then
            set out to out & (character id (code + 32))
        else
            set out to out & c
        end if
    end repeat
    return out
end toLower`, labelHint)

	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("accessibility: focus osascript failed: %w — %s", err, strings.TrimSpace(string(out)))
	}

	result := strings.TrimSpace(string(out))
	switch {
	case strings.HasPrefix(result, "ok:"):
		return nil
	case strings.HasPrefix(result, "no window:"):
		frontName := strings.TrimPrefix(result, "no window:")
		return fmt.Errorf("accessibility: frontmost app %q has no open window (expected %q)", frontName, appName)
	case strings.HasPrefix(result, "not found:"):
		frontName := strings.TrimPrefix(result, "not found:")
		return fmt.Errorf("accessibility: no text field matching %q in %q", labelHint, frontName)
	default:
		return fmt.Errorf("accessibility: unexpected focus result: %s", result)
	}
}

// Click searches the FRONTMOST application's frontmost window for a UI
// element whose name, title, or description matches label, and clicks it.
//
// appName is only used for error messages — the actual search targets the
// currently frontmost process. Callers must have already focused the target
// app via open_app before calling this.
//
// Requires macOS Accessibility permission.
func Click(appName, label string) error {
	if label == "" {
		return fmt.Errorf("accessibility: label is required")
	}

	script := fmt.Sprintf(`
tell application "System Events"
    set frontApp to first process whose frontmost is true
    set frontName to name of frontApp
    tell frontApp
        try
            set allElements to entire contents of window 1
        on error
            return "no window:" & frontName
        end try
        repeat with el in allElements
            try
                if (name of el) is %q then
                    click el
                    return "ok:" & frontName
                end if
            end try
            try
                if (title of el) is %q then
                    click el
                    return "ok:" & frontName
                end if
            end try
            try
                if (description of el) is %q then
                    click el
                    return "ok:" & frontName
                end if
            end try
        end repeat
        return "not found:" & frontName
    end tell
end tell`, label, label, label)

	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("accessibility: osascript failed: %w — %s", err, strings.TrimSpace(string(out)))
	}

	result := strings.TrimSpace(string(out))
	switch {
	case strings.HasPrefix(result, "ok:"):
		return nil
	case strings.HasPrefix(result, "no window:"):
		frontName := strings.TrimPrefix(result, "no window:")
		return fmt.Errorf("accessibility: frontmost app %q has no open window (expected %q)", frontName, appName)
	case strings.HasPrefix(result, "not found:"):
		frontName := strings.TrimPrefix(result, "not found:")
		return fmt.Errorf("accessibility: no element named %q found in %q", label, frontName)
	default:
		return fmt.Errorf("accessibility: unexpected result: %s", result)
	}
}
