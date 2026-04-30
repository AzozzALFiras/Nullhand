//go:build darwin

package accessibility

import (
	"fmt"
	"os/exec"
	"strings"
)

// ListElements dumps a compact summary of the macOS Accessibility tree of the
// frontmost application. Each line is:
//
//	<indent>role | name | description
//
// maxDepth caps recursion. AppleScript's "entire contents" gives a flat list
// of every UI element in the frontmost process — we don't have explicit depth
// info, so we cap by total line count instead.
//
// Requires Accessibility permission in System Settings → Privacy & Security
// → Accessibility for the bot's executing process.
func ListElements(maxDepth int) (string, error) {
	if maxDepth <= 0 {
		maxDepth = 8
	}
	// Translate "depth" to a soft line cap: each depth level might have a few
	// dozen elements. 8 → ~200 lines, 4 → ~50 lines, etc.
	cap := maxDepth * 25
	if cap > 400 {
		cap = 400
	}

	script := fmt.Sprintf(`
on run
	set lim to %d
	set out to ""
	tell application "System Events"
		try
			set frontProc to first application process whose frontmost is true
			set procName to name of frontProc
		on error
			return "frontmost=unknown — accessibility unavailable"
		end try
		set out to "frontmost=" & procName & linefeed
		try
			set n to 0
			repeat with el in (entire contents of frontProc)
				if n >= lim then
					set out to out & "...[truncated at " & lim & " elements]" & linefeed
					exit repeat
				end if
				try
					set roleStr to ""
					set nameStr to ""
					set descStr to ""
					try
						set roleStr to role of el as string
					end try
					try
						set nameStr to (title of el as string)
					end try
					try
						set descStr to (value of attribute "AXDescription" of el as string)
					end try
					if (nameStr is "") and (descStr is "") and (roleStr is "AXGroup" or roleStr is "AXUnknown") then
						-- Skip noisy empty containers.
					else
						set parts to roleStr
						if nameStr is not "" then set parts to parts & " | name=" & nameStr
						if descStr is not "" then set parts to parts & " | desc=" & descStr
						set out to out & parts & linefeed
						set n to n + 1
					end if
				end try
			end repeat
		end try
	end tell
	return out
end run
`, cap)

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
