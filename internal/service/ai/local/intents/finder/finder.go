package finder

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSmart(
		// ── Go to folder ─────────────────────────────────────────────
		// "open finder and go to /Users" / "افتح فايندر وروح لـ /Documents"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open\s+)?(?:finder|فايندر|الملفات)\s+(?:go\s+to|open|روح\s+(?:ل|لـ)|افتح)\s+(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				path := strings.TrimSpace(m[1])
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "finder_go_to",
					"params_json": intents.MustJSON(map[string]string{"path": path}),
				})}
			},
		},
	)
}
