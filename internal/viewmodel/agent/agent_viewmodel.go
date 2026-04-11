package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
	aisvc "github.com/AzozzALFiras/nullhand/internal/service/ai"
	a11ysvc "github.com/AzozzALFiras/nullhand/internal/service/macos/accessibility"
	appsvc "github.com/AzozzALFiras/nullhand/internal/service/macos/apps"
	filesvc "github.com/AzozzALFiras/nullhand/internal/service/macos/files"
	kbsvc "github.com/AzozzALFiras/nullhand/internal/service/macos/keyboard"
	mousesvc "github.com/AzozzALFiras/nullhand/internal/service/macos/mouse"
	palettesvc "github.com/AzozzALFiras/nullhand/internal/service/macos/palette"
	screensvc "github.com/AzozzALFiras/nullhand/internal/service/macos/screen"
	shellsvc "github.com/AzozzALFiras/nullhand/internal/service/macos/shell"
	recipesvc "github.com/AzozzALFiras/nullhand/internal/service/recipe"
)

const (
	maxSteps     = 30
	systemPrompt = `You are Nullhand, an AI agent that controls a macOS computer.
Complete tasks in the FEWEST tool calls possible.

## You have no vision
You cannot see the screen. Never call click(x,y) unless the user gave exact
coordinates. Only call take_screenshot when the user explicitly asks to see
the screen — the image goes to their Telegram, you will not see it.

## The canonical flow for "type X in app Y and send it"
Exactly THREE tool calls. Do not add more unless the task explicitly needs it.

1. open_app("<full app name>")    ← launches AND focuses the app
2. type_text("<message>")          ← pastes via clipboard (works for any language)
3. press_key("return")             ← submits

That is the entire flow. Do NOT call list_recipes, verify_clipboard, or any
palette/focus tool preemptively.

## App name tips
- VS Code   → "Visual Studio Code"
- Chrome    → "Google Chrome"
- WhatsApp  → "WhatsApp"
- Slack     → "Slack"
- Safari    → "Safari"
- Messages  → "Messages"

## Common shortcuts (for press_key)
- return / enter   → submit
- cmd+return       → send in some rich editors
- cmd+l            → browser URL bar
- cmd+t / cmd+w    → new / close tab
- escape           → cancel

## Rules
- Do ONLY what the user asked. Nothing more.
- Do NOT verify, inspect, or explore unless asked.
- If the task is "open X", call open_app once and STOP.
- Your final reply must be ONE short sentence.

`
)

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

// ViewModel orchestrates the multi-step AI agent loop.
type ViewModel struct {
	provider aisvc.Provider
	recipes  *recipesvc.Service
	tools    []aimodel.ToolDefinition

	// Per-task clipboard hold for the verify → restore flow. Populated by
	// type_text, consumed by restore_clipboard. Not safe for concurrent
	// tasks — but tasks are serialized by the agent loop anyway.
	savedClipboard    string
	hadSavedClipboard bool

	// manualFocus is the callback used by request_manual_focus. Set per-Run
	// by the caller, may be nil.
	manualFocus ManualFocusFunc
}

// New creates an agent ViewModel with the given AI provider and recipe service.
func New(provider aisvc.Provider, recipes *recipesvc.Service) *ViewModel {
	vm := &ViewModel{provider: provider, recipes: recipes}
	vm.tools = vm.buildToolDefinitions()
	return vm
}

