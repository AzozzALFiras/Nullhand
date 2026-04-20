package discord

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSmart(
		// ── Send DM ──────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open\s+)?discord\s+(?:send|message|dm|ارسل)\s+(?:to\s+|ل)?(.+?)\s+(?:say|send|قل|ارسل)\s+(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				recipient := strings.TrimSpace(m[1])
				message := intents.StripQuotes(strings.TrimSpace(m[2]))
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "discord_send_dm",
					"params_json": intents.MustJSON(map[string]string{"recipient": recipient, "message": message}),
				})}
			},
		},

		// ── Open DM ──────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open\s+)?discord\s+(?:dm|chat|محادثة)\s+(?:with\s+|مع\s+)?(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				recipient := strings.TrimSpace(m[1])
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "discord_dm_focus",
					"params_json": intents.MustJSON(map[string]string{"recipient": recipient}),
				})}
			},
		},
	)
}
