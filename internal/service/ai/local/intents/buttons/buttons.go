package buttons

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSmart(
		// ── Click a labelled button (English first, more specific) ──
		// "click the send button" / "click button send" / "press the X button"
		intents.Intent{
			Priority: intents.PriorityHigh,
			Re: regexp.MustCompile(`(?i)^(?:click|tap|press|hit|select)\s+(?:on\s+)?(?:the\s+)?(?:button\s+(?:labelled\s+|labeled\s+|named\s+|called\s+)?)(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				label := strings.TrimSpace(intents.StripQuotes(m[1]))
				if label == "" {
					return nil
				}
				return clickButtonCalls(label)
			},
		},

		// "click the X button" / "press X button"
		intents.Intent{
			Priority: intents.PriorityHigh,
			Re:       regexp.MustCompile(`(?i)^(?:click|tap|press|hit|select)\s+(?:on\s+)?(?:the\s+)?(.+?)\s+button\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				label := strings.TrimSpace(intents.StripQuotes(m[1]))
				if label == "" {
					return nil
				}
				return clickButtonCalls(label)
			},
		},

		// ── Arabic: "اضغط زر إرسال" / "اضغط على زر X" / "انقر زر X" ──
		intents.Intent{
			Priority: intents.PriorityHigh,
			Re:       regexp.MustCompile(`^(?:اضغط|إضغط|انقر|إنقر|اختر|حدد)\s+(?:على\s+)?(?:زر|الزر|زرّ)\s*[:\-]?\s*(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				label := strings.TrimSpace(intents.StripQuotes(m[1]))
				if label == "" {
					return nil
				}
				return clickButtonCalls(label)
			},
		},

		// ── Arabic: "اضغط زر إرسال" reversed: "زر إرسال اضغط" — uncommon, skip ──
		// ── Arabic: "اضغط على إرسال" (without word "زر") — fallback, lower priority ──
		intents.Intent{
			Priority: intents.PriorityNormal,
			Re:       regexp.MustCompile(`^(?:اضغط|إضغط|انقر|إنقر)\s+(?:على\s+)?(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				label := strings.TrimSpace(intents.StripQuotes(m[1]))
				// Reject coordinates ("123 456") and known modifier-y/key-y words
				if label == "" || isCoordinateLike(label) || isKeyLike(label) {
					return nil
				}
				return clickButtonCalls(label)
			},
		},

		// ── "click X in app Y" / "press X in Y" ──
		// e.g. "click Send in WhatsApp"
		intents.Intent{
			Priority: intents.PriorityHigh,
			Re:       regexp.MustCompile(`(?i)^(?:click|tap|press)\s+(.+?)\s+(?:in|inside)\s+(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				label := strings.TrimSpace(intents.StripQuotes(m[1]))
				app := intents.ResolveAppName(strings.TrimSpace(m[2]))
				if label == "" || app == "" || isCoordinateLike(label) || isKeyLike(label) {
					return nil
				}
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "press_button_in_app",
					"params_json": intents.MustJSON(map[string]string{"app": app, "label": label}),
				})}
			},
		},
	)
}

func clickButtonCalls(label string) []aimodel.ToolCall {
	return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
		"name":        "click_button",
		"params_json": intents.MustJSON(map[string]string{"label": label}),
	})}
}

// isCoordinateLike returns true if s looks like coordinate input.
var coordRe = regexp.MustCompile(`^\d+\s*[,\s]\s*\d+$`)

func isCoordinateLike(s string) bool { return coordRe.MatchString(s) }

// keyHints reserves words that are clearly keys/shortcuts so we don't try to
// click them as button labels.
var keyHints = map[string]bool{
	"return": true, "enter": true, "escape": true, "esc": true, "tab": true,
	"backspace": true, "delete": true, "space": true, "ctrl+c": true, "ctrl+v": true,
}

func isKeyLike(s string) bool {
	low := strings.ToLower(strings.TrimSpace(s))
	if keyHints[low] {
		return true
	}
	if strings.Contains(low, "ctrl+") || strings.Contains(low, "cmd+") ||
		strings.Contains(low, "alt+") || strings.Contains(low, "shift+") {
		return true
	}
	return false
}
