package system

import (
	"regexp"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
	"github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSimple(
		// ── Status ───────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:status|حالة|الحالة|معلومات|info)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_shell", map[string]string{
					"command": "echo CPU: $(top -l 1 | grep 'CPU usage' | head -1) && echo MEM: $(top -l 1 | grep 'PhysMem' | head -1)",
				})}
			},
		},

		// ── Apps list ────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:apps|running\s+apps|التطبيقات|البرامج|اعرض\s+التطبيقات)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_shell", map[string]string{
					"command": "osascript -e 'tell application \"System Events\" to get name of every process whose background only is false'",
				})}
			},
		},

		// ── Lock screen ──────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:lock|lock\s+screen|اقفل\s+الشاشة|قفل)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("press_key", map[string]string{"key": "ctrl+cmd+q"})}
			},
		},

		// ── Spotlight search ─────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:spotlight|سبوت\s*لايت)\s+(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
					"name":        "spotlight_search",
					"params_json": intents.MustJSON(map[string]string{"query": m[1]}),
				})}
			},
		},
	)
}
