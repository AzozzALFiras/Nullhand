package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	aisvc "github.com/AzozzALFiras/Nullhand/internal/service/ai"
	recipesvc "github.com/AzozzALFiras/Nullhand/internal/service/recipe"
)

const maxSteps = 30

// ProgressFunc is called after each agent step with a status message.
type ProgressFunc func(message string)

// PhotoFunc is called when the AI wants to deliver a photo directly to the
// user. The implementation must send the bytes to Telegram and return.
// After this call the image bytes must be considered delivered and discarded.
type PhotoFunc func(data []byte, caption string) error

// ManualFocusFunc is called when the AI invokes request_manual_focus. The
// implementation must prompt the user, block until the user replies, and
// return the reply text. Errors indicate cancellation or timeout.
type ManualFocusFunc func(reason string) (string, error)

// BrowseFunc is called when the AI invokes browse_folder. The
// implementation opens an interactive file browser in Telegram.
type BrowseFunc func(path string)

// ViewModel orchestrates the multi-step AI agent loop.
type ViewModel struct {
	provider  aisvc.Provider
	recipes   *recipesvc.Service
	tools     []aimodel.ToolDefinition
	hasVision bool

	// Per-task clipboard hold for the verify → restore flow. Populated by
	// type_text, consumed by restore_clipboard. Not safe for concurrent
	// tasks — but tasks are serialized by the agent loop anyway.
	savedClipboard    string
	hadSavedClipboard bool

	// manualFocus is the callback used by request_manual_focus. Set per-Run
	// by the caller, may be nil.
	manualFocus ManualFocusFunc

	// browse is the callback used by browse_folder. Set per-Run by the caller.
	browse BrowseFunc

	// lastToolCalls stores the tool calls from the last Run for session tracking.
	lastToolCalls []aimodel.ToolCall
}

// Provider returns the underlying AI provider (for type assertions).
func (vm *ViewModel) Provider() aisvc.Provider {
	return vm.provider
}

// LastToolCalls returns the tool names and args from the last execution.
// Used by the bot to update session context.
func (vm *ViewModel) LastToolCalls() []aimodel.ToolCall {
	return vm.lastToolCalls
}

// New creates an agent ViewModel with the given AI provider and recipe service.
func New(provider aisvc.Provider, recipes *recipesvc.Service) *ViewModel {
	vm := &ViewModel{provider: provider, recipes: recipes}
	if vc, ok := provider.(aisvc.VisionCapable); ok {
		vm.hasVision = vc.SupportsVision()
	}
	vm.tools = vm.buildToolDefinitions()
	return vm
}

