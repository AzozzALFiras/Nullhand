// Package mac implements the macOS-only desktop features: Shortcuts, audio
// and media control, Spotlight search with Quick Look previews, power and
// appearance control, and on-screen alerts.
//
// The real implementations live in the *_darwin.go files; mac_other.go returns
// ErrUnsupported everywhere else. That split lets the shared viewmodels call
// these functions on Linux too and answer with a clear "macOS only" message
// instead of failing to build. All parsing and argument building lives here,
// without a build tag, so it stays testable on every platform.
package mac

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ErrUnsupported is returned by every call in this package on non-macOS builds.
var ErrUnsupported = errors.New("this feature is macOS-only")

// ErrNoMediaApp is returned when neither Music nor Spotify is running, so
// there is nothing to control or report.
var ErrNoMediaApp = errors.New("no media app is running (open Music or Spotify first)")

// maxSpokenChars bounds /say input: `say` blocks until it finishes speaking,
// and nobody wants the Mac reading out a pasted document.
const maxSpokenChars = 500

// VolumeState is the system output volume and mute flag.
type VolumeState struct {
	Output int // 0-100
	Muted  bool
}

// Summary renders the volume for a chat reply.
func (v VolumeState) Summary() string {
	if v.Muted {
		return fmt.Sprintf("🔇 Muted (volume %d%%)", v.Output)
	}
	return fmt.Sprintf("🔊 Volume %d%%", v.Output)
}

// Battery is the state reported by `pmset -g batt`.
type Battery struct {
	Present  bool   // false on desktop Macs
	Percent  int    // 0-100
	State    string // "charging", "discharging", "charged", "AC attached", ...
	Source   string // "AC Power" or "Battery Power"
	TimeLeft string // "3:21", empty when pmset is still calculating
}

// Summary renders the battery state for a chat reply.
func (b Battery) Summary() string {
	if !b.Present {
		return fmt.Sprintf("🔌 No battery — running on %s", b.Source)
	}
	icon := "🔋"
	if b.Percent <= 20 && b.State == "discharging" {
		icon = "🪫"
	}
	out := fmt.Sprintf("%s %d%% — %s", icon, b.Percent, b.State)
	if b.TimeLeft != "" {
		out += fmt.Sprintf(", %s remaining", b.TimeLeft)
	}
	return out
}

// NowPlaying is the current track of the media app that is running.
type NowPlaying struct {
	App    string // "Music" or "Spotify"
	State  string // "playing", "paused", "stopped"
	Title  string
	Artist string
	Album  string
}

// Summary renders the track for a chat reply.
func (n NowPlaying) Summary() string {
	icon := "⏸"
	if n.State == "playing" {
		icon = "▶️"
	}
	line := fmt.Sprintf("%s %s", icon, n.Title)
	if n.Artist != "" {
		line += " — " + n.Artist
	}
	if n.Album != "" {
		line += fmt.Sprintf("\n💿 %s", n.Album)
	}
	return line + fmt.Sprintf("\n(%s, %s)", n.App, n.State)
}

// QuoteAppleScript renders s as an AppleScript string literal, surrounding
// quotes included. Escaping matters beyond breaking the script: a label or
// filename carrying a quote would otherwise end the literal and let the rest
// of the text run as AppleScript statements.
func QuoteAppleScript(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// clampPercent keeps a volume request inside 0-100.
func clampPercent(p int) int {
	switch {
	case p < 0:
		return 0
	case p > 100:
		return 100
	default:
		return p
	}
}

// parseVolumeSettings reads the output of AppleScript's `get volume settings`:
//
//	output volume:44, input volume:30, alert volume:100, output muted:false
func parseVolumeSettings(out string) (VolumeState, error) {
	fields := map[string]string{}
	for _, part := range strings.Split(out, ",") {
		k, v, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		fields[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}

	raw, ok := fields["output volume"]
	if !ok {
		return VolumeState{}, fmt.Errorf("unexpected volume output %q", strings.TrimSpace(out))
	}
	// AppleScript reports "missing value" when the active output device has no
	// software volume control (some external DACs and HDMI displays).
	level, err := strconv.Atoi(raw)
	if err != nil {
		return VolumeState{}, fmt.Errorf("the current audio output device has no software volume control")
	}
	return VolumeState{Output: clampPercent(level), Muted: fields["output muted"] == "true"}, nil
}

// parseBattery reads the output of `pmset -g batt`:
//
//	Now drawing from 'AC Power'
//	 -InternalBattery-0 (id=21823587)	95%; AC attached; not charging present: true
func parseBattery(out string) (Battery, error) {
	var b Battery
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 || lines[0] == "" {
		return b, fmt.Errorf("pmset returned no battery information")
	}

	if _, source, ok := strings.Cut(lines[0], "drawing from "); ok {
		b.Source = strings.Trim(strings.TrimSpace(source), "'")
	}

	for _, line := range lines[1:] {
		if !strings.Contains(line, "%") {
			continue
		}
		// "	95%; AC attached; not charging present: true"
		fields := strings.Split(line, ";")
		pct := strings.TrimSpace(fields[0])
		if idx := strings.LastIndex(pct, "\t"); idx >= 0 {
			pct = pct[idx+1:]
		}
		n, err := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(pct), "%"))
		if err != nil {
			continue
		}
		b.Present = true
		b.Percent = clampPercent(n)
		if len(fields) > 1 {
			b.State = strings.TrimSpace(fields[1])
		}
		if len(fields) > 2 {
			// "3:21 remaining present: true" / "no estimate present: true"
			rest := strings.TrimSpace(fields[2])
			if t, _, ok := strings.Cut(rest, " remaining"); ok {
				b.TimeLeft = strings.TrimSpace(t)
			}
		}
		break
	}

	if !b.Present && b.Source == "" {
		return b, fmt.Errorf("could not parse pmset output %q", strings.TrimSpace(out))
	}
	return b, nil
}

