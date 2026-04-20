package vscode

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSmart(
		// ── Type in Claude chat ───────────────────────────────────────
		// "in vs code tell claude ..." / "open vscode and write in claude ..."
		// "tell claude in vscode ..." / "اكتب في كلود ..."
		intents.Intent{
			Re: regexp.MustCompile("(?i)^(?:(?:in\\s+)?(?:vs\\s*code|vscode|visual\\s+studio\\s+code)\\s+.*?(?:write|type|send|told?|tell)\\s+.*?(?:box|claude|chat|message)\\s+.*?[\"\\x{201c}](.+?)[\"\\x{201d}]|open\\s+(?:vs\\s*code|vscode|visual\\s+studio\\s+code)\\s+and\\s+(?:write|type|send|tell)\\s+(?:in\\s+|to\\s+)?(?:claude|the\\s+box)\\s+[\"\\x{201c}]?(.+?)[\"\\x{201d}]?|tell\\s+claude\\s+(?:in\\s+)?(?:vs\\s*code|vscode)\\s+[\"\\x{201c}]?(.+?)[\"\\x{201d}]?)(?:\\s+and\\s+send)?\\.?$"),
			Build: func(m []string) []aimodel.ToolCall {
				message := ""
				for _, g := range m[1:] {
					if strings.TrimSpace(g) != "" {
						message = intents.StripQuotes(strings.TrimSpace(g))
						break
					}
				}
				if message == "" {
					message = "hello"
				}
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "vscode_type_in_claude",
					"params_json": intents.MustJSON(map[string]string{"message": message}),
				})}
			},
		},
	)
}
