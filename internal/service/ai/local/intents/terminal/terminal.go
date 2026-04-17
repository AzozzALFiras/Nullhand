package terminal

import (
	"regexp"
	"strings"

	aimodel "github.com/iamakillah/Nullhand_Linux/internal/model/ai"
	"github.com/iamakillah/Nullhand_Linux/internal/service/ai/local/intents"
)

// terminalNames matches any way of saying "terminal" in English or Arabic.
const terminalNames = `(?:terminal|term|iterm|the\s+terminal|التيرمنل|الطرفية|تيرمنل|الترمنل|طرفية)`

// actionWords matches optional action words between terminal and command.
const actionWords = `(?:and\s+)?(?:run|do|execute|type|write|enter|perform|use|try|نفذ|شغل|اكتب|نفّذ|سوي|جرب|استخدم)?`

func init() {
	intents.RegisterSmart(
		// ── Open terminal + anything = run it ─────────────────────────
		// "open terminal and do ls"
		// "open terminal and ls -la"
		// "open terminal ls"
		// "open terminal run git status"
		// "افتح التيرمنل ونفذ ls -la"
		// "افتح التيرمنل ls"
		// "افتح التيرمنل واكتب git pull"
		// "launch terminal and try npm install"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open|launch|start|افتح|شغل|شغّل)\s+` + terminalNames + `\s+(?:` + actionWords + `\s+)?(.+?)$`),
			Build: buildTerminalRun,
		},

		// ── "terminal {cmd}" (shortest form) ─────────────────────────
		// "terminal ls" / "terminal git status" / "تيرمنل ls -la"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^` + terminalNames + `\s+` + actionWords + `\s*(.+?)$`),
			Build: buildTerminalRun,
		},

		// ── "{cmd} in terminal" ───────────────────────────────────────
		// "run ls in terminal" / "do git pull in terminal"
		// "نفذ git pull في التيرمنل" / "ls -la في التيرمنل"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^` + actionWords + `\s*(.+?)\s+(?:in|on|في|بـ|داخل)\s+` + terminalNames + `$`),
			Build: buildTerminalRun,
		},

		// ── "in terminal {cmd}" ──────────────────────────────────────
		// "in terminal run ls" / "في التيرمنل نفذ git pull"
		// "في التيرمنل ls -la"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:in|on|في|بـ|داخل)\s+` + terminalNames + `\s+` + actionWords + `\s*(.+?)$`),
			Build: buildTerminalRun,
		},
	)

	intents.RegisterSimple(
		// ── Terminal cancel ───────────────────────────────────────────
		// "terminal cancel" / "cancel terminal" / "الغ التيرمنل"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:` + terminalNames + `\s+(?:cancel|stop|الغ|اوقف)|(?:cancel|stop|الغ|اوقف)\s+` + terminalNames + `)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{"name": "terminal_cancel"})}
			},
		},

		// ── Terminal clear ───────────────────────────────────────────
		// "terminal clear" / "clear terminal" / "امسح التيرمنل"
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:` + terminalNames + `\s+(?:clear|clean|امسح|نظف|مسح)|(?:clear|clean|امسح|نظف|مسح)\s+` + terminalNames + `)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{"name": "terminal_clear"})}
			},
		},
	)
}

// buildTerminalRun is the shared builder for all terminal run patterns.
func buildTerminalRun(m []string) []aimodel.ToolCall {
	// Find the last non-empty capture group (the command).
	cmd := ""
	for i := len(m) - 1; i >= 1; i-- {
		if strings.TrimSpace(m[i]) != "" {
			cmd = strings.TrimSpace(m[i])
			break
		}
	}
	if cmd == "" {
		return nil
	}
	return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
		"name":        "terminal_run_command",
		"params_json": intents.MustJSON(map[string]string{"command": cmd}),
	})}
}
