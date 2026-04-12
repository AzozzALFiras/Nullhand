package browser

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
	"github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSmart(
		// ── Open browser and go to URL ───────────────────────────────
		// "open safari and go to github.com" / "افتح سفاري وروح لـ github.com"
		// "open chrome and navigate to google.com"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open|افتح)\s+(.+?)\s+(?:and\s+(?:go\s+to|navigate\s+to|open|visit)|و\s*(?:روح|اذهب|انتقل|افتح)\s+(?:ل|لـ|الى|إلى)\s*)(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				app := intents.ResolveAppName(m[1])
				url := strings.TrimSpace(m[2])
				if intents.IsBrowser(app) {
					return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
						"name":        "browser_open_url",
						"params_json": intents.MustJSON(map[string]string{"browser": app, "url": url}),
					})}
				}
				return []aimodel.ToolCall{intents.ToolCall("open_app", map[string]string{"app_name": app})}
			},
		},

		// ── Open browser and search ──────────────────────────────────
		// "open safari and search for X" / "افتح سفاري وابحث عن X"
		// "open chrome and google X"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open|افتح)\s+(.+?)\s+(?:and\s+(?:search|google|look|find)\s+(?:for\s+)?|و\s*(?:ابحث|بحث)\s+(?:عن\s+)?)(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				app := intents.ResolveAppName(m[1])
				query := strings.TrimSpace(m[2])
				if intents.IsBrowser(app) {
					return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
						"name":        "browser_google_search",
						"params_json": intents.MustJSON(map[string]string{"browser": app, "query": query}),
					})}
				}
				return []aimodel.ToolCall{intents.ToolCall("open_app", map[string]string{"app_name": app})}
			},
		},

		// ── Search directly (no browser specified) ───────────────────
		// "search for X" / "ابحث عن X" / "google X"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:search|google|ابحث|بحث)\s+(?:for\s+|about\s+|عن\s+)?(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				query := strings.TrimSpace(m[1])
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "browser_google_search",
					"params_json": intents.MustJSON(map[string]string{"browser": "Safari", "query": query}),
				})}
			},
		},

		// ── Open URL directly ────────────────────────────────────────
		// "go to github.com" / "روح لـ github.com"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:go\s+to|visit|navigate\s+to|روح\s+(?:ل|لـ)|اذهب\s+(?:ل|لـ|الى|إلى))\s+(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				url := strings.TrimSpace(m[1])
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "browser_open_url",
					"params_json": intents.MustJSON(map[string]string{"browser": "Safari", "url": url}),
				})}
			},
		},
	)
}
