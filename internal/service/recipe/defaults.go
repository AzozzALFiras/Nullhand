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
		"whatsapp_send_message": {
			Description: "Open a WhatsApp chat and send a message (params: contact, message)",
			Parameters:  []string{"contact", "message"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "WhatsApp"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+n"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
				{Kind: recipemodel.StepTypeText, Text: "{{contact}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
				{Kind: recipemodel.StepPressKey, Key: "return"},
				{Kind: recipemodel.StepSleepMs, Ms: 400},
				{Kind: recipemodel.StepTypeText, Text: "{{message}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPressKey, Key: "return"},
			},
		},
		"vscode_type_in_claude": {
			Description: "Focus VS Code Claude chat and type a message (param: message)",
			Parameters:  []string{"message"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Visual Studio Code"},
				{Kind: recipemodel.StepSleepMs, Ms: 800},
				{Kind: recipemodel.StepPalette, Shortcut: "cmd+shift+p", Command: "Claude Code: Focus Chat"},
				{Kind: recipemodel.StepSleepMs, Ms: 600},
				{Kind: recipemodel.StepTypeText, Text: "{{message}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPressKey, Key: "return"},
			},
		},
		"slack_send_message": {
			Description: "Send a message to a Slack channel (params: channel, message)",
			Parameters:  []string{"channel", "message"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Slack"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+k"},
				{Kind: recipemodel.StepSleepMs, Ms: 250},
				{Kind: recipemodel.StepTypeText, Text: "{{channel}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPressKey, Key: "return"},
				{Kind: recipemodel.StepSleepMs, Ms: 400},
				{Kind: recipemodel.StepTypeText, Text: "{{message}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPressKey, Key: "return"},
			},
		},
		"discord_send_dm": {
			Description: "Send a Discord DM (params: recipient, message)",
			Parameters:  []string{"recipient", "message"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Discord"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+k"},
				{Kind: recipemodel.StepSleepMs, Ms: 250},
				{Kind: recipemodel.StepTypeText, Text: "{{recipient}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPressKey, Key: "return"},
				{Kind: recipemodel.StepSleepMs, Ms: 400},
				{Kind: recipemodel.StepTypeText, Text: "{{message}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPressKey, Key: "return"},
			},
		},
		"messages_send_imessage": {
			Description: "Send an iMessage (params: contact, message)",
			Parameters:  []string{"contact", "message"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Messages"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+n"},
				{Kind: recipemodel.StepSleepMs, Ms: 250},
				{Kind: recipemodel.StepTypeText, Text: "{{contact}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPressKey, Key: "return"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
				{Kind: recipemodel.StepTypeText, Text: "{{message}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPressKey, Key: "return"},
			},
		},
		"finder_go_to": {
			Description: "Open Finder Go To Folder dialog (param: path)",
			Parameters:  []string{"path"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Finder"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+shift+g"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
				{Kind: recipemodel.StepTypeText, Text: "{{path}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepPressKey, Key: "return"},
			},
		},

		// ── Terminal recipes ─────────────────────────────────────────────
		"terminal_run_command": {
			Description: "Open Terminal, type a command and run it (param: command)",
			Parameters:  []string{"command"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Terminal"},
				{Kind: recipemodel.StepSleepMs, Ms: 800},
				{Kind: recipemodel.StepTypeText, Text: "{{command}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
				{Kind: recipemodel.StepPressKey, Key: "return"},
			},
		},
		"terminal_cancel": {
			Description: "Send Ctrl+C to cancel the running command in Terminal",
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Terminal"},
				{Kind: recipemodel.StepSleepMs, Ms: 150},
				{Kind: recipemodel.StepPressKey, Key: "ctrl+c"},
			},
		},
		"terminal_eof": {
			Description: "Send Ctrl+D (end of input / exit) in Terminal",
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Terminal"},
				{Kind: recipemodel.StepSleepMs, Ms: 150},
				{Kind: recipemodel.StepPressKey, Key: "ctrl+d"},
			},
		},
		"terminal_clear": {
			Description: "Clear the Terminal screen",
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Terminal"},
				{Kind: recipemodel.StepSleepMs, Ms: 150},
				{Kind: recipemodel.StepPressKey, Key: "cmd+k"},
			},
		},

		// ── Browser recipes ──────────────────────────────────────────────
		"browser_open_url": {
			Description: "Open a URL in a browser (params: browser, url)",
			Parameters:  []string{"browser", "url"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "{{browser}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
				{Kind: recipemodel.StepPressKey, Key: "cmd+l"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepTypeText, Text: "{{url}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 150},
				{Kind: recipemodel.StepPressKey, Key: "return"},
			},
		},
		"browser_google_search": {
			Description: "Search Google in a browser (params: browser, query)",
			Parameters:  []string{"browser", "query"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "{{browser}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
				{Kind: recipemodel.StepPressKey, Key: "cmd+l"},
				{Kind: recipemodel.StepSleepMs, Ms: 200},
				{Kind: recipemodel.StepTypeText, Text: "{{query}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 150},
				{Kind: recipemodel.StepPressKey, Key: "return"},
			},
		},
		"browser_new_tab": {
			Description: "Open a new browser tab (param: browser)",
			Parameters:  []string{"browser"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "{{browser}}"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+t"},
			},
		},
		"browser_close_tab": {
			Description: "Close the current browser tab (param: browser)",
			Parameters:  []string{"browser"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "{{browser}}"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+w"},
			},
		},
		"browser_find_in_page": {
			Description: "Find text in the current page (params: browser, text)",
			Parameters:  []string{"browser", "text"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "{{browser}}"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+f"},
				{Kind: recipemodel.StepSleepMs, Ms: 250},
				{Kind: recipemodel.StepTypeText, Text: "{{text}}"},
			},
		},
		"browser_next_tab": {
			Description: "Switch to the next browser tab (param: browser)",
			Parameters:  []string{"browser"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "{{browser}}"},
				{Kind: recipemodel.StepPressKey, Key: "ctrl+tab"},
			},
		},
		"browser_prev_tab": {
			Description: "Switch to the previous browser tab (param: browser)",
			Parameters:  []string{"browser"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "{{browser}}"},
				{Kind: recipemodel.StepPressKey, Key: "ctrl+shift+tab"},
			},
		},
		"browser_back": {
			Description: "Go back in browser history (param: browser)",
			Parameters:  []string{"browser"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "{{browser}}"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+["},
			},
		},
		"browser_forward": {
			Description: "Go forward in browser history (param: browser)",
			Parameters:  []string{"browser"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "{{browser}}"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+]"},
			},
		},
		"browser_reload": {
			Description: "Reload the current page (param: browser)",
			Parameters:  []string{"browser"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "{{browser}}"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+r"},
			},
		},

		// ── VS Code terminal recipes ─────────────────────────────────────
		"vscode_open_terminal": {
			Description: "Open/focus the integrated terminal in VS Code",
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Visual Studio Code"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
				{Kind: recipemodel.StepPalette, Shortcut: "cmd+shift+p", Command: "Terminal: Focus Terminal"},
			},
		},
		"vscode_new_terminal": {
			Description: "Create a new terminal instance in VS Code",
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Visual Studio Code"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
				{Kind: recipemodel.StepPalette, Shortcut: "cmd+shift+p", Command: "Terminal: Create New Terminal"},
			},
		},
		"vscode_terminal_run": {
			Description: "Open VS Code terminal and run a command (param: command)",
			Parameters:  []string{"command"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "Visual Studio Code"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
				{Kind: recipemodel.StepPalette, Shortcut: "cmd+shift+p", Command: "Terminal: Focus Terminal"},
				{Kind: recipemodel.StepSleepMs, Ms: 300},
				{Kind: recipemodel.StepTypeText, Text: "{{command}}"},
				{Kind: recipemodel.StepSleepMs, Ms: 150},
				{Kind: recipemodel.StepPressKey, Key: "return"},
			},
		},
	}
}
