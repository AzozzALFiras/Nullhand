package slack

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSmart(
		// ── Send message to channel ──────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open\s+)?slack\s+(?:send|message|ارسل)\s+(?:to\s+|في\s+|ل)?#?(.+?)\s+(?:say|message|قل|ارسل)\s+(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				channel := strings.TrimSpace(m[1])
				message := intents.StripQuotes(strings.TrimSpace(m[2]))
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "slack_send_message",
					"params_json": intents.MustJSON(map[string]string{"channel": channel, "message": message}),
				})}
			},
		},

		// ── Open channel ─────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open\s+)?slack\s+(?:channel|go\s+to|قناة)\s+#?(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				channel := strings.TrimSpace(m[1])
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "slack_channel_focus",
					"params_json": intents.MustJSON(map[string]string{"channel": channel}),
				})}
			},
		},
	)
}