// buildSystemPrompt returns the system prompt, adapting instructions based on
// whether the AI provider supports vision.
func (vm *ViewModel) buildSystemPrompt() string {
	var sb strings.Builder

	sb.WriteString(`You are Nullhand, an AI agent that controls a desktop computer.
Complete tasks in the FEWEST tool calls possible.

`)

	if vm.hasVision {
		sb.WriteString(`## Vision
You CAN see the screen via analyze_screenshot. When you need to find a UI
element, navigate a complex interface, or determine where to click:
1. Call analyze_screenshot to see the current screen
2. Identify the target element and its approximate coordinates
3. Use click(x, y) to interact with it

Use vision for multi-step navigation when recipes are not available.
Use it sparingly — each call costs tokens.

`)
	} else {
		sb.WriteString(`## No Vision
You cannot see the screen. Use list_ui_elements and accessibility tools to
discover UI elements. Prefer run_recipe for common navigation tasks
(call list_recipes to see them). Only call take_screenshot when the user
explicitly asks to see the screen — the image goes to their Telegram,
you will not see it.

`)
	}

	sb.WriteString(`## Recipes
Call list_recipes to discover pre-built automation workflows for common tasks.
Use run_recipe with the recipe name and parameters.
ALWAYS check recipes first before doing manual multi-step navigation.

## Terminal Control
To run commands in Terminal:
- run_recipe("terminal_run_command", {"command":"ls -la"})
- Or: open_app("Terminal") → type_text("command") → press_key("return")
- Cancel running command: press_key("ctrl+c")
- End of input: press_key("ctrl+d")
- Clear screen: run_recipe("terminal_clear")

Sudo flow (when a command needs password):
1. Run the sudo command in Terminal
2. Terminal will ask for password
3. Call request_manual_focus("I need your sudo password to continue")
4. User sends password via Telegram
5. type_text(the_password) → press_key("return")

To read Terminal output: use analyze_screenshot (vision) or take_screenshot.

## Browser Control
- Open URL: run_recipe("browser_open_url", {"browser":"Firefox","url":"https://..."})
- Google search: run_recipe("browser_google_search", {"browser":"Firefox","query":"..."})
- New/close tab: run_recipe("browser_new_tab/browser_close_tab", {"browser":"Firefox"})
- Find in page: run_recipe("browser_find_in_page", {"browser":"Firefox","text":"..."})
- Navigate tabs: browser_next_tab, browser_prev_tab
- Back/forward: browser_back, browser_forward
- Reload: browser_reload
- Click links/buttons: use analyze_screenshot to see the page, then click(x, y)
- Default browser: "Firefox". For Chrome use "Google Chrome".

## VS Code / IDE Control
- Open terminal: run_recipe("vscode_open_terminal")
- Run command in terminal: run_recipe("vscode_terminal_run", {"command":"npm start"})
- Claude chat: run_recipe("vscode_type_in_claude", {"message":"..."})
- New Claude chat: run_recipe("vscode_new_claude_chat")
- Command palette: focus_via_palette("ctrl+shift+p", "command name")

## The canonical flow for "type X in app Y and send it"
1. Check if a recipe exists (list_recipes → run_recipe)
2. If no recipe: open_app → type_text → press_key("return")
3. For Electron apps (VS Code, Slack, Discord): use focus_via_palette or run_recipe

## App name tips
- VS Code → "Visual Studio Code", Chrome → "Google Chrome"
- Firefox, Slack, Terminal → use as-is

## Common shortcuts (for press_key)
- return/enter → submit, escape → cancel
- ctrl+l → browser URL bar, ctrl+f → find in page
- ctrl+t/ctrl+w → new/close tab, ctrl+tab → next tab
- ctrl+c → cancel command, ctrl+d → EOF, ctrl+z → suspend
- ctrl+shift+k → clear terminal, ctrl+r → reload page

## Rules
- Do ONLY what the user asked. Nothing more.
- Do NOT verify, inspect, or explore unless asked.
- If the task is "open X", call open_app once and STOP.
- Your final reply must be ONE short sentence.

`)
	return sb.String()
}