// Run executes a natural-language task and returns the final reply text.
// progress is called after each tool execution (may be nil to suppress updates).
// sendPhoto delivers screenshots directly to the user.
// manualFocus is invoked when the AI calls request_manual_focus (may be nil).
func (vm *ViewModel) Run(ctx context.Context, task string, progress ProgressFunc, sendPhoto PhotoFunc, manualFocus ManualFocusFunc) (string, error) {
	vm.manualFocus = manualFocus
	history := []aimodel.Message{
		{
			Role:  aimodel.RoleSystem,
			Parts: []aimodel.MessagePart{{Type: aimodel.ContentTypeText, Text: systemPrompt}},
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
		for _, tc := range resp.ToolCalls {
			log.Printf("agent: tool_call %s args=%v", tc.ToolName, tc.Arguments)
			output, toolErr := vm.executeTool(tc, sendPhoto)
			log.Printf("agent: tool_result %s → %q err=%v", tc.ToolName, truncate(output, 120), toolErr)
			_ = toolErr

			if progress != nil {
				status := fmt.Sprintf("%s → %s", tc.ToolName, truncate(output, 80))
				progress(status)
			}

			history = append(history, aimodel.Message{
				Role:       aimodel.RoleTool,
				Parts:      []aimodel.MessagePart{{Type: aimodel.ContentTypeText, Text: output}},
				ToolCallID: tc.ID,
			})
		}
	}

	return "", fmt.Errorf("agent: exceeded maximum steps (%d)", maxSteps)
}

// executeTool calls the appropriate macOS service based on tool name.
// Returns (output text, error). Screenshots are NEVER returned to the AI —
// they are delivered straight to the user via sendPhoto and the image bytes
// are discarded immediately.
func (vm *ViewModel) executeTool(tc aimodel.ToolCall, sendPhoto PhotoFunc) (string, error) {
	args := tc.Arguments

	switch tc.ToolName {
	case "take_screenshot":
		if sendPhoto == nil {
			return "error: screenshot delivery not available", fmt.Errorf("sendPhoto is nil")
		}
		data, err := screensvc.Capture()
		if err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		if err := sendPhoto(data, "📸"); err != nil {
			// Clear the reference so the GC can reclaim the bytes immediately.
			data = nil
			return fmt.Sprintf("error delivering screenshot: %v", err), err
		}
		// The bytes are now out of our hands; drop the reference.
		data = nil
		return "screenshot delivered to user (AI cannot see it)", nil

	case "open_app":
		app := args["app_name"]
		if err := appsvc.Open(app); err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return fmt.Sprintf("opened %s", app), nil

	case "click":
		x, y, err := parseXY(args)
		if err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		if err := mousesvc.Click(x, y); err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return fmt.Sprintf("clicked at %d,%d", x, y), nil

	case "click_ui_element":
		app := args["app_name"]
		label := args["label"]
		if err := a11ysvc.Click(app, label); err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return fmt.Sprintf("clicked %q in %s", label, app), nil

	case "focus_text_field":
		app := args["app_name"]
		label := args["label"]
		if err := a11ysvc.FocusField(app, label); err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return fmt.Sprintf("focused text field %q in %s", label, app), nil

	case "focus_via_palette":
		shortcut := args["palette_shortcut"]
		command := args["command_name"]
		if err := palettesvc.Run(shortcut, command); err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return fmt.Sprintf("invoked palette command %q", command), nil

	case "list_ui_elements":
		depth := 8
		if d := args["max_depth"]; d != "" {
			_, _ = fmt.Sscanf(d, "%d", &depth)
		}
		tree, err := a11ysvc.ListElements(depth)
		if err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return tree, nil

	case "is_native_app":
		app := args["app_name"]
		if appsvc.IsNativeAX(app) {
			return "native (focus_text_field will work)", nil
		}
		return "electron or unknown (use focus_via_palette or run_recipe)", nil

	case "list_recipes":
		if vm.recipes == nil {
			return "no recipes available", nil
		}
		var sb strings.Builder
		for _, r := range vm.recipes.List() {
			sb.WriteString(r.Name)
			if r.Description != "" {
				sb.WriteString(" — ")
				sb.WriteString(r.Description)
			}
			if len(r.Parameters) > 0 {
				sb.WriteString(" [params: ")
				sb.WriteString(strings.Join(r.Parameters, ", "))
				sb.WriteString("]")
			}
			sb.WriteString("\n")
		}
		return strings.TrimRight(sb.String(), "\n"), nil

	case "run_recipe":
		if vm.recipes == nil {
			return "error: recipe service not available", fmt.Errorf("recipes disabled")
		}
		name := args["name"]
		params := map[string]string{}
		if raw := args["params_json"]; raw != "" {
			if err := json.Unmarshal([]byte(raw), &params); err != nil {
				return fmt.Sprintf("error: bad params_json: %v", err), err
			}
		}
		dryRun := args["dry_run"] == "true"
		plan, err := vm.recipes.Run(name, params, dryRun)
		if err != nil {
			return fmt.Sprintf("error: %v\n%s", err, plan), err
		}
		if dryRun {
			return fmt.Sprintf("dry-run ok:\n%s", plan), nil
		}
		return fmt.Sprintf("recipe %q ok:\n%s", name, plan), nil

	case "right_click":
		x, y, err := parseXY(args)
		if err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		if err := mousesvc.RightClick(x, y); err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return fmt.Sprintf("right-clicked at %d,%d", x, y), nil

	case "double_click":
		x, y, err := parseXY(args)
		if err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		if err := mousesvc.DoubleClick(x, y); err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return fmt.Sprintf("double-clicked at %d,%d", x, y), nil

	case "move_mouse":
		x, y, err := parseXY(args)
		if err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		if err := mousesvc.Move(x, y); err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return fmt.Sprintf("moved mouse to %d,%d", x, y), nil

	case "type_text":
		text := args["text"]
		if err := kbsvc.Type(text); err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return "typed text", nil

	case "verify_clipboard":
		got := kbsvc.ReadClipboard()
		return got, nil

	case "restore_clipboard":
		if vm.hadSavedClipboard {
			kbsvc.RestoreClipboard(vm.savedClipboard)
			vm.savedClipboard = ""
			vm.hadSavedClipboard = false
			return "clipboard restored", nil
		}
		return "nothing to restore", nil

	case "request_manual_focus":
		if vm.manualFocus == nil {
			return "error: manual focus not available in this context", fmt.Errorf("no manualFocus callback")
		}
		reason := args["reason"]
		if reason == "" {
			reason = "I could not locate the target input automatically."
		}
		reply, err := vm.manualFocus(reason)
		if err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return fmt.Sprintf("user confirmed: %s", truncate(reply, 60)), nil

	case "press_key":
		key := args["key"]
		if err := kbsvc.PressKey(key); err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return fmt.Sprintf("pressed %s", key), nil

	case "run_shell":
		cmd := args["command"]
		out, err := shellsvc.Run(cmd)
		if err != nil {
			return fmt.Sprintf("error: %v\n%s", err, out), err
		}
		return out, nil

	case "read_file":
		path := args["path"]
		content, err := filesvc.Read(path)
		if err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return content, nil

	case "list_directory":
		path := args["path"]
		if path == "" {
			path = "."
		}
		entries, err := filesvc.List(path)
		if err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return strings.Join(entries, "\n"), nil

	case "get_clipboard":
		text, err := filesvc.GetClipboard()
		if err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return text, nil

	case "set_clipboard":
		text := args["text"]
		if err := filesvc.SetClipboard(text); err != nil {
			return fmt.Sprintf("error: %v", err), err
		}
		return "clipboard set", nil

	default:
		return fmt.Sprintf("unknown tool: %s", tc.ToolName), nil
	}
}

// buildToolDefinitions returns the list of tools exposed to the AI.
// Kept minimal on purpose — the three-step flow (open_app → type_text →
// press_key) handles the vast majority of tasks. Advanced helpers
// (run_recipe, focus_via_palette, verify_clipboard, etc.) exist in the
// codebase and can be re-exposed here if a scenario actually needs them.
//
// Privacy note: take_screenshot captures the screen and sends the image
// DIRECTLY to the user's Telegram chat. The AI receives only a short text
// confirmation — it cannot view the image.
func (vm *ViewModel) buildToolDefinitions() []aimodel.ToolDefinition {
	return []aimodel.ToolDefinition{
		{
			Name: "take_screenshot",
			Description: "Capture the screen and deliver the image directly to the user's " +
				"Telegram chat. You will NOT see the image. Only call when the user " +
				"explicitly asks to see the screen.",
		},
		{
			Name:        "open_app",
			Description: "Launch and focus a macOS application by name.",
			Parameters: []aimodel.ToolParameter{
				{Name: "app_name", Type: "string", Description: "Application display name, e.g. Safari, Visual Studio Code", Required: true},
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
			Description: "Left-click at the given screen coordinates. Only use if the user gave exact coordinates.",
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
	}
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
