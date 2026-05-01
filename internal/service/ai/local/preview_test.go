package local

import (
	"strings"
	"testing"

	recipemodel "github.com/AzozzALFiras/Nullhand/internal/model/recipe"
)

// fakeRecipes is a tiny RecipeProvider for tests.
type fakeRecipes map[string]recipemodel.Recipe

func (f fakeRecipes) Get(name string) (recipemodel.Recipe, bool) {
	r, ok := f[name]
	return r, ok
}

func TestIsPreviewRequest(t *testing.T) {
	cases := []struct {
		input     string
		wantOK    bool
		wantInner string
	}{
		{"preview: open Firefox", true, "open Firefox"},
		{"dry-run: take a screenshot", true, "take a screenshot"},
		{"dry run: open Firefox", true, "open Firefox"},
		{"simulate: open Firefox", true, "open Firefox"},
		{"معاينة: افتح Firefox", true, "افتح Firefox"},
		{"جرب: افتح Firefox", true, "افتح Firefox"},
		{"جرّب: افتح Firefox", true, "افتح Firefox"},
		{"open Firefox", false, ""},
		{"preview only — no colon", false, ""},
	}
	for _, tc := range cases {
		gotInner, gotOK := IsPreviewRequest(tc.input)
		if gotOK != tc.wantOK {
			t.Errorf("IsPreviewRequest(%q) ok = %v, want %v", tc.input, gotOK, tc.wantOK)
		}
		if gotOK && gotInner != tc.wantInner {
			t.Errorf("IsPreviewRequest(%q) inner = %q, want %q", tc.input, gotInner, tc.wantInner)
		}
	}
}

func TestPreviewExpandsRecipeSteps(t *testing.T) {
	recipes := fakeRecipes{
		"browser_open_url": {
			Name:        "browser_open_url",
			Description: "test",
			Parameters:  []string{"browser", "url"},
			Steps: []recipemodel.Step{
				{Kind: recipemodel.StepOpenApp, AppName: "{{browser}}"},
				{Kind: recipemodel.StepPressKey, Key: "cmd+l"},
				{Kind: recipemodel.StepTypeText, Text: "{{url}}"},
				{Kind: recipemodel.StepPressKey, Key: "return"},
			},
		},
	}

	plan := Preview("open firefox and go to github.com", recipes)
	if !strings.Contains(plan, "Run recipe \"browser_open_url\"") {
		t.Errorf("plan should mention browser_open_url; got:\n%s", plan)
	}
	if !strings.Contains(plan, "open_app(\"Firefox\")") {
		t.Errorf("plan should expand placeholder to Firefox; got:\n%s", plan)
	}
	if !strings.Contains(plan, "type_text(\"github.com\")") {
		t.Errorf("plan should expand url placeholder; got:\n%s", plan)
	}
}

func TestPreviewUnrecognized(t *testing.T) {
	plan := Preview("xyzzy nonsense", nil)
	if !strings.Contains(plan, "could not understand") {
		t.Errorf("expected an error message in the plan, got:\n%s", plan)
	}
}