// Run executes a natural-language task and returns the final reply text.
// progress is called after each tool execution (may be nil to suppress updates).
// sendPhoto delivers screenshots directly to the user.
// manualFocus is invoked when the AI calls request_manual_focus (may be nil).
func (vm *ViewModel) Run(ctx context.Context, task string, progress ProgressFunc, sendPhoto PhotoFunc, manualFocus ManualFocusFunc, browse BrowseFunc) (string, error) {
	vm.manualFocus = manualFocus
	vm.browse = browse
	vm.lastToolCalls = nil
	history := []aimodel.Message{
		{
			Role:  aimodel.RoleSystem,
			Parts: []aimodel.MessagePart{{Type: aimodel.ContentTypeText, Text: vm.buildSystemPrompt()}},
		},
		{
			Role:  aimodel.RoleUser,
			Parts: []aimodel.MessagePart{{Type: aimodel.ContentTypeText, Text: task}},
		},
	}

	for step := 0; step < maxSteps; step++ {
		resp, err := vm.provider.Chat(ctx, history, vm.tools)
		if err != nil {
			return "", fmt.Errorf("agent: AI call failed: %w", err)
		}

		// Append assistant turn (include any tool calls so the next turn is valid).
		assistantParts := []aimodel.MessagePart{}
		if resp.Text != "" {
			assistantParts = append(assistantParts, aimodel.MessagePart{
				Type: aimodel.ContentTypeText,
				Text: resp.Text,
			})
		}
		history = append(history, aimodel.Message{
			Role:      aimodel.RoleAssistant,
			Parts:     assistantParts,
			ToolCalls: resp.ToolCalls,
		})

		if resp.Done || len(resp.ToolCalls) == 0 {
			return resp.Text, nil
		}

		// Execute each tool call and collect results.
		vm.lastToolCalls = append(vm.lastToolCalls, resp.ToolCalls...)
		for _, tc := range resp.ToolCalls {
			log.Printf("agent: tool_call %s args=%v", tc.ToolName, tc.Arguments)
			parts, toolErr := vm.executeTool(tc, sendPhoto)
			logText := partsText(parts)
			log.Printf("agent: tool_result %s → %q err=%v", tc.ToolName, truncate(logText, 120), toolErr)
			_ = toolErr

			if progress != nil {
				status := fmt.Sprintf("%s → %s", tc.ToolName, truncate(logText, 80))
				progress(status)
			}

			history = append(history, aimodel.Message{
				Role:       aimodel.RoleTool,
				Parts:      parts,
				ToolCallID: tc.ID,
			})
		}
	}

	return "", fmt.Errorf("agent: exceeded maximum steps (%d)", maxSteps)
}

