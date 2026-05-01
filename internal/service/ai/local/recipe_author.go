package local

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	recipemodel "github.com/AzozzALFiras/Nullhand/internal/model/recipe"
)

// AuthorRequest is what ParseAuthorRequest returns when the user asks to save
// a new recipe via natural language.
type AuthorRequest struct {
	Name        string             // recipe identifier (snake_case, no spaces)
	Description string             // human label
	Recipe      recipemodel.Recipe // ready-to-store recipe
}

// authorRe matches both English and Arabic "save this as recipe X:" openers.
// Capture group 1 = recipe name (1-4 words). Group 2 = step list.
var authorRe = regexp.MustCompile(`(?is)^(?:` +
	// English: "save this as recipe X: ..." / "save recipe X: ..." / "remember as routine X: ..."
	`(?:save|remember|store)\s+(?:this\s+)?(?:as\s+)?(?:a\s+)?(?:recipe|routine|workflow|macro)\s+([\p{L}\p{N}_\- ]{1,40}?)\s*[:\-]\s*(.+)` +
	`|` +
	// Arabic: "احفظ هذا كروتين X: ..." / "احفظ روتين X: ..." / "احفظ كوصفة X: ..."
	`(?:احفظ|احفظ\s+لي|سجّل|سجل)\s+(?:هذا\s+|هذه\s+|لي\s+)?(?:ك|ال)?(?:روتين|الروتين|وصفة|الوصفة|ماكرو)\s+([\p{L}\p{N}_\- ]{1,40}?)\s*[:\-]\s*(.+)` +
	`)$`)

// stepSplitRe splits a step list on commas, semicolons, "then"/"and"/"ثم"/newlines.
var stepSplitRe = regexp.MustCompile(`(?i)\s*(?:[,;،]|\sthen\s|\sand\s|\sثم\s|\n)\s*`)

// snakeRe replaces non-alphanumeric runs with single underscores.
var snakeRe = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// ParseAuthorRequest detects "save this as recipe X: step1, step2, ..." in
// either English or Arabic and converts each step into a recipemodel.Step.
//
// Returns nil if the message is not a recipe-authoring request. Returns an
// error if the request matches but the steps could not be parsed (e.g.
// unsupported tool call, empty step list).
func ParseAuthorRequest(text string) (*AuthorRequest, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, nil
	}

	m := authorRe.FindStringSubmatch(text)
	if m == nil {
		return nil, nil
	}
	// authorRe has two alternatives, each with two captures. Pick the first
	// non-empty pair (group 1+2 = English; group 3+4 = Arabic).
	rawName := firstNonEmpty(m[1], m[3])
	stepList := firstNonEmpty(m[2], m[4])
	rawName = strings.TrimSpace(rawName)
	stepList = strings.TrimSpace(stepList)
	if rawName == "" || stepList == "" {
		return nil, fmt.Errorf("recipe-author: missing name or steps")
	}

	name := normalizeName(rawName)
	if name == "" {
		return nil, fmt.Errorf("recipe-author: invalid recipe name %q", rawName)
	}

	rawSteps := stepSplitRe.Split(stepList, -1)
	var steps []recipemodel.Step
	for i, s := range rawSteps {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		// Reuse the existing parser to turn each step into one or more tool calls.
		calls := Parse(s)
		if len(calls) == 0 {
			return nil, fmt.Errorf("recipe-author: step %d (%q) is not understood", i+1, s)
		}
		for _, tc := range calls {
			step, err := toolCallToStep(tc)
			if err != nil {
				return nil, fmt.Errorf("recipe-author: step %d (%q): %w", i+1, s, err)
			}
			steps = append(steps, step)
		}
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("recipe-author: no usable steps")
	}

	return &AuthorRequest{
		Name:        name,
		Description: "User-defined recipe: " + rawName,
		Recipe: recipemodel.Recipe{
			Name:        name,
			Description: "User-defined recipe: " + rawName,
			Steps:       steps,
		},
	}, nil
}

// toolCallToStep maps a single tool call from the parser into a Recipe step.
// Tool calls that are not representable as a step (e.g. take_screenshot,
// browse_folder, run_shell) yield an error so the user knows.
func toolCallToStep(tc aimodel.ToolCall) (recipemodel.Step, error) {
	args := tc.Arguments
	switch tc.ToolName {
	case "open_app":
		return recipemodel.Step{Kind: recipemodel.StepOpenApp, AppName: args["app_name"]}, nil
	case "type_text":
		return recipemodel.Step{Kind: recipemodel.StepTypeText, Text: args["text"]}, nil
	case "press_key":
		return recipemodel.Step{Kind: recipemodel.StepPressKey, Key: args["key"]}, nil
	case "wait":
		ms := 0
		if v := args["ms"]; v != "" {
			ms, _ = strconv.Atoi(v)
		}
		return recipemodel.Step{Kind: recipemodel.StepSleepMs, Ms: ms}, nil
	case "click_text":
		return recipemodel.Step{Kind: recipemodel.StepClickText, Text: args["text"]}, nil
	case "click_ui_element_fuzzy", "click_ui_element":
		return recipemodel.Step{Kind: recipemodel.StepClickFuzzy, Label: args["label"]}, nil
	case "wait_for_text":
		ms, _ := strconv.Atoi(args["timeout_ms"])
		if ms == 0 {
			ms = 5000
		}
		return recipemodel.Step{Kind: recipemodel.StepWaitForText, Text: args["text"], Ms: ms}, nil
	case "wait_for_window":
		ms, _ := strconv.Atoi(args["timeout_ms"])
		if ms == 0 {
			ms = 5000
		}
		return recipemodel.Step{Kind: recipemodel.StepWaitForWindow, Text: args["title"], Ms: ms}, nil
	case "wait_for_element":
		ms, _ := strconv.Atoi(args["timeout_ms"])
		if ms == 0 {
			ms = 5000
		}
		return recipemodel.Step{Kind: recipemodel.StepWaitForElement, Label: args["label"], Ms: ms}, nil
	case "clear_field":
		return recipemodel.Step{Kind: recipemodel.StepClearField}, nil
	case "focus_via_palette":
		return recipemodel.Step{Kind: recipemodel.StepPalette, Shortcut: args["palette_shortcut"], Command: args["command_name"]}, nil
	case "focus_text_field":
		return recipemodel.Step{Kind: recipemodel.StepFocusField, Label: args["label"]}, nil
	case "run_recipe":
		// Composing recipes inside recipes would require nested execution;
		// out of scope for v1 of authoring.
		return recipemodel.Step{}, fmt.Errorf("nested recipes (run_recipe) cannot be saved inside a custom recipe yet")
	default:
		return recipemodel.Step{}, fmt.Errorf("step type %q is not supported in custom recipes", tc.ToolName)
	}
}

// normalizeName converts a free-form name into a snake_case identifier suitable
// as a recipe key. "Morning Routine" → "morning_routine".
func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = snakeRe.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		return ""
	}
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

func firstNonEmpty(parts ...string) string {
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			return p
		}
	}
	return ""
}
