package settings

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSmart(
		// ── Search inside Settings ───────────────────────────────────
		// "search settings for X" / "ابحث في الإعدادات عن X"
		// "find X in settings" / "settings search X"
		intents.Intent{
			Priority: intents.PriorityVeryHigh,
			Re: regexp.MustCompile(`(?i)^(?:` +
				// "search/find ... settings ... X"
				`(?:search|find|look)(?:\s+for)?\s+(?:in\s+)?(?:system\s+)?settings\s+(?:for\s+|about\s+)?(.+?)` +
				`|` +
				// "settings search X" / "settings: X"
				`(?:system\s+)?settings\s+(?:search|find|for)\s+(.+?)` +
				`|` +
				// Arabic: "ابحث في الإعدادات عن X" / "ابحث في الضبط عن X"
				`(?:ابحث|بحث|فتش|جد)\s+(?:في\s+(?:الإعدادات|الاعدادات|الضبط|إعدادات\s+النظام|اعدادات)\s+)(?:عن\s+|على\s+)?(.+?)` +
				`|` +
				// Arabic compact: "إعدادات WiFi" / "اعدادات WiFi" with explicit search verb
				`(?:بحث|ابحث)\s+(?:إعدادات|الإعدادات|الاعدادات|اعدادات)\s+(.+?)` +
				`)$`),
			Build: func(m []string) []aimodel.ToolCall {
				query := firstNonEmpty(m[1:])
				if query == "" {
					return nil
				}
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "settings_search",
					"params_json": intents.MustJSON(map[string]string{"query": query}),
				})}
			},
		},

		// ── Open a specific Settings panel ───────────────────────────
		// "open WiFi settings" / "افتح إعدادات WiFi"
		// "open Bluetooth in settings" / "go to display settings"
		intents.Intent{
			Priority: intents.PriorityVeryHigh,
			Re: regexp.MustCompile(`(?i)^(?:` +
				// "open/go to X settings"
				`(?:open|go\s+to|navigate\s+to|show)\s+(.+?)\s+settings` +
				`|` +
				// "open settings X" / "settings X"
				`(?:open\s+)?(?:system\s+)?settings\s+(?:for\s+|panel\s+)?(.+?)` +
				`|` +
				// Arabic: "افتح إعدادات X" / "افتح ضبط X" / "اعدادات X"
				`(?:افتح|إفتح|اذهب\s+إلى|انتقل\s+إلى)\s+(?:إعدادات|الإعدادات|اعدادات|الاعدادات|ضبط|الضبط)\s+(.+?)` +
				`)$`),
			Build: func(m []string) []aimodel.ToolCall {
				panel := firstNonEmpty(m[1:])
				if panel == "" {
					return nil
				}
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "settings_open_panel",
					"params_json": intents.MustJSON(map[string]string{"panel": panel}),
				})}
			},
		},
	)
}

// firstNonEmpty returns the first trimmed non-empty match.
func firstNonEmpty(matches []string) string {
	for _, m := range matches {
		s := strings.TrimSpace(m)
		if s != "" {
			return s
		}
	}
	return ""
}