// jsonUnmarshal is a thin wrapper so platform tool files can call it without
// importing encoding/json directly alongside their other imports.
func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// buildToolDefinitions returns the list of tools exposed to the AI.
func (vm *ViewModel) buildToolDefinitions() []aimodel.ToolDefinition {
	tools := []aimodel.ToolDefinition{
		{
			Name: "take_screenshot",
			Description: "Capture the screen and deliver the image directly to the user's " +
				"Telegram chat. You will NOT see the image. Only call when the user " +
				"explicitly asks to see the screen.",
		},
		{
			Name:        "open_app",
			Description: "Launch and focus an application by name.",
			Parameters: []aimodel.ToolParameter{
				{Name: "app_name", Type: "string", Description: "Application display name, e.g. Firefox, Visual Studio Code", Required: true},
			},
		},
		{
			Name:        "close_app",
			Description: "Close or kill an application by name. Sends a graceful close first, then force-kills if needed.",
			Parameters: []aimodel.ToolParameter{
				{Name: "app_name", Type: "string", Description: "Application display name, e.g. Firefox, Visual Studio Code", Required: true},
			},
		},
		{
			Name: "type_text",
			Description: "Paste text into the currently focused element via the clipboard. " +
				"Works for any language and any keyboard layout (Arabic, Chinese, emoji). " +
				"Clipboard is saved and restored automatically.",
			Parameters: []aimodel.ToolParameter{
				{Name: "text", Type: "string", Description: "Text to type", Required: true},
			},
		},
		{
			Name:        "press_key",
			Description: "Press a key or keyboard shortcut, e.g. return, cmd+t, escape, cmd+l.",
			Parameters: []aimodel.ToolParameter{
				{Name: "key", Type: "string", Description: "Key or shortcut string", Required: true},
			},
		},
		{
			Name:        "click",
			Description: "Left-click at the given screen coordinates.",
			Parameters:  xyParams(),
		},
		{
			Name:        "right_click",
			Description: "Right-click at the given screen coordinates.",
			Parameters:  xyParams(),
		},
		{
			Name:        "double_click",
			Description: "Double-click at the given screen coordinates.",
			Parameters:  xyParams(),
		},
		{
			Name:        "move_mouse",
			Description: "Move the cursor to the given screen coordinates without clicking.",
			Parameters:  xyParams(),
		},
		{
			Name:        "run_shell",
			Description: "Run a whitelisted shell command and return its output.",
			Parameters: []aimodel.ToolParameter{
				{Name: "command", Type: "string", Description: "Shell command to run", Required: true},
			},
		},
		{
			Name:        "read_file",
			Description: "Read and return the contents of a file.",
			Parameters: []aimodel.ToolParameter{
				{Name: "path", Type: "string", Description: "Absolute or ~ path to the file", Required: true},
			},
		},
		{
			Name: "browse_folder",
			Description: "Open an interactive file browser in Telegram with buttons. " +
				"The user can navigate folders, open projects in VS Code, run git commands, etc.",
			Parameters: []aimodel.ToolParameter{
				{Name: "path", Type: "string", Description: "Directory path to browse (supports ~, documents, etc.)", Required: true},
			},
		},
		{
			Name:        "list_directory",
			Description: "List the files and directories at the given path.",
			Parameters: []aimodel.ToolParameter{
				{Name: "path", Type: "string", Description: "Directory path (default: .)", Required: false},
			},
		},
		{Name: "get_clipboard", Description: "Return the current clipboard text content."},
		{
			Name:        "set_clipboard",
			Description: "Copy text to the clipboard.",
			Parameters: []aimodel.ToolParameter{
				{Name: "text", Type: "string", Description: "Text to copy", Required: true},
			},
		},
		// --- Recipes ---
		{
			Name: "list_recipes",
			Description: "List available automation recipes. Recipes are pre-built multi-step " +
				"workflows for common tasks like navigating to a WhatsApp contact, focusing " +
				"a Slack channel, opening a VS Code Claude chat, etc. Call this to discover " +
				"available recipes before using run_recipe.",
		},
		{
			Name: "run_recipe",
			Description: "Execute a named recipe. Recipes automate multi-step UI navigation " +
				"(e.g. whatsapp_new_message opens WhatsApp, Cmd+N, types contact name, presses return). " +
				"Use list_recipes first to see available recipes and their parameters.",
			Parameters: []aimodel.ToolParameter{
				{Name: "name", Type: "string", Description: "Recipe name from list_recipes", Required: true},
				{Name: "params_json", Type: "string", Description: `JSON object of parameters, e.g. {"contact":"John"}`, Required: false},
				{Name: "dry_run", Type: "string", Description: "Set to 'true' to preview steps without executing", Required: false},
			},
		},
		// --- Accessibility & navigation ---
		{
			Name: "click_ui_element",
			Description: "Click a UI element by its accessibility label in the frontmost app. " +
				"Works best with native macOS apps (Safari, Mail, Messages, etc.).",
			Parameters: []aimodel.ToolParameter{
				{Name: "app_name", Type: "string", Description: "App name (for error context)", Required: true},
				{Name: "label", Type: "string", Description: "Element name/title/description to click", Required: true},
			},
		},
		{
			Name: "focus_text_field",
			Description: "Focus a text input field by label in the frontmost native macOS app. " +
				"Only works for native Cocoa apps, not Electron apps like VS Code.",
			Parameters: []aimodel.ToolParameter{
				{Name: "app_name", Type: "string", Description: "App name", Required: true},
				{Name: "label", Type: "string", Description: "Placeholder, title, or description of the field", Required: true},
			},
		},
		{
			Name: "focus_via_palette",
			Description: "Open a command palette (e.g. Cmd+Shift+P in VS Code) and run a command. " +
				"Use this for Electron apps where focus_text_field doesn't work.",
			Parameters: []aimodel.ToolParameter{
				{Name: "palette_shortcut", Type: "string", Description: "Shortcut to open palette, e.g. cmd+shift+p", Required: true},
				{Name: "command_name", Type: "string", Description: "Command to type and execute in the palette", Required: true},
			},
		},
		{
			Name: "list_ui_elements",
			Description: "Dump the accessibility tree of the frontmost window. Returns element " +
				"classes, roles, titles, and descriptions. Use to discover clickable elements.",
			Parameters: []aimodel.ToolParameter{
				{Name: "max_depth", Type: "string", Description: "Max tree depth (default 8)", Required: false},
			},
		},
		{
			Name: "is_native_app",
			Description: "Check if an app exposes native accessibility elements. Returns 'native' " +
				"or 'electron'. If native, focus_text_field works. If electron, use recipes or palette.",
			Parameters: []aimodel.ToolParameter{
				{Name: "app_name", Type: "string", Description: "Application name to check", Required: true},
			},
		},
		// --- Mouse: scroll & drag ---
		{
			Name: "scroll",
			Description: "Scroll in a direction. Useful for navigating long pages, lists, or chat histories.",
			Parameters: []aimodel.ToolParameter{
				{Name: "direction", Type: "string", Description: "up, down, left, or right", Required: true},
				{Name: "steps", Type: "string", Description: "Number of scroll steps (default 3)", Required: false},
			},
		},
		{
			Name: "drag",
			Description: "Drag from one point to another (e.g. to move a window or slider).",
			Parameters: []aimodel.ToolParameter{
				{Name: "x1", Type: "integer", Description: "Start X coordinate", Required: true},
				{Name: "y1", Type: "integer", Description: "Start Y coordinate", Required: true},
				{Name: "x2", Type: "integer", Description: "End X coordinate", Required: true},
				{Name: "y2", Type: "integer", Description: "End Y coordinate", Required: true},
			},
		},
		// --- User interaction ---
		{
			Name: "request_manual_focus",
			Description: "Ask the user for help or information via Telegram. Use when you need " +
				"something from the user (like a sudo password, confirmation, or manual action). " +
				"Blocks until the user replies. The user's reply text is returned to you.",
			Parameters: []aimodel.ToolParameter{
				{Name: "reason", Type: "string", Description: "Explain what you need from the user", Required: true},
			},
		},
		// --- Timing control ---
		{
			Name: "wait",
			Description: "Wait for a specified number of milliseconds. Use between UI actions " +
				"to allow apps time to update (e.g. 300ms after opening an app, 200ms after pressing a key).",
			Parameters: []aimodel.ToolParameter{
				{Name: "ms", Type: "string", Description: "Milliseconds to wait (max 5000)", Required: true},
			},
		},
	}

	// Vision tool — only available when the AI provider supports images.
	if vm.hasVision {
		tools = append(tools, aimodel.ToolDefinition{
			Name: "analyze_screenshot",
			Description: "Capture the screen and add the image to this conversation so you " +
				"can see it. Use this to find UI elements, determine click coordinates, read " +
				"text on screen, and navigate complex interfaces. The logical screen resolution " +
				"is included — use those coordinates for click/move actions.",
		})
	}

	return tools
}

// textParts wraps a string in a single-element MessagePart slice.
func textParts(s string) []aimodel.MessagePart {
	return []aimodel.MessagePart{{Type: aimodel.ContentTypeText, Text: s}}
}

// partsText extracts the concatenated text from a slice of MessageParts.
func partsText(parts []aimodel.MessagePart) string {
	var sb strings.Builder
	for _, p := range parts {
		if p.Type == aimodel.ContentTypeText {
			sb.WriteString(p.Text)
		}
	}
	return sb.String()
}

// xyParams returns the standard x,y coordinate parameters.
func xyParams() []aimodel.ToolParameter {
	return []aimodel.ToolParameter{
		{Name: "x", Type: "integer", Description: "X screen coordinate", Required: true},
		{Name: "y", Type: "integer", Description: "Y screen coordinate", Required: true},
	}
}

func parseXY(args map[string]string) (int, int, error) {
	var x, y int
	if _, err := fmt.Sscanf(args["x"], "%d", &x); err != nil {
		return 0, 0, fmt.Errorf("invalid x: %v", err)
	}
	if _, err := fmt.Sscanf(args["y"], "%d", &y); err != nil {
		return 0, 0, fmt.Errorf("invalid y: %v", err)
	}
	return x, y, nil
}

func truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
