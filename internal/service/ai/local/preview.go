package local

import (
	"fmt"
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	recipemodel "github.com/AzozzALFiras/Nullhand/internal/model/recipe"
)

// RecipeProvider is the minimal interface Preview needs to expand run_recipe
// calls into their constituent steps. Implemented by recipe.Service.Get.
type RecipeProvider interface {
	Get(name string) (recipemodel.Recipe, bool)
}

// previewRe detects "preview:" / "dry-run:" / "معاينة:" / "جرب:" prefixes.
// Capture group 1 = the inner natural-language command to preview.
var previewRe = regexp.MustCompile(`(?is)^(?:preview|dry[-_ ]?run|simulate|معاينة|جرّب|جرب)\s*[:\-]\s*(.+)$`)

// IsPreviewRequest reports whether text starts with a preview prefix and, if
// so, returns the inner command text. The bot uses this to route preview
// requests away from execution.
func IsPreviewRequest(text string) (inner string, ok bool) {
	m := previewRe.FindStringSubmatch(strings.TrimSpace(text))
	if m == nil {
		return "", false
	}
	return strings.TrimSpace(m[1]), true
}

// Preview turns a natural-language command into a human-readable execution
// plan WITHOUT performing any action. Returns the plan as a multi-line
// string suitable for sending back to the user.
//
// recipes is optional — if provided, run_recipe(...) calls are expanded into
// their nested steps so the user sees the full picture instead of just the
// recipe name.
func Preview(text string, recipes RecipeProvider) string {
	calls := Parse(text)
	if len(calls) == 0 {
		return "❌ I could not understand that command.\n\nNothing would be executed."
	}
	var sb strings.Builder
	sb.WriteString("🔎 Preview — these tool calls would run (NO actions performed):\n\n")
	step := 0
	for _, tc := range calls {
		step = appendCallPlan(&sb, tc, recipes, step, 0)
	}
	sb.WriteString("\nSend the same command without `preview:` to actually run it.")
	return sb.String()
}

// appendCallPlan writes one tool call's description, recursing into recipes.
// Returns the updated step counter.
func appendCallPlan(sb *strings.Builder, tc aimodel.ToolCall, recipes RecipeProvider, step, depth int) int {
	step++
	indent := strings.Repeat("  ", depth)
	sb.WriteString(fmt.Sprintf("%s[%d] %s%s\n", indent, step, formatCallHeader(tc), formatCallSuffix(tc)))

	// Recurse into recipe steps.
	if tc.ToolName == "run_recipe" && recipes != nil {
		name := tc.Arguments["name"]
		params := parseSimpleJSON(tc.Arguments["params_json"])
		if r, ok := recipes.Get(name); ok {
			for _, raw := range r.Steps {
				resolved := resolveStepPlaceholders(raw, params)
				step++
				sb.WriteString(fmt.Sprintf("%s    %d. %s\n", indent, step, describeStep(resolved)))
			}
		}
	}
	return step
}

// formatCallHeader returns the high-level tool name for the plan.
func formatCallHeader(tc aimodel.ToolCall) string {
	switch tc.ToolName {
	case "open_app":
		return fmt.Sprintf("Open application: %q", tc.Arguments["app_name"])
	case "close_app":
		return fmt.Sprintf("Close application: %q", tc.Arguments["app_name"])
	case "type_text":
		return fmt.Sprintf("Type text: %q", truncate(tc.Arguments["text"], 60))
	case "press_key":
		return fmt.Sprintf("Press key: %q", tc.Arguments["key"])
	case "click":
		return fmt.Sprintf("Click at coordinates (%s, %s)", tc.Arguments["x"], tc.Arguments["y"])
	case "click_text":
		return fmt.Sprintf("OCR-locate and click on screen: %q", tc.Arguments["text"])
	case "click_ui_element_fuzzy", "click_ui_element":
		return fmt.Sprintf("Click UI element: %q", tc.Arguments["label"])
	case "wait":
		return fmt.Sprintf("Wait %sms", tc.Arguments["ms"])
	case "wait_for_text":
		return fmt.Sprintf("Wait for screen text: %q", tc.Arguments["text"])
	case "wait_for_window":
		return fmt.Sprintf("Wait for window title containing: %q", tc.Arguments["title"])
	case "wait_for_element":
		return fmt.Sprintf("Wait for UI element: %q", tc.Arguments["label"])
	case "clear_field":
		return "Clear focused field (Ctrl+A then Delete)"
	case "take_screenshot":
		return "Take a screenshot and send to you"
	case "browse_folder":
		return fmt.Sprintf("Open file browser at %q", tc.Arguments["path"])
	case "run_shell":
		return fmt.Sprintf("Run shell command: %q", truncate(tc.Arguments["command"], 60))
	case "read_file":
		return fmt.Sprintf("Read file: %q", tc.Arguments["path"])
	case "list_directory":
		return fmt.Sprintf("List directory: %q", tc.Arguments["path"])
	case "set_clipboard":
		return fmt.Sprintf("Set clipboard to: %q", truncate(tc.Arguments["text"], 60))
	case "get_clipboard":
		return "Read clipboard contents"
	case "scroll":
		steps := tc.Arguments["steps"]
		if steps == "" {
			steps = "3"
		}
		return fmt.Sprintf("Scroll %s by %s steps", tc.Arguments["direction"], steps)
	case "run_recipe":
		return fmt.Sprintf("Run recipe %q", tc.Arguments["name"])
	case "focus_via_palette":
		return fmt.Sprintf("Open palette (%s) and run %q", tc.Arguments["palette_shortcut"], tc.Arguments["command_name"])
	case "list_recipes":
		return "List all available recipes"
	default:
		return fmt.Sprintf("%s(%v)", tc.ToolName, tc.Arguments)
	}
}

