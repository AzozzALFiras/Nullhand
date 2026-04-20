package cursor

import (
	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
)

const appName = "Cursor"

// BuildFeature converts a Cursor feature name into tool calls.
func BuildFeature(feature, command, message, query string) []aimodel.ToolCall {
	switch feature {
	case "terminal":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": appName}),
			intents.ToolCall("wait", map[string]string{"ms": "500"}),
			intents.ToolCall("press_key", map[string]string{"key": "ctrl+`"}),
		}

	case "terminal_run":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": appName}),
			intents.ToolCall("wait", map[string]string{"ms": "500"}),
			intents.ToolCall("press_key", map[string]string{"key": "ctrl+`"}),
			intents.ToolCall("wait", map[string]string{"ms": "400"}),
			intents.ToolCall("type_text", map[string]string{"text": command}),
			intents.ToolCall("press_key", map[string]string{"key": "return"}),
		}

	case "claude", "chat", "agent":
		return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{"name": "cursor_chat_focus"})}

	case "search":
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": appName}),
			intents.ToolCall("wait", map[string]string{"ms": "500"}),
			intents.ToolCall("press_key", map[string]string{"key": "cmd+shift+f"}),
			intents.ToolCall("wait", map[string]string{"ms": "300"}),
			intents.ToolCall("type_text", map[string]string{"text": query}),
		}

	default:
		return []aimodel.ToolCall{intents.ToolCall("open_app", map[string]string{"app_name": appName})}
	}
}
