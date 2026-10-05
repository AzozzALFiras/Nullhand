//go:build darwin

package local

import (
	"strings"
	"testing"
)

// The macOS intents are registered only on darwin, so this file is too.
func TestParserMacIntents(t *testing.T) {
	cases := []struct {
		input    string
		wantTool string
		wantArg  string // argument key to check, empty to skip
		wantVal  string
	}{
		// ── Volume ────────────────────────────────────────────────
		{"volume 40", "control_audio", "percent", "40"},
		{"vol 5%", "control_audio", "percent", "5"},
		{"الصوت 70", "control_audio", "percent", "70"},
		{"اضبط الصوت على 20", "control_audio", "percent", "20"},
		{"volume up", "control_audio", "action", "up"},
		{"ارفع الصوت", "control_audio", "action", "up"},
		{"اخفض الصوت", "control_audio", "action", "down"},
		{"mute", "control_audio", "action", "mute"},
		{"اكتم الصوت", "control_audio", "action", "mute"},
		{"unmute", "control_audio", "action", "unmute"},

		// ── Media ─────────────────────────────────────────────────
		{"what's playing", "control_media", "action", "info"},
		{"الاغنية الحالية", "control_media", "action", "info"},
		{"next track", "control_media", "action", "next"},
		{"الاغنية التالية", "control_media", "action", "next"},
		{"previous track", "control_media", "action", "previous"},
		{"pause the music", "control_media", "action", "pause"},
		{"أوقف الموسيقى", "control_media", "action", "pause"},
		{"شغل الموسيقى", "control_media", "action", "play"},

		// ── Shortcuts ─────────────────────────────────────────────
		{"run shortcut Airplane Mode Lock", "run_shortcut", "name", "Airplane Mode Lock"},
		{"شغل اختصار تشغيل الاضاءة", "run_shortcut", "name", "تشغيل الاضاءة"},
		{"نفذ الاختصار Good Morning", "run_shortcut", "name", "Good Morning"},
		{"shortcuts", "list_shortcuts", "", ""},
		{"الاختصارات", "list_shortcuts", "", ""},

		// ── Speech & notifications ────────────────────────────────
		{"say dinner is ready", "say_text", "text", "dinner is ready"},
		{"قل الغداء جاهز", "say_text", "text", "الغداء جاهز"},
		{"notify build finished", "show_notification", "text", "build finished"},
		{"نبه انتهى البناء", "show_notification", "text", "انتهى البناء"},

		// ── Spotlight & Quick Look ────────────────────────────────
		{"find invoice.pdf", "find_files", "query", "invoice.pdf"},
		{"find files report", "find_files", "query", "report"},
		{"ابحث عن ملف الفاتورة", "find_files", "query", "الفاتورة"},
		{"preview ~/Desktop/plan.pdf", "preview_file", "path", "~/Desktop/plan.pdf"},

		// ── Power & appearance ────────────────────────────────────
		{"battery", "mac_power", "action", "battery"},
		{"كم البطارية", "mac_power", "action", "battery"},
		{"lock", "mac_power", "action", "lock"},
		{"lock the screen", "mac_power", "action", "lock"},
		{"اقفل الشاشة", "mac_power", "action", "lock"},
		{"sleep", "mac_power", "action", "sleep"},
		{"dark mode", "mac_power", "action", "dark_toggle"},
		{"الوضع الليلي", "mac_power", "action", "dark_toggle"},
		{"keep awake", "mac_power", "action", "awake"},
		{"keep awake 90", "mac_power", "minutes", "90"},
		{"keep awake 2h", "mac_power", "minutes", "120"},
		{"لا تنم 45", "mac_power", "minutes", "45"},
		{"awake off", "mac_power", "action", "awake_off"},
	}

	for _, c := range cases {
		calls := Parse(c.input)
		if len(calls) == 0 {
			t.Errorf("%q: no tool call produced", c.input)
			continue
		}
		if calls[0].ToolName != c.wantTool {
			t.Errorf("%q: got tool %q, want %q", c.input, calls[0].ToolName, c.wantTool)
			continue
		}
		if c.wantArg == "" {
			continue
		}
		if got := calls[0].Arguments[c.wantArg]; got != c.wantVal {
			t.Errorf("%q: %s = %q, want %q", c.input, c.wantArg, got, c.wantVal)
		}
	}
}

// The macOS intents are registered first, so they must not swallow phrases
// that belong to the existing cross-platform intents.
func TestParserMacIntentsDoNotHijackOtherPhrases(t *testing.T) {
	cases := []struct {
		input    string
		wantTool string
	}{
		{"open Safari", "open_app"},
		{"take a screenshot", "take_screenshot"},
		{"type hello world", "type_text"},
		{"run ls -la", "run_shell"},
		{"اكتب مرحبا", "type_text"},
		{"افتح Safari", "open_app"},
	}
	for _, c := range cases {
		calls := Parse(c.input)
		if len(calls) == 0 {
			t.Errorf("%q: no tool call produced", c.input)
			continue
		}
		if calls[0].ToolName != c.wantTool {
			t.Errorf("%q: got %q, want %q — a macOS intent is too greedy", c.input, calls[0].ToolName, c.wantTool)
		}
	}
}

// A bare "next" is a Next button far more often than a track skip, so it must
// not be captured by the media intent.
func TestParserMacMediaNeedsExplicitTrackWords(t *testing.T) {
	for _, bare := range []string{"next", "previous", "التالي", "السابق"} {
		for _, call := range Parse(bare) {
			if call.ToolName == "control_media" {
				t.Errorf("%q should not be read as a track skip", bare)
			}
		}
	}
}

func TestPreviewDescribesMacTools(t *testing.T) {
	cases := map[string]string{
		"battery":        "power",
		"volume 30":      "volume",
		"say hello":      "Speak out loud",
		"find notes.txt": "Search for files",
	}
	for input, want := range cases {
		got := Preview(input, nil)
		if !strings.Contains(got, want) {
			t.Errorf("preview of %q should mention %q, got:\n%s", input, want, got)
		}
	}
}
