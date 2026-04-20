package browser

import (
	"regexp"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSimple(
		// ── New tab ──────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:new\s+tab|تاب\s+جديد|فتح\s+تاب)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("press_key", map[string]string{"key": "cmd+t"})}
			},
		},

		// ── Close tab ────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:close\s+tab|اغلق\s+(?:التاب|تاب)|سكر\s+(?:التاب|تاب))\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("press_key", map[string]string{"key": "cmd+w"})}
			},
		},

		// ── Next tab ─────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:next\s+tab|التاب\s+(?:التالي|الجاي)|تاب\s+(?:تالي|جاي))\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("press_key", map[string]string{"key": "ctrl+tab"})}
			},
		},

		// ── Previous tab ─────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:prev(?:ious)?\s+tab|التاب\s+السابق|تاب\s+سابق)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("press_key", map[string]string{"key": "ctrl+shift+tab"})}
			},
		},

		// ── Reload ───────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:reload|refresh|حدث\s+الصفحة|تحديث|اعد\s+تحميل)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("press_key", map[string]string{"key": "cmd+r"})}
			},
		},

		// ── Back ─────────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:go\s+back|back|رجوع|ارجع)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("press_key", map[string]string{"key": "cmd+["})}
			},
		},

		// ── Forward ──────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:go\s+forward|forward|تقدم|للامام)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("press_key", map[string]string{"key": "cmd+]"})}
			},
		},

		// ── Find in page ─────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:find|find\s+in\s+page|ابحث\s+في\s+الصفحة|بحث\s+في\s+الصفحة)\s+(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{
					intents.ToolCall("press_key", map[string]string{"key": "cmd+f"}),
					intents.ToolCall("wait", map[string]string{"ms": "300"}),
					intents.ToolCall("type_text", map[string]string{"text": m[1]}),
				}
			},
		},
	)
}
