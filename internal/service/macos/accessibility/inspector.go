package accessibility

import (
	"fmt"
	"os/exec"
	"strings"
)

// ListElements returns a compact summary of the AX tree of the currently
// frontmost application's main window. Each line is:
//
//	<indent>class | role | title | description | placeholder
//
// maxDepth caps recursion so very complex apps do not return 10 KB of text.
// A depth of 6–10 is usually enough to reveal whether an app exposes its
// content via AX at all — Electron apps come back with a tiny tree of only
// AXGroup containers, while native apps return hundreds of elements.
func ListElements(maxDepth int) (string, error) {
	if maxDepth <= 0 {
		maxDepth = 8
	}

	script := fmt.Sprintf(`
tell application "System Events"
    set frontApp to first process whose frontmost is true
    set frontName to name of frontApp
    tell frontApp
        try
            set w to window 1
        on error
            return "frontmost=" & frontName & " — no window"
        end try
        set out to "frontmost=" & frontName & linefeed
        set out to out & my dumpUI(w, 0, %d, "")
        return out
    end tell
end tell

on dumpUI(el, depth, maxDepth, indent)
    if depth > maxDepth then return ""
    set line to indent
    try
        set line to line & (class of el as string)
    on error
        set line to line & "?"
    end try
    try
        set line to line & " [" & (role of el as string) & "]"
    end try
    try
        set t to (title of el as string)
        if t is not "" and t is not missing value then
            set line to line & " title=" & t
        end if
    end try
    try
        set d to (description of el as string)
        if d is not "" and d is not missing value then
            set line to line & " desc=" & d
        end if
    end try
    try
        set p to ((value of attribute "AXPlaceholderValue" of el) as string)
        if p is not "" then set line to line & " placeholder=" & p
    end try
    set out to line & linefeed
    try
        set kids to UI elements of el
    on error
        return out
    end try
    repeat with k in kids
        set out to out & my dumpUI(k, depth + 1, maxDepth, indent & "  ")
    end repeat
    return out
end dumpUI`, maxDepth)

	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inspector: osascript: %w — %s", err, strings.TrimSpace(string(out)))
	}
	result := strings.TrimSpace(string(out))
	if len(result) > 4000 {
		result = result[:4000] + "\n...[truncated]"
	}
	return result, nil
}
