package vscode

import (
	"regexp"
	"strings"

	aimodel "github.com/iamakillah/Nullhand_Linux/internal/model/ai"
	"github.com/iamakillah/Nullhand_Linux/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSmart(
		// ── Open VS Code terminal ────────────────────────────────────
		// "open vs code terminal" / "افتح تيرمنل VS Code"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open|افتح)\s+(?:vs\s*code|vscode|visual\s+studio\s+code)\s+(?:terminal|تيرمنل|الطرفية)$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{"name": "vscode_open_terminal"})}
			},
		},

		// ── Run command in VS Code terminal ──────────────────────────
		// "open vs code and run npm install" / "في vscode نفذ git status"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open\s+(?:vs\s*code|vscode|visual\s+studio\s+code)\s+and\s+(?:run|execute|type)\s+|(?:in|في)\s+(?:vs\s*code|vscode)\s+(?:run|execute|نفذ|شغل)\s+)(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				cmd := strings.TrimSpace(m[1])
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "vscode_terminal_run",
					"params_json": intents.MustJSON(map[string]string{"command": cmd}),
				})}
			},
		},

		// ── New Claude chat ──────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:new|افتح)\s+(?:claude\s+chat|محادثة\s+كلود)\s+(?:in\s+|في\s+)?(?:vs\s*code|vscode)?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{"name": "vscode_new_claude_chat"})}
			},
		},
	)
}
