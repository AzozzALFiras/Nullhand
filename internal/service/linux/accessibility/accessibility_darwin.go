//go:build darwin

package accessibility

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ElementSummary is a compact description of a UI element returned by
// FindElements. Coordinates are in screen points (top-left origin).
type ElementSummary struct {
	Role        string
	Name        string
	Description string
	X, Y, W, H  int
}

// FocusField finds a text input field whose name or description contains
// labelHint (case-insensitive) and clicks it to give it focus.
//
// Requires Accessibility permission in System Settings → Privacy & Security
// → Accessibility for whatever process invokes osascript (Terminal, the
// Nullhand binary itself, etc.).
func FocusField(appName, labelHint string) error {
	if labelHint == "" {
		return fmt.Errorf("accessibility: label is required")
	}

	// AppleScript: walk the UI elements of the frontmost process, find a text
	// field/text area whose attributes contain the needle, click it.
	script := fmt.Sprintf(`
on run
	set needle to %q
	tell application "System Events"
		set frontProc to first application process whose frontmost is true
		set procName to name of frontProc
		try
			set found to my findField(frontProc, needle)
			if found is missing value then
				return "not found:" & procName
			end if
			-- Click the element via its position + size.
			tell found
				try
					set {x, y} to position
					set {w, h} to size
					set cx to x + (w div 2)
					set cy to y + (h div 2)
					-- Use mouse click for reliability.
					tell application "System Events" to click at {cx, cy}
					return "ok:" & procName
				on error errMsg
					return "not found:" & procName & " (click failed: " & errMsg & ")"
				end try
			end tell
		on error errMsg
			return "not found:" & procName & " (" & errMsg & ")"
		end try
	end tell
end run

on findField(parent, needle)
	tell application "System Events"
		try
			repeat with el in (entire contents of parent)
				try
					set r to role of el
					if r is in {"AXTextField", "AXTextArea", "AXSearchField", "AXComboBox"} then
						set n to ""
						set d to ""
						try
							set n to value of attribute "AXDescription" of el as string
						end try
						try
							set d to value of attribute "AXHelp" of el as string
						end try
						set t to ""
						try
							set t to title of el as string
						end try
						set hay to (n & " " & d & " " & t)
						ignoring case
							if hay contains needle then
								return el
							end if
						end ignoring
					end if
				end try
			end repeat
		end try
	end tell
	return missing value
end findField
`, labelHint)
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("accessibility: osascript: %w — %s", err, strings.TrimSpace(string(out)))
	}
	result := strings.TrimSpace(string(out))
	switch {
	case strings.HasPrefix(result, "ok:"):
		return nil
	case strings.HasPrefix(result, "not found:"):
		return fmt.Errorf("accessibility: no field matching %q in %s", labelHint, strings.TrimPrefix(result, "not found:"))
	default:
		return fmt.Errorf("accessibility: unexpected result: %s", result)
	}
}

// Click finds a UI element whose name (or description, or title) exactly
// matches label and clicks it. Use ClickFuzzy for substring/case-insensitive.
func Click(appName, label string) error {
	if label == "" {
		return fmt.Errorf("accessibility: label is required")
	}
	return clickWithMatcher(label, true /*exact*/)
}

// ClickFuzzy clicks a UI element matching label as a case-insensitive
// substring of its name, description, or title.
func ClickFuzzy(appName, label string) error {
	if label == "" {
		return fmt.Errorf("accessibility: label is required")
	}
	return clickWithMatcher(label, false /*fuzzy*/)
}

