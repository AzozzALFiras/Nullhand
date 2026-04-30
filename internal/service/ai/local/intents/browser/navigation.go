package browser

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
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
		// LOW priority — many other patterns start with these verbs.
		intents.Intent{
			Priority: intents.PriorityLow,
			Re:       regexp.MustCompile(`(?i)^(?:search|google|ابحث|بحث)\s+(?:for\s+|about\s+|عن\s+)?(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				query := strings.TrimSpace(m[1])
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "browser_google_search",
					"params_json": intents.MustJSON(map[string]string{"browser": "Firefox", "query": query}),
				})}
			},
		},

		// ── Open URL directly ────────────────────────────────────────
		// "go to github.com" / "روح لـ github.com"
		intents.Intent{
			Priority: intents.PriorityNormal,
			Re:       regexp.MustCompile(`(?i)^(?:go\s+to|visit|navigate\s+to|روح\s+(?:ل|لـ|الى|إلى)|اذهب\s+(?:ل|لـ|الى|إلى)|انتقل\s+(?:ل|لـ|الى|إلى)|زر\s+موقع)\s+(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				url := strings.TrimSpace(m[1])
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "browser_open_url",
					"params_json": intents.MustJSON(map[string]string{"browser": "Firefox", "url": url}),
				})}
			},
		},

		// ── Type URL in address bar ──────────────────────────────────
		// "type X in address bar" / "اكتب X في شريط العنوان"
		// "type X in <browser> address bar"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:type|write|اكتب|أكتب)\s+(.+?)\s+(?:in|inside|في)\s+(?:(?:the\s+)?(?:address\s+bar|url\s+bar|location\s+bar)|شريط\s+(?:العنوان|الويب|الموقع))(?:\s+of\s+(.+?))?\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				url := strings.TrimSpace(m[1])
				browser := "Firefox"
				if m[2] != "" {
					browser = intents.ResolveAppName(strings.TrimSpace(m[2]))
				}
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "browser_open_url",
					"params_json": intents.MustJSON(map[string]string{"browser": browser, "url": url}),
				})}
			},
		},

		// ── New tab ──────────────────────────────────────────────────
		// "new tab" / "علامة تبويب جديدة" / "تبويب جديد" / "open new tab"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open\s+)?new\s+tab|^(?:افتح\s+)?(?:علامة\s+تبويب|تبويب)\s+جديد(?:ة)?\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "browser_new_tab",
					"params_json": intents.MustJSON(map[string]string{"browser": "Firefox"}),
				})}
			},
		},

		// ── Close tab ────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:close\s+(?:current\s+)?tab|close\s+this\s+tab)\.?$|^(?:أغلق|اغلق|إغلق)\s+(?:علامة\s+التبويب|التبويب|التاب)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "browser_close_tab",
					"params_json": intents.MustJSON(map[string]string{"browser": "Firefox"}),
				})}
			},
		},

		// ── Back / forward / refresh ─────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:back|go\s+back|previous\s+page)\.?$|^(?:ارجع|إرجع|رجوع|الصفحة\s+السابقة)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "browser_back",
					"params_json": intents.MustJSON(map[string]string{"browser": "Firefox"}),
				})}
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:forward|go\s+forward|next\s+page)\.?$|^(?:الأمام|للأمام|الصفحة\s+التالية|تقدم)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "browser_forward",
					"params_json": intents.MustJSON(map[string]string{"browser": "Firefox"}),
				})}
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:refresh|reload|refresh\s+page|reload\s+page)\.?$|^(?:تحديث|حدّث|أعد\s+تحميل|إعادة\s+تحميل)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "browser_reload",
					"params_json": intents.MustJSON(map[string]string{"browser": "Firefox"}),
				})}
			},
		},
	)
}
