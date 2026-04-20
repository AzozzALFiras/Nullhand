package recipe

import (
	"fmt"
	"sort"
	"strings"
	"time"

	recipemodel "github.com/AzozzALFiras/Nullhand/internal/model/recipe"
	appsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/apps"
	kbsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/keyboard"
	palettesvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/palette"
)

// Service runs recipes by dispatching their steps to the underlying macOS
// services. It holds the merged recipe set (defaults + user overrides) in
// memory and never touches the filesystem itself — loading is the job of
// the recipe repository.
type Service struct {
	recipes map[string]recipemodel.Recipe
}

// New creates a recipe service with the given set of recipes (already merged).
func New(recipes map[string]recipemodel.Recipe) *Service {
	return &Service{recipes: recipes}
}

// List returns all recipes sorted by name.
func (s *Service) List() []recipemodel.Recipe {
	out := make([]recipemodel.Recipe, 0, len(s.recipes))
	for _, r := range s.recipes {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Names returns the recipe names sorted alphabetically.
func (s *Service) Names() []string {
	names := make([]string, 0, len(s.recipes))
	for name := range s.recipes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Get returns a single recipe by name.
func (s *Service) Get(name string) (recipemodel.Recipe, bool) {
	r, ok := s.recipes[name]
	return r, ok
}

// Run executes the named recipe. If dryRun is true, no steps are actually
// performed — only a human-readable plan is returned. params are substituted
// into step fields as {{name}} placeholders before execution.
func (s *Service) Run(name string, params map[string]string, dryRun bool) (string, error) {
	r, ok := s.recipes[name]
	if !ok {
		return "", fmt.Errorf("recipe %q not found", name)
	}

	var log []string
	for i, raw := range r.Steps {
		step := resolveStep(raw, params)
		log = append(log, fmt.Sprintf("[%d] %s", i+1, describeStep(step)))

		if dryRun {
			continue
		}

		if err := executeStep(step); err != nil {
			return strings.Join(log, "\n"), fmt.Errorf("step %d (%s): %w", i+1, step.Kind, err)
		}
	}
	return strings.Join(log, "\n"), nil
}

// resolveStep substitutes {{param}} placeholders in all string fields.
func resolveStep(step recipemodel.Step, params map[string]string) recipemodel.Step {
	step.AppName = subst(step.AppName, params)
	step.Key = subst(step.Key, params)
	step.Text = subst(step.Text, params)
	step.Shortcut = subst(step.Shortcut, params)
	step.Command = subst(step.Command, params)
	step.Label = subst(step.Label, params)
	return step
}

func subst(s string, params map[string]string) string {
	if s == "" || len(params) == 0 {
		return s
	}
	for k, v := range params {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}

func describeStep(step recipemodel.Step) string {
	switch step.Kind {
	case recipemodel.StepOpenApp:
		return fmt.Sprintf("open_app(%q)", step.AppName)
	case recipemodel.StepPressKey:
		return fmt.Sprintf("press_key(%q)", step.Key)
	case recipemodel.StepTypeText:
		return fmt.Sprintf("type_text(%q)", truncate(step.Text, 40))
	case recipemodel.StepPalette:
		return fmt.Sprintf("palette(%q, %q)", step.Shortcut, step.Command)
	case recipemodel.StepSleepMs:
		return fmt.Sprintf("sleep(%dms)", step.Ms)
	case recipemodel.StepFocusField:
		return fmt.Sprintf("focus_field(label=%q)", step.Label)
	default:
		return string(step.Kind)
	}
}

func executeStep(step recipemodel.Step) error {
	switch step.Kind {
	case recipemodel.StepOpenApp:
		return appsvc.Open(step.AppName)
	case recipemodel.StepPressKey:
		return kbsvc.PressKey(step.Key)
	case recipemodel.StepTypeText:
		return kbsvc.Type(step.Text)
	case recipemodel.StepPalette:
		return palettesvc.Run(step.Shortcut, step.Command)
	case recipemodel.StepSleepMs:
		time.Sleep(time.Duration(step.Ms) * time.Millisecond)
		return nil
	default:
		return fmt.Errorf("unknown step kind %q", step.Kind)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