// clickWithMatcher implements both Click and ClickFuzzy.
func clickWithMatcher(label string, exact bool) error {
	matchClause := `if hay is needle then return el`
	if !exact {
		matchClause = `ignoring case
				if hay contains needle then return el
			end ignoring`
	}

	script := fmt.Sprintf(`
on run
	set needle to %q
	tell application "System Events"
		set frontProc to first application process whose frontmost is true
		set procName to name of frontProc
		try
			set found to my findElement(frontProc, needle)
			if found is missing value then
				return "not found:" & procName
			end if
			tell found
				try
					-- Try AXPress action first (works for buttons, links).
					perform action "AXPress"
					return "ok:" & procName
				on error
					try
						set {x, y} to position
						set {w, h} to size
						set cx to x + (w div 2)
						set cy to y + (h div 2)
						tell application "System Events" to click at {cx, cy}
						return "ok:" & procName
					on error errMsg
						return "not found:" & procName & " (click fallback failed: " & errMsg & ")"
					end try
				end try
			end tell
		on error errMsg
			return "not found:" & procName & " (" & errMsg & ")"
		end try
	end tell
end run

on findElement(parent, needle)
	tell application "System Events"
		try
			repeat with el in (entire contents of parent)
				try
					set hay to ""
					try
						set hay to hay & (title of el as string) & " "
					end try
					try
						set hay to hay & (value of attribute "AXDescription" of el as string) & " "
					end try
					try
						set hay to hay & (value of attribute "AXHelp" of el as string) & " "
					end try
					try
						set hay to hay & (name of el as string) & " "
					end try
					%s
				end try
			end repeat
		end try
	end tell
	return missing value
end findElement
`, label, matchClause)

	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("accessibility: osascript: %w — %s", err, strings.TrimSpace(string(out)))
	}
	result := strings.TrimSpace(string(out))
	switch {
	case strings.HasPrefix(result, "ok:"):
		return nil
	case strings.HasPrefix(result, "not found:"):
		return fmt.Errorf("accessibility: no element matching %q in %s", label, strings.TrimPrefix(result, "not found:"))
	default:
		return fmt.Errorf("accessibility: unexpected result: %s", result)
	}
}

// WaitForElement polls every 250ms for an element matching label, up to
// timeoutMs. Match is case-insensitive substring on title/description/help/name.
func WaitForElement(appName, label string, timeoutMs int) error {
	if label == "" {
		return fmt.Errorf("accessibility: label is required")
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for {
		summaries, err := FindElements(appName, label, 1)
		if err == nil && len(summaries) > 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("accessibility: timeout after %dms waiting for %q", timeoutMs, label)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// FindElements returns up to max UI element summaries whose title, name,
// description, or help text contains label (case-insensitive substring).
func FindElements(appName, label string, max int) ([]ElementSummary, error) {
	if label == "" {
		return nil, fmt.Errorf("accessibility: label is required")
	}
	if max <= 0 {
		max = 10
	}

	script := fmt.Sprintf(`
on run
	set needle to %q
	set lim to %d
	set out to ""
	tell application "System Events"
		try
			set frontProc to first application process whose frontmost is true
		on error
			return "ERR:no frontmost process"
		end try
		try
			set count_ to 0
			repeat with el in (entire contents of frontProc)
				if count_ >= lim then exit repeat
				try
					set hay to ""
					set roleStr to ""
					set nameStr to ""
					set descStr to ""
					try
						set roleStr to role of el as string
					end try
					try
						set nameStr to title of el as string
					end try
					try
						set descStr to value of attribute "AXDescription" of el as string
					end try
					if descStr is "" then
						try
							set descStr to value of attribute "AXHelp" of el as string
						end try
					end if
					set hay to nameStr & " " & descStr
					ignoring case
						if hay contains needle then
							set x_ to 0
							set y_ to 0
							set w_ to 0
							set h_ to 0
							try
								set {x_, y_} to position of el
								set {w_, h_} to size of el
							end try
							set out to out & roleStr & "|" & nameStr & "|" & descStr & "|" & x_ & "|" & y_ & "|" & w_ & "|" & h_ & linefeed
							set count_ to count_ + 1
						end if
					end ignoring
				end try
			end repeat
		end try
	end tell
	if out is "" then
		return "EMPTY"
	end if
	return out
end run
`, label, max)

	rawOut, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("accessibility: osascript: %w — %s", err, strings.TrimSpace(string(rawOut)))
	}
	result := strings.TrimSpace(string(rawOut))
	if strings.HasPrefix(result, "ERR:") {
		return nil, fmt.Errorf("accessibility: %s", strings.TrimPrefix(result, "ERR:"))
	}
	if result == "EMPTY" || result == "" {
		return nil, nil
	}
	var out []ElementSummary
	for _, line := range strings.Split(result, "\n") {
		fields := strings.Split(line, "|")
		if len(fields) < 7 {
			continue
		}
		var x, y, w, h int
		fmt.Sscanf(fields[3], "%d", &x)
		fmt.Sscanf(fields[4], "%d", &y)
		fmt.Sscanf(fields[5], "%d", &w)
		fmt.Sscanf(fields[6], "%d", &h)
		out = append(out, ElementSummary{
			Role:        fields[0],
			Name:        fields[1],
			Description: fields[2],
			X:           x, Y: y, W: w, H: h,
		})
	}
	return out, nil
}
