package local

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
)

// intent is a pattern that matches a user utterance and produces tool calls.
type intent struct {
	re    *regexp.Regexp
	build func(matches []string) []aimodel.ToolCall
}

// intents is the ordered list of recognised intent patterns. Earlier entries
// win over later ones, so put more specific patterns first.
//
// Supported languages: English and Arabic.
var intents = []intent{
	// ── Screenshot ────────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:take\s+(?:a\s+)?screenshot|screenshot|snap(?:shot)?|لقطة(?:\s*شاشة)?|خذ\s+لقطة(?:\s+شاشة)?|التقط\s+(?:لقطة|شاشة|صورة))\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("take_screenshot", nil)}
		},
	},

	// ── Clipboard: paste / get ────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:paste|الصق|اعرض\s+الحافظة|clipboard)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("get_clipboard", nil)}
		},
	},

	// ── Clipboard: copy <text> ────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:copy|انسخ)\s+(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("set_clipboard", map[string]string{"text": strings.TrimSpace(m[1])})}
		},
	},

	// ── Send / submit (press return) ──────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:send|submit|ارسل|أرسل|اضغط\s+(?:ارسال|enter|return|إدخال))\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("press_key", map[string]string{"key": "return"})}
		},
	},

	// ── Press key: "press cmd+t" / "اضغط cmd+t" ───────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:press|hit|اضغط)\s+(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("press_key", map[string]string{"key": strings.TrimSpace(m[1])})}
		},
	},

	// ── Open app ──────────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:open|launch|start|افتح|شغّل\s+تطبيق)\s+(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("open_app", map[string]string{"app_name": strings.TrimSpace(m[1])})}
		},
	},

	// ── Type / write ──────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:type|write|اكتب|أكتب)\s+(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			text := strings.TrimSpace(m[1])
			// Strip matched surrounding quotes.
			text = stripQuotes(text)
			return []aimodel.ToolCall{toolCall("type_text", map[string]string{"text": text})}
		},
	},

	// ── Click UI element by label: "click Send" ───────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:click|press\s+button|انقر|اضغط\s+زر)\s+(?:on\s+)?(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			label := stripQuotes(strings.TrimSpace(m[1]))
			return []aimodel.ToolCall{toolCall("click_ui_element", map[string]string{
				"app_name": "", // empty means "frontmost" — handled by service fallback
				"label":    label,
			})}
		},
	},

	// ── Read file ─────────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:read|show|cat|اقرأ|اعرض\s+محتوى)\s+(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("read_file", map[string]string{"path": strings.TrimSpace(m[1])})}
		},
	},

	// ── List directory ────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:list|ls|dir|اعرض|القائمة|محتوى\s+مجلد)\s+(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("list_directory", map[string]string{"path": strings.TrimSpace(m[1])})}
		},
	},

	// ── Run shell ─────────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:run|shell|exec|execute|شغل|نفذ)\s+(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("run_shell", map[string]string{"command": strings.TrimSpace(m[1])})}
		},
	},
}

// stripQuotes removes a matching pair of surrounding quotes from s.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') ||
			(first == '\'' && last == '\'') ||
			(first == '`' && last == '`') {
			return s[1 : len(s)-1]
		}
		// Unicode smart quotes “ ” and « »
		if strings.HasPrefix(s, "“") && strings.HasSuffix(s, "”") {
			return strings.TrimSuffix(strings.TrimPrefix(s, "“"), "”")
		}
	}
	return s
}
