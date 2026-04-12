package antigravity

import (
	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
	"github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents"
)

const appName = "Antigravity"

// BuildFeature converts an Antigravity feature name into tool calls.
func BuildFeature(feature, command, message, query string) []aimodel.ToolCall {
	switch feature {
	case "terminal":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": appName}),
			intents.ToolCall("wait", map[string]string{"ms": "800"}),
			intents.ToolCall("press_key", map[string]string{"key": "ctrl+`"}),
		}

	case "terminal_run":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": appName}),
			intents.ToolCall("wait", map[string]string{"ms": "800"}),
			intents.ToolCall("press_key", map[string]string{"key": "ctrl+`"}),
			intents.ToolCall("wait", map[string]string{"ms": "400"}),
			intents.ToolCall("type_text", map[string]string{"text": command}),
			intents.ToolCall("press_key", map[string]string{"key": "return"}),
		}

	case "claude", "chat", "agent":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": appName}),
			intents.ToolCall("wait", map[string]string{"ms": "800"}),
			intents.ToolCall("press_key", map[string]string{"key": "cmd+shift+p"}),
			intents.ToolCall("wait", map[string]string{"ms": "300"}),
			intents.ToolCall("type_text", map[string]string{"text": "AI Chat"}),
			intents.ToolCall("press_key", map[string]string{"key": "return"}),
		}

	case "search":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": appName}),
			intents.ToolCall("wait", map[string]string{"ms": "500"}),
			intents.ToolCall("press_key", map[string]string{"key": "cmd+shift+f"}),
			intents.ToolCall("wait", map[string]string{"ms": "300"}),
			intents.ToolCall("type_text", map[string]string{"text": query}),
		}

	case "go_to_file":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": appName}),
			intents.ToolCall("wait", map[string]string{"ms": "500"}),
			intents.ToolCall("press_key", map[string]string{"key": "cmd+p"}),
			intents.ToolCall("wait", map[string]string{"ms": "300"}),
			intents.ToolCall("type_text", map[string]string{"text": query}),
		}

	case "git_push":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": appName}),
			intents.ToolCall("wait", map[string]string{"ms": "800"}),
			intents.ToolCall("press_key", map[string]string{"key": "ctrl+`"}),
			intents.ToolCall("wait", map[string]string{"ms": "400"}),
			intents.ToolCall("type_text", map[string]string{"text": "git add . && git commit -m 'update' && git push"}),
			intents.ToolCall("press_key", map[string]string{"key": "return"}),
		}

	case "git_pull":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": appName}),
			intents.ToolCall("wait", map[string]string{"ms": "800"}),
			intents.ToolCall("press_key", map[string]string{"key": "ctrl+`"}),
			intents.ToolCall("wait", map[string]string{"ms": "400"}),
			intents.ToolCall("type_text", map[string]string{"text": "git pull"}),
			intents.ToolCall("press_key", map[string]string{"key": "return"}),
		}

	case "settings":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": appName}),
			intents.ToolCall("press_key", map[string]string{"key": "cmd+,"}),
		}

	case "extensions":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": appName}),
			intents.ToolCall("press_key", map[string]string{"key": "cmd+shift+x"}),
		}

	default:
		return []aimodel.ToolCall{intents.ToolCall("open_app", map[string]string{"app_name": appName})}
	}
}
