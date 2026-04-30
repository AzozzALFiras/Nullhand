package whatsapp

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSmart(
		// ── Send message to contact ──────────────────────────────────
		// "open whatsapp and send Azozz hello"
		// "افتح واتساب وارسل/وأرسل/راسل لعزوز رسالة مرحبا"
		// "افتح الواتساب وارسل رسالة لعزوز وقل له مرحبا"
		intents.Intent{
			Priority: intents.PriorityHigh,
			Re:       regexp.MustCompile(`(?i)^(?:open|افتح)\s+(?:whatsapp|واتساب|واتس|الواتساب|الواتس)\s+(?:and\s+(?:send|message|write|tell)|و\s*(?:ارسل|أرسل|اكتب|أكتب|راسل|ابعث|إبعث))\s+(?:to\s+|ل|لـ\s*|إلى\s*|الى\s*)?(\S+(?:\s+\S+)?)\s+(?:a\s+message\s+|message\s+|رسالة\s+|وقل(?:\s+له)?\s+|قل(?:\s+له)?\s+|وقول(?:\s+له)?\s+|نص\s+)(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				contact := strings.TrimSpace(m[1])
				message := intents.StripQuotes(strings.TrimSpace(m[2]))
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "whatsapp_send_message",
					"params_json": intents.MustJSON(map[string]string{"contact": contact, "message": message}),
				})}
			},
		},

		// ── Send message (simpler pattern) ───────────────────────────
		// "whatsapp Azozz hello" / "واتساب عزوز مرحبا"
		intents.Intent{
			Priority: intents.PriorityHigh,
			Re:       regexp.MustCompile(`(?i)^(?:whatsapp|واتساب|واتس)\s+(\S+(?:\s+\S+)?)\s+(?:say|send|قل|ارسل|أرسل)\s+(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				contact := strings.TrimSpace(m[1])
				message := intents.StripQuotes(strings.TrimSpace(m[2]))
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "whatsapp_send_message",
					"params_json": intents.MustJSON(map[string]string{"contact": contact, "message": message}),
				})}
			},
		},

		// ── Open chat (no message) ───────────────────────────────────
		// "open whatsapp chat with Azozz" / "افتح واتساب محادثة عزوز"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open|افتح)\s+(?:whatsapp|واتساب|واتس|الواتساب|الواتس)\s+(?:chat\s+(?:with\s+)?|محادثة\s+|شات\s+)(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				contact := strings.TrimSpace(m[1])
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "whatsapp_new_message",
					"params_json": intents.MustJSON(map[string]string{"contact": contact}),
				})}
			},
		},

		// ── Arabic: colon-separated send ───────────────────────────
		// "ارسل لعزوز في الواتساب: مرحبا" / "واتساب عزوز: مرحبا"
		// "send azozz on whatsapp: hello"
		intents.Intent{
			Priority: intents.PriorityHigh,
			Re: regexp.MustCompile(`(?i)^(?:` +
				// "ارسل/راسل (لـ|إلى) X (في|على) (الواتساب|واتساب): MSG"
				`(?:ارسل|أرسل|راسل|ابعث|إبعث)\s+(?:ل|لـ|إلى|الى)?\s*(\S+(?:\s+\S+)?)\s+(?:في|على|عبر)\s+(?:whatsapp|واتساب|الواتساب|واتس|الواتس)\s*[:\-]\s*(.+)` +
				`|` +
				// "واتساب X: MSG" / "whatsapp X: MSG"
				`(?:whatsapp|واتساب|الواتساب|واتس|الواتس)\s+(\S+(?:\s+\S+)?)\s*[:\-]\s*(.+)` +
				`|` +
				// "send to X on whatsapp: MSG"
				`send\s+(?:a\s+message\s+)?to\s+(\S+(?:\s+\S+)?)\s+on\s+whatsapp\s*[:\-]\s*(.+)` +
				`)$`),
			Build: func(m []string) []aimodel.ToolCall {
				contact := strings.TrimSpace(firstNonEmpty([]string{m[1], m[3], m[5]}))
				message := strings.TrimSpace(intents.StripQuotes(firstNonEmpty([]string{m[2], m[4], m[6]})))
				if contact == "" || message == "" {
					return nil
				}
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "whatsapp_send_message",
					"params_json": intents.MustJSON(map[string]string{"contact": contact, "message": message}),
				})}
			},
		},

		// ── "ارسل ل X مرحبا" without colon (assumes single-word contact) ──
		intents.Intent{
			Priority: intents.PriorityHigh,
			Re:       regexp.MustCompile(`^(?:ارسل|أرسل|راسل|ابعث)\s+(?:ل|لـ|إلى|الى)\s*(\S+)\s+(?:في|على|عبر)\s+(?:whatsapp|واتساب|الواتساب|واتس|الواتس)\s+(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				contact := strings.TrimSpace(m[1])
				message := strings.TrimSpace(intents.StripQuotes(m[2]))
				if contact == "" || message == "" {
					return nil
				}
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "whatsapp_send_message",
					"params_json": intents.MustJSON(map[string]string{"contact": contact, "message": message}),
				})}
			},
		},
	)
}

// firstNonEmpty returns the first trimmed non-empty match in the slice.
func firstNonEmpty(parts []string) string {
	for _, s := range parts {
		s = strings.TrimSpace(s)
		if s != "" {
			return s
		}
	}
	return ""
}
