package terminal

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
	"github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSmart(
		// ── Open terminal and run command ─────────────────────────────
		// "open terminal and run ls -la" / "افتح التيرمنل ونفذ ls -la"
		// "افتح التيرمنل واكتب git status" / "open term and execute npm install"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open|launch|افتح|شغل)\s+(?:terminal|term|iterm|التيرمنل|الطرفية|تيرمنل|الترمنل)\s+(?:and\s+(?:run|execute|type|write)|و\s*(?:نفذ|شغل|اكتب|نفّذ|اكتب\s+فيه)|ثم\s+(?:نفذ|شغل|اكتب))\s+(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				cmd := strings.TrimSpace(m[1])
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "terminal_run_command",
					"params_json": intents.MustJSON(map[string]string{"command": cmd}),
				})}
			},
		},

		// ── Run command in terminal (without "open") ─────────────────
		// "run ls -la in terminal" / "نفذ git pull في التيرمنل"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:run|execute|نفذ|شغل)\s+(.+?)\s+(?:in\s+(?:terminal|term)|في\s+(?:التيرمنل|الطرفية|تيرمنل))$`),
			Build: func(m []string) []aimodel.ToolCall {
				cmd := strings.TrimSpace(m[1])
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "terminal_run_command",
					"params_json": intents.MustJSON(map[string]string{"command": cmd}),
				})}
			},
		},
	)

	intents.RegisterSimple(
		// ── Terminal cancel ───────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:terminal\s+cancel|cancel\s+terminal|الغ\s+(?:التيرمنل|الامر))\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{"name": "terminal_cancel"})}
			},
		},

		// ── Terminal clear ───────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:clear\s+terminal|terminal\s+clear|امسح\s+(?:التيرمنل|الشاشة)|نظف\s+(?:التيرمنل|الشاشة))\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{"name": "terminal_clear"})}
			},
		},
	)
}
