package messages

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSmart(
		// ── Send iMessage ────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open\s+)?(?:messages|imessage|الرسائل)\s+(?:send|ارسل)\s+(?:to\s+|ل)?(.+?)\s+(?:say|message|قل|ارسل)\s+(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				contact := strings.TrimSpace(m[1])
				message := intents.StripQuotes(strings.TrimSpace(m[2]))
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "messages_send_imessage",
					"params_json": intents.MustJSON(map[string]string{"contact": contact, "message": message}),
				})}
			},
		},
	)
}
