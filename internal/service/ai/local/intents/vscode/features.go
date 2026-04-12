package vscode

import (
	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
	"github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents"
)

// BuildFeature converts a VS Code feature name into tool calls.
func BuildFeature(feature, command, message, query string) []aimodel.ToolCall {
	switch feature {
	case "terminal":
		return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{"name": "vscode_open_terminal"})}

	case "terminal_run":
		return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
			"name":        "vscode_terminal_run",
			"params_json": intents.MustJSON(map[string]string{"command": command}),
		})}

	case "claude":
		if message != "" {
			return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
				"name":        "vscode_type_in_claude",
				"params_json": intents.MustJSON(map[string]string{"message": message}),
			})}
		}
		return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{"name": "vscode_claude_chat_focus"})}

	case "new_claude":
		return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{"name": "vscode_new_claude_chat"})}

	case "search":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": "Visual Studio Code"}),
			intents.ToolCall("wait", map[string]string{"ms": "500"}),
			intents.ToolCall("press_key", map[string]string{"key": "cmd+shift+f"}),
			intents.ToolCall("wait", map[string]string{"ms": "300"}),
			intents.ToolCall("type_text", map[string]string{"text": query}),
		}

	case "go_to_file":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": "Visual Studio Code"}),
			intents.ToolCall("wait", map[string]string{"ms": "500"}),
			intents.ToolCall("press_key", map[string]string{"key": "cmd+p"}),
			intents.ToolCall("wait", map[string]string{"ms": "300"}),
			intents.ToolCall("type_text", map[string]string{"text": query}),
		}

	case "git_push":
		return []aimodel.ToolCall{
			intents.ToolCall("run_recipe", map[string]string{"name": "vscode_open_terminal"}),
			intents.ToolCall("wait", map[string]string{"ms": "500"}),
			intents.ToolCall("type_text", map[string]string{"text": "git add . && git commit -m 'update' && git push"}),
			intents.ToolCall("press_key", map[string]string{"key": "return"}),
		}

	case "git_pull":
		return []aimodel.ToolCall{
			intents.ToolCall("run_recipe", map[string]string{"name": "vscode_open_terminal"}),
			intents.ToolCall("wait", map[string]string{"ms": "500"}),
			intents.ToolCall("type_text", map[string]string{"text": "git pull"}),
			intents.ToolCall("press_key", map[string]string{"key": "return"}),
		}

	case "git_commit":
		return []aimodel.ToolCall{
			intents.ToolCall("run_recipe", map[string]string{"name": "vscode_open_terminal"}),
			intents.ToolCall("wait", map[string]string{"ms": "500"}),
			intents.ToolCall("type_text", map[string]string{"text": "git add . && git commit -m 'update'"}),
			intents.ToolCall("press_key", map[string]string{"key": "return"}),
		}

	case "git_status":
		return []aimodel.ToolCall{
			intents.ToolCall("run_recipe", map[string]string{"name": "vscode_open_terminal"}),
			intents.ToolCall("wait", map[string]string{"ms": "500"}),
			intents.ToolCall("type_text", map[string]string{"text": "git status"}),
			intents.ToolCall("press_key", map[string]string{"key": "return"}),
		}

	case "settings":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": "Visual Studio Code"}),
			intents.ToolCall("press_key", map[string]string{"key": "cmd+,"}),
		}

	case "extensions":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": "Visual Studio Code"}),
			intents.ToolCall("press_key", map[string]string{"key": "cmd+shift+x"}),
		}

	case "palette":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": "Visual Studio Code"}),
			intents.ToolCall("wait", map[string]string{"ms": "500"}),
			intents.ToolCall("press_key", map[string]string{"key": "cmd+shift+p"}),
		}

	case "new_file":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": "Visual Studio Code"}),
			intents.ToolCall("press_key", map[string]string{"key": "cmd+n"}),
		}

	case "close_all":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": "Visual Studio Code"}),
			intents.ToolCall("press_key", map[string]string{"key": "cmd+k"}),
			intents.ToolCall("wait", map[string]string{"ms": "100"}),
			intents.ToolCall("press_key", map[string]string{"key": "cmd+w"}),
		}

	default:
		return []aimodel.ToolCall{intents.ToolCall("open_app", map[string]string{"app_name": "Visual Studio Code"})}
	}
}
