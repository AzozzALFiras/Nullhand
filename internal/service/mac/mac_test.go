package mac

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestQuoteAppleScript(t *testing.T) {
	cases := []struct{ in, want string }{
		{`plain`, `"plain"`},
		{`say "hi"`, `"say \"hi\""`},
		{`C:\path\x`, `"C:\\path\\x"`},
		{"two\nlines", `"two\nlines"`},
		{"tab\there", `"tab\there"`},
		{"مرحبا", `"مرحبا"`},
	}
	for _, c := range cases {
		if got := QuoteAppleScript(c.in); got != c.want {
			t.Errorf("QuoteAppleScript(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

// A quote in user text must not be able to close the literal and append
// AppleScript statements of its own.
func TestQuoteAppleScriptBlocksInjection(t *testing.T) {
	malicious := `x" & (do shell script "touch /tmp/pwned") & "`
	quoted := QuoteAppleScript(malicious)

	// The literal may only be closed by its own final quote: walking the body
	// and skipping escape pairs must never meet a bare quote.
	body := quoted[1 : len(quoted)-1]
	for i := 0; i < len(body); i++ {
		switch body[i] {
		case '\\':
			i++ // the escaped character cannot end the literal
		case '"':
			t.Fatalf("unescaped quote at index %d lets the rest run as AppleScript: %s", i, quoted)
		}
	}
}

func TestParseVolumeSettings(t *testing.T) {
	got, err := parseVolumeSettings("output volume:44, input volume:30, alert volume:100, output muted:false")
	if err != nil {
		t.Fatal(err)
	}
	if got.Output != 44 || got.Muted {
		t.Errorf("got %+v, want {44 false}", got)
	}

	got, err = parseVolumeSettings("output volume:0, input volume:30, alert volume:100, output muted:true")
	if err != nil {
		t.Fatal(err)
	}
	if got.Output != 0 || !got.Muted {
		t.Errorf("got %+v, want {0 true}", got)
	}
}

func TestParseVolumeSettingsErrors(t *testing.T) {
	// Some external DACs and HDMI displays report no software volume.
	if _, err := parseVolumeSettings("output volume:missing value, output muted:false"); err == nil {
		t.Error("missing value must be reported as an error, not volume 0")
	}
	if _, err := parseVolumeSettings("unexpected"); err == nil {
		t.Error("unparseable output must return an error")
	}
}

func TestClampPercent(t *testing.T) {
	for in, want := range map[int]int{-10: 0, 0: 0, 50: 50, 100: 100, 180: 100} {
		if got := clampPercent(in); got != want {
			t.Errorf("clampPercent(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestParseBattery(t *testing.T) {
	cases := []struct {
		name     string
		out      string
		present  bool
		percent  int
		state    string
		source   string
		timeLeft string
	}{
		{
			name:    "plugged in, not charging",
			out:     "Now drawing from 'AC Power'\n -InternalBattery-0 (id=21823587)\t95%; AC attached; not charging present: true",
			present: true, percent: 95, state: "AC attached", source: "AC Power",
		},
		{
			name:    "discharging with estimate",
			out:     "Now drawing from 'Battery Power'\n -InternalBattery-0 (id=123)\t72%; discharging; 3:21 remaining present: true",
			present: true, percent: 72, state: "discharging", source: "Battery Power", timeLeft: "3:21",
		},
		{
			name:    "charging",
			out:     "Now drawing from 'AC Power'\n -InternalBattery-0 (id=123)\t45%; charging; 1:05 remaining present: true",
			present: true, percent: 45, state: "charging", source: "AC Power", timeLeft: "1:05",
		},
		{
			name:    "no estimate yet",
			out:     "Now drawing from 'Battery Power'\n -InternalBattery-0 (id=123)\t88%; discharging; (no estimate) present: true",
			present: true, percent: 88, state: "discharging", source: "Battery Power",
		},
		{
			name:   "desktop Mac without a battery",
			out:    "Now drawing from 'AC Power'",
			source: "AC Power",
		},
	}
	for _, c := range cases {
		got, err := parseBattery(c.out)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got.Present != c.present || got.Percent != c.percent ||
			got.State != c.state || got.Source != c.source || got.TimeLeft != c.timeLeft {
			t.Errorf("%s:\n got %+v\nwant {Present:%v Percent:%d State:%q Source:%q TimeLeft:%q}",
				c.name, got, c.present, c.percent, c.state, c.source, c.timeLeft)
		}
	}
}

func TestParseBatteryRejectsGarbage(t *testing.T) {
	if _, err := parseBattery(""); err == nil {
		t.Error("empty pmset output must return an error")
	}
	if _, err := parseBattery("something else entirely"); err == nil {
		t.Error("unrecognised pmset output must return an error")
	}
}

func TestBatterySummary(t *testing.T) {
	cases := []struct {
		b        Battery
		contains string
	}{
		{Battery{Present: true, Percent: 95, State: "AC attached"}, "95%"},
		{Battery{Present: true, Percent: 72, State: "discharging", TimeLeft: "3:21"}, "3:21 remaining"},
		{Battery{Present: true, Percent: 8, State: "discharging"}, "🪫"},
		{Battery{Present: false, Source: "AC Power"}, "No battery"},
	}
	for _, c := range cases {
		if got := c.b.Summary(); !strings.Contains(got, c.contains) {
			t.Errorf("Summary() = %q, want it to contain %q", got, c.contains)
		}
	}
}

func TestParseShortcutNames(t *testing.T) {
	got := parseShortcutNames("\nAirplane Mode Lock\nPIP image\n\n  Search YouTube  \n\n")
	want := []string{"Airplane Mode Lock", "PIP image", "Search YouTube"}
	if len(got) != len(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("name %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNormalizeMediaAction(t *testing.T) {
	for in, want := range map[string]string{
		"play": "play", "PAUSE": "pause", "next": "next track",
		"skip": "next track", "prev": "previous track", "toggle": "playpause",
	} {
		got, err := normalizeMediaAction(in)
		if err != nil || got != want {
			t.Errorf("normalizeMediaAction(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := normalizeMediaAction("eject"); err == nil {
		t.Error("unknown action must return an error listing the valid ones")
	}
}

func TestParseTrack(t *testing.T) {
	raw := "playing" + trackSeparator + "Bohemian Rhapsody" + trackSeparator + "Queen" + trackSeparator + "A Night at the Opera"
	got, err := parseTrack("Music", raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "playing" || got.Title != "Bohemian Rhapsody" || got.Artist != "Queen" || got.Album != "A Night at the Opera" {
		t.Errorf("got %+v", got)
	}
	if s := got.Summary(); !strings.Contains(s, "Bohemian Rhapsody") || !strings.Contains(s, "Queen") {
		t.Errorf("Summary() = %q", s)
	}
}

func TestParseTrackStopped(t *testing.T) {
	if _, err := parseTrack("Spotify", "stopped"+trackSeparator+""); err == nil {
		t.Error("a stopped player has no track and must report that")
	}
	if _, err := parseTrack("Music", "garbage"); err == nil {
		t.Error("output without a separator must return an error")
	}
}

func TestSpotlightScopes(t *testing.T) {
	// The data-volume path comes first: -onlyin silently finds nothing for the
	// short /Users form on APFS.
	got := spotlightScopes("/Users/me/Documents")
	if len(got) != 2 || got[0] != "/System/Volumes/Data/Users/me/Documents" || got[1] != "/Users/me/Documents" {
		t.Errorf("got %q", got)
	}
	if got := spotlightScopes("/System/Volumes/Data/Users/me"); len(got) != 1 {
		t.Errorf("an already-resolved path needs no alternative, got %q", got)
	}
	if got := spotlightScopes(""); got != nil {
		t.Errorf("no folder means no scope, got %q", got)
	}
}

func TestMdfindArgs(t *testing.T) {
	got := strings.Join(mdfindArgs("report.pdf", "/System/Volumes/Data/Users/me"), " ")
	if got != "-onlyin /System/Volumes/Data/Users/me -name report.pdf" {
		t.Errorf("got %q", got)
	}
	if got := strings.Join(mdfindArgs("report.pdf", ""), " "); got != "-name report.pdf" {
		t.Errorf("unscoped search should search the whole index, got %q", got)
	}
}

func TestFindArgs(t *testing.T) {
	got := findArgs("report", "/Users/me")
	joined := strings.Join(got, " ")
	if got[0] != "/Users/me" {
		t.Errorf("find must start at the folder, got %q", got[0])
	}
	for _, want := range []string{"-maxdepth 6", "-iname *report*", "-not -path */.*"} {
		if !strings.Contains(joined, want) {
			t.Errorf("args %q missing %q", joined, want)
		}
	}
}

func TestKeepAwakeArgs(t *testing.T) {
	if got := strings.Join(keepAwakeArgs(0), " "); got != "-dimsu" {
		t.Errorf("no duration means until cancelled, got %q", got)
	}
	if got := strings.Join(keepAwakeArgs(90*time.Minute), " "); got != "-dimsu -t 5400" {
		t.Errorf("got %q", got)
	}
}

func TestSayText(t *testing.T) {
	if _, err := sayText("   "); err == nil {
		t.Error("empty text must return an error")
	}
	long := strings.Repeat("ا", maxSpokenChars+50)
	got, err := sayText(long)
	if err != nil {
		t.Fatal(err)
	}
	if n := len([]rune(got)); n != maxSpokenChars+1 { // +1 for the ellipsis
		t.Errorf("text should be truncated to %d runes plus an ellipsis, got %d", maxSpokenChars, n)
	}
}

func TestNonEmptyLines(t *testing.T) {
	got := nonEmptyLines("/a/b\n\n  /c/d  \n")
	if len(got) != 2 || got[0] != "/a/b" || got[1] != "/c/d" {
		t.Errorf("got %q", got)
	}
}

func TestLimitResults(t *testing.T) {
	ten := make([]string, 10)
	if got := limitResults(ten, 3); len(got) != 3 {
		t.Errorf("got %d results, want 3", len(got))
	}
	if got := limitResults(ten, 0); len(got) != 10 {
		t.Errorf("a non-positive limit falls back to the default, got %d", len(got))
	}
}

func TestExpandPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := expandPath("~/Documents/x.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, "Documents/x.pdf"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if _, err := expandPath("  "); err == nil {
		t.Error("empty path must return an error")
	}

	// A search with no folder starts at home.
	dir, err := expandDir("")
	if err != nil || dir != home {
		t.Errorf("expandDir(\"\") = %q, %v; want %q", dir, err, home)
	}
}

func TestVolumeStateSummary(t *testing.T) {
	if got := (VolumeState{Output: 30}).Summary(); !strings.Contains(got, "30%") {
		t.Errorf("got %q", got)
	}
	if got := (VolumeState{Output: 30, Muted: true}).Summary(); !strings.Contains(got, "Muted") {
		t.Errorf("muted state must be visible, got %q", got)
	}
}

func TestIsPlayCommand(t *testing.T) {
	// Only a play request may launch the app; skipping or pausing nothing
	// makes no sense.
	for _, cmd := range []string{"play", "playpause"} {
		if !isPlayCommand(cmd) {
			t.Errorf("%q should allow launching the media app", cmd)
		}
	}
	for _, cmd := range []string{"pause", "next track", "previous track"} {
		if isPlayCommand(cmd) {
			t.Errorf("%q must not launch the media app", cmd)
		}
	}
}
