package recipe

import (
	"fmt"
	"sort"
	"strings"
	"time"

	recipemodel "github.com/AzozzALFiras/Nullhand/internal/model/recipe"
	a11ysvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/accessibility"
	appsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/apps"
	kbsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/keyboard"
	mousesvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/mouse"
	ocrsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/ocr"
	palettesvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/palette"
	screensvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/screen"
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

// Set adds or replaces a recipe in the in-memory registry. Use this for
// user-defined recipes saved at runtime (e.g. via "save this as routine X").
// Persistence to disk is the caller's responsibility.
func (s *Service) Set(name string, r recipemodel.Recipe) {
	r.Name = name
	s.recipes[name] = r
}

// All returns a copy of the in-memory recipe map (suitable for persisting).
func (s *Service) All() map[string]recipemodel.Recipe {
	out := make(map[string]recipemodel.Recipe, len(s.recipes))
	for name, r := range s.recipes {
		out[name] = r
	}
	return out
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
	case recipemodel.StepWaitForWindow:
		return fmt.Sprintf("wait_for_window(%q, %dms)", step.Text, step.Ms)
	case recipemodel.StepWaitForText:
		return fmt.Sprintf("wait_for_text(%q, %dms)", truncate(step.Text, 40), step.Ms)
	case recipemodel.StepWaitForElement:
		return fmt.Sprintf("wait_for_element(label=%q, %dms)", step.Label, step.Ms)
	case recipemodel.StepClickText:
		return fmt.Sprintf("click_text(%q)", truncate(step.Text, 40))
	case recipemodel.StepClickFuzzy:
		return fmt.Sprintf("click_fuzzy(label=%q)", step.Label)
	case recipemodel.StepClearField:
		return "clear_field()"
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
	case recipemodel.StepFocusField:
		return a11ysvc.FocusField("", step.Label)
	case recipemodel.StepWaitForWindow:
		_, err := screensvc.WaitForWindow(step.Text, step.Ms)
		return err
	case recipemodel.StepWaitForText:
		_, err := ocrsvc.WaitForText(step.Text, step.Ms)
		return err
	case recipemodel.StepWaitForElement:
		return a11ysvc.WaitForElement("", step.Label, step.Ms)
	case recipemodel.StepClickText:
		// OCR-based click: locate text on screen, click its center.
		box, found, err := ocrsvc.LocateText(step.Text)
		if err != nil {
			return fmt.Errorf("click_text(%q): %w", step.Text, err)
		}
		if !found {
			return fmt.Errorf("click_text(%q): text not found on screen", step.Text)
		}
		return mousesvc.Click(box.CenterX, box.CenterY)
	case recipemodel.StepClickFuzzy:
		// AT-SPI fuzzy first; OCR fallback if AT-SPI fails (e.g. Electron app).
		if err := a11ysvc.ClickFuzzy("", step.Label); err == nil {
			return nil
		}
		box, found, err := ocrsvc.LocateText(step.Label)
		if err != nil {
			return fmt.Errorf("click_fuzzy(%q): AT-SPI failed and OCR error: %w", step.Label, err)
		}
		if !found {
			return fmt.Errorf("click_fuzzy(%q): not found via AT-SPI or OCR", step.Label)
		}
		return mousesvc.Click(box.CenterX, box.CenterY)
	case recipemodel.StepClearField:
		return kbsvc.ClearField()
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
