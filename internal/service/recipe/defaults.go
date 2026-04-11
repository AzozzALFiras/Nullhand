package recipe

import recipemodel "github.com/AzozzALFiras/nullhand/internal/model/recipe"

// Defaults returns the built-in recipes shipped with Nullhand. The user can
// override any of these by adding an entry with the same name to
// ~/.nullhand/recipes.json.
func Defaults() map[string]recipemodel.Recipe {
	return map[string]recipemodel.Recipe{
		"vscode_claude_chat_focus": {
			Description: "Focus the Claude Code chat input inside VS Code",
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Visual Studio Code"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPalette, Shortcut: "cmd+shift+p", Command: "Claude Code: Focus Chat"},
			},
		},
		"vscode_new_claude_chat": {
			Description: "Open a fresh Claude Code chat in VS Code",
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Visual Studio Code"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPalette, Shortcut: "cmd+shift+p", Command: "Claude Code: New Chat"},
			},
		},
		"cursor_chat_focus": {
			Description: "Focus the Cursor AI chat input",
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Cursor"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPalette, Shortcut: "cmd+shift+p", Command: "Cursor: Focus Chat"},
			},
		},
		"slack_channel_focus": {
			Description: "Jump to a Slack channel and focus its message box (param: channel)",
			Parameters:  []string{"channel"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Slack"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+k"},
				{Kind: recipemodel.StepSleepMs, Ms: 250},
				{Kind: recipemodel.StepTypeText, Text: "{{channel}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPressKey, Key: "return"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
			},
		},
		"discord_dm_focus": {
			Description: "Open a Discord DM via quick switcher (param: recipient)",
			Parameters:  []string{"recipient"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Discord"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+k"},
				{Kind: recipemodel.StepSleepMs, Ms: 250},
				{Kind: recipemodel.StepTypeText, Text: "{{recipient}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPressKey, Key: "return"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
			},
		},
		"whatsapp_new_message": {
			Description: "Start a new WhatsApp chat (param: contact)",
			Parameters:  []string{"contact"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "WhatsApp"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+n"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
				{Kind: recipemodel.StepTypeText, Text: "{{contact}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPressKey, Key: "return"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
			},
		},
		"messages_new_imessage": {
			Description: "Start a new iMessage conversation (param: contact)",
			Parameters:  []string{"contact"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Messages"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+n"},
				{Kind: recipemodel.StepSleepMs, Ms: 250},
				{Kind: recipemodel.StepTypeText, Text: "{{contact}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPressKey, Key: "return"},
				{Kind: recipemodel.StepSleepMs, Ms: 250},
			},
		},
		"mail_new_message": {
			Description: "Open a new Mail compose window",
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Mail"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+n"},
				{Kind: recipemodel.StepSleepMs, Ms: 250},
			},
		},
		"browser_url_bar": {
			Description: "Focus the browser address bar (param: browser)",
			Parameters:  []string{"browser"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "{{browser}}"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+l"},
			},
		},
		"spotlight_search": {
			Description: "Open Spotlight and run a query (param: query)",
			Parameters:  []string{"query"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepPressKey, Key: "cmd+space"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepTypeText, Text: "{{query}}"},
			},
		},
		"notion_quick_find": {
			Description: "Open Notion quick find (param: query)",
			Parameters:  []string{"query"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Notion"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+p"},
				{Kind: recipemodel.StepSleepMs, Ms: 250},
				{Kind: recipemodel.StepTypeText, Text: "{{query}}"},
			},
		},
		"obsidian_quick_open": {
			Description: "Open Obsidian's quick switcher (param: query)",
			Parameters:  []string{"query"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Obsidian"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+o"},
				{Kind: recipemodel.StepSleepMs, Ms: 250},
				{Kind: recipemodel.StepTypeText, Text: "{{query}}"},
			},
		},
		"terminal_focus": {
			Description: "Focus Terminal (param: terminal — e.g. Terminal or iTerm)",
			Parameters:  []string{"terminal"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "{{terminal}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 150},
			},
		},
	}
}
