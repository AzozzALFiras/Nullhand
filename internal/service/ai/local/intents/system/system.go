package system

import (
	"regexp"

	aimodel "github.com/iamakillah/Nullhand_Linux/internal/model/ai"
	"github.com/iamakillah/Nullhand_Linux/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSimple(
		// ── Status ───────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:status|حالة|الحالة|معلومات|info)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_shell", map[string]string{
					"command": "echo CPU: $(top -bn1 | grep '%Cpu' | head -1) && echo MEM: $(top -bn1 | grep 'MiB Mem' | head -1)",
				})}
			},
		},

		// ── Apps list ────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:apps|running\s+apps|التطبيقات|البرامج|اعرض\s+التطبيقات)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_shell", map[string]string{
					"command": "wmctrl -l | awk '{print $4}'",
				})}
			},
		},

		// ── Lock screen ──────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:lock|lock\s+screen|اقفل\s+الشاشة|قفل)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_shell", map[string]string{
					"command": "xdg-screensaver lock",
				})}
			},
		},

		// ── Spotlight search (replaced with rofi / app launcher) ─────
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
