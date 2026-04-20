package closeapp

import (
	"regexp"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSimple(
		// English: close/kill/quit/exit <app>
		// Arabic: أغلق/اغلق/أقفل/اقفل <app>
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:close|kill|quit|exit|أغلق|اغلق|أقفل|اقفل)\s+(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("close_app", map[string]string{
					"app_name": intents.ResolveAppName(m[1]),
				})}
			},
		},
	)
}