// parseShortcutNames turns `shortcuts list` output into names, dropping the
// blank lines the CLI emits around the list.
func parseShortcutNames(out string) []string {
	var names []string
	for _, line := range strings.Split(out, "\n") {
		if name := strings.TrimSpace(line); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// mediaActions maps what users type (in both languages, via the intent layer)
// to the AppleScript command that both Music and Spotify understand.
var mediaActions = map[string]string{
	"play":      "play",
	"resume":    "play",
	"pause":     "pause",
	"playpause": "playpause",
	"toggle":    "playpause",
	"next":      "next track",
	"skip":      "next track",
	"previous":  "previous track",
	"prev":      "previous track",
	"back":      "previous track",
}

// isPlayCommand reports whether an AppleScript transport command starts
// playback, which is the only case where launching the app makes sense.
func isPlayCommand(command string) bool {
	return command == "play" || command == "playpause"
}

// normalizeMediaAction resolves an action name to its AppleScript command.
func normalizeMediaAction(action string) (string, error) {
	cmd, ok := mediaActions[strings.ToLower(strings.TrimSpace(action))]
	if !ok {
		return "", fmt.Errorf("unknown media action %q (use play, pause, next or previous)", action)
	}
	return cmd, nil
}

// trackSeparator joins the fields the now-playing AppleScript returns. It is
// the ASCII unit separator, which no track title contains.
const trackSeparator = "\x1f"

// parseTrack reads the separator-joined output of the now-playing script.
func parseTrack(app, raw string) (NowPlaying, error) {
	fields := strings.Split(strings.TrimSpace(raw), trackSeparator)
	if len(fields) < 2 {
		return NowPlaying{}, fmt.Errorf("unexpected now-playing output %q", strings.TrimSpace(raw))
	}
	n := NowPlaying{App: app, State: strings.TrimSpace(fields[0]), Title: strings.TrimSpace(fields[1])}
	if len(fields) > 2 {
		n.Artist = strings.TrimSpace(fields[2])
	}
	if len(fields) > 3 {
		n.Album = strings.TrimSpace(fields[3])
	}
	if n.Title == "" {
		return n, fmt.Errorf("%s reports no current track", app)
	}
	return n, nil
}

// spotlightScopes lists the paths to try for mdfind's -onlyin, best first.
// On APFS the home directory is a firmlink: everyone types /Users/me, but the
// index stores /System/Volumes/Data/Users/me and -onlyin silently returns
// nothing for the short form — the caller uses the first path that exists.
func spotlightScopes(dir string) []string {
	if dir == "" {
		return nil
	}
	const dataVolume = "/System/Volumes/Data"
	if strings.HasPrefix(dir, dataVolume) {
		return []string{dir}
	}
	return []string{filepath.Join(dataVolume, dir), dir}
}

// mdfindArgs builds a filename search, scoped to one folder when scope is set.
func mdfindArgs(query, scope string) []string {
	args := []string{}
	if scope != "" {
		args = append(args, "-onlyin", scope)
	}
	return append(args, "-name", query)
}

// findArgs builds the `find` fallback used when Spotlight has nothing indexed
// for the folder being searched. It stays shallow and skips dot-directories so
// it returns in a reasonable time on a large home folder.
func findArgs(query, dir string) []string {
	return []string{
		dir, "-maxdepth", "6",
		"-not", "-path", "*/.*",
		"-iname", "*" + query + "*",
	}
}

// keepAwakeArgs builds the caffeinate invocation: block idle, display and disk
// sleep for d. A zero or negative duration means "until cancelled".
func keepAwakeArgs(d time.Duration) []string {
	args := []string{"-dimsu"}
	if d > 0 {
		args = append(args, "-t", strconv.Itoa(int(d.Round(time.Second).Seconds())))
	}
	return args
}

// sayText validates and bounds text destined for the `say` command.
func sayText(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("nothing to say")
	}
	runes := []rune(text)
	if len(runes) > maxSpokenChars {
		text = string(runes[:maxSpokenChars]) + "…"
	}
	return text, nil
}

// limitResults trims a result list to limit entries (limit <= 0 means a
// sensible default rather than unbounded output).
func limitResults(paths []string, limit int) []string {
	if limit <= 0 {
		limit = 20
	}
	if len(paths) > limit {
		return paths[:limit]
	}
	return paths
}

// nonEmptyLines splits tool output into trimmed, non-empty lines.
func nonEmptyLines(out string) []string {
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

// expandPath resolves ~ and makes the path absolute.
func expandPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("expand %q: %w", path, err)
		}
		path = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(path, "~"), "/"))
	}
	return filepath.Abs(path)
}

// expandDir is expandPath with the home folder as the default, which is the
// right starting point for a file search from a phone.
func expandDir(dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return os.UserHomeDir()
	}
	return expandPath(dir)
}