// formatCallSuffix appends recipe params for run_recipe so the user sees them.
func formatCallSuffix(tc aimodel.ToolCall) string {
	if tc.ToolName != "run_recipe" {
		return ""
	}
	if raw := tc.Arguments["params_json"]; raw != "" && raw != "{}" {
		return " with params " + truncate(raw, 80)
	}
	return ""
}

// describeStep formats a recipe step for the plan (parallel to the same-named
// helper in recipe_service.go but lives here so we don't import that package).
func describeStep(s recipemodel.Step) string {
	switch s.Kind {
	case recipemodel.StepOpenApp:
		return fmt.Sprintf("open_app(%q)", s.AppName)
	case recipemodel.StepPressKey:
		return fmt.Sprintf("press_key(%q)", s.Key)
	case recipemodel.StepTypeText:
		return fmt.Sprintf("type_text(%q)", truncate(s.Text, 50))
	case recipemodel.StepPalette:
		return fmt.Sprintf("palette(%q, %q)", s.Shortcut, s.Command)
	case recipemodel.StepSleepMs:
		return fmt.Sprintf("sleep(%dms)", s.Ms)
	case recipemodel.StepFocusField:
		return fmt.Sprintf("focus_field(%q)", s.Label)
	case recipemodel.StepWaitForWindow:
		return fmt.Sprintf("wait_for_window(%q, %dms)", s.Text, s.Ms)
	case recipemodel.StepWaitForText:
		return fmt.Sprintf("wait_for_text(%q, %dms)", truncate(s.Text, 40), s.Ms)
	case recipemodel.StepWaitForElement:
		return fmt.Sprintf("wait_for_element(%q, %dms)", s.Label, s.Ms)
	case recipemodel.StepClickText:
		return fmt.Sprintf("click_text(%q)", truncate(s.Text, 40))
	case recipemodel.StepClickFuzzy:
		return fmt.Sprintf("click_fuzzy(%q)", s.Label)
	case recipemodel.StepClearField:
		return "clear_field()"
	default:
		return string(s.Kind)
	}
}

// resolveStepPlaceholders substitutes {{name}} placeholders in step fields.
func resolveStepPlaceholders(s recipemodel.Step, params map[string]string) recipemodel.Step {
	subst := func(in string) string {
		if in == "" || len(params) == 0 {
			return in
		}
		out := in
		for k, v := range params {
			out = strings.ReplaceAll(out, "{{"+k+"}}", v)
		}
		return out
	}
	s.AppName = subst(s.AppName)
	s.Key = subst(s.Key)
	s.Text = subst(s.Text)
	s.Shortcut = subst(s.Shortcut)
	s.Command = subst(s.Command)
	s.Label = subst(s.Label)
	return s
}

// parseSimpleJSON is a tiny string-only JSON object parser for tool call
// params_json values. Returns an empty map on any parse error.
func parseSimpleJSON(s string) map[string]string {
	out := map[string]string{}
	s = strings.TrimSpace(s)
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return out
	}
	body := s[1 : len(s)-1]
	i := 0
	for i < len(body) {
		for i < len(body) && (body[i] == ' ' || body[i] == '\t' || body[i] == ',' || body[i] == '\n') {
			i++
		}
		if i >= len(body) || body[i] != '"' {
			return out
		}
		i++
		ks := i
		for i < len(body) && body[i] != '"' {
			i++
		}
		if i >= len(body) {
			return out
		}
		key := body[ks:i]
		i++
		for i < len(body) && (body[i] == ' ' || body[i] == ':') {
			i++
		}
		if i >= len(body) || body[i] != '"' {
			return out
		}
		i++
		vs := i
		for i < len(body) && body[i] != '"' {
			if body[i] == '\\' && i+1 < len(body) {
				i += 2
				continue
			}
			i++
		}
		out[key] = body[vs:i]
		if i < len(body) {
			i++
		}
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
