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
		// "افتح واتساب وارسل لعزوز مرحبا"
		// "افتح الواتساب وارسل رسالة لعزوز وقل له مرحبا"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open|افتح)\s+(?:whatsapp|واتساب|واتس|الواتساب|الواتس)\s+(?:and\s+(?:send|message|write|tell)|و\s*(?:ارسل|اكتب|راسل))\s+(?:to\s+|ل|لـ\s*)?(.+?)\s+(?:a\s+message\s+|message\s+|رسالة\s+|وقل(?:\s+له)?\s+|قل(?:\s+له)?\s+|وقول(?:\s+له)?\s+)(.+?)$`),
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
			Re: regexp.MustCompile(`(?i)^(?:whatsapp|واتساب|واتس)\s+(.+?)\s+(?:say|send|قل|ارسل)\s+(.+?)$`),
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
	)
}
