package recipe

// StepKind identifies the action a recipe step performs.
type StepKind string

const (
	StepOpenApp    StepKind = "open_app"
	StepPressKey   StepKind = "press_key"
	StepTypeText   StepKind = "type_text"
	StepPalette    StepKind = "palette"
	StepSleepMs    StepKind = "sleep_ms"
	StepFocusField StepKind = "focus_field"
)

// Step is a single action in a recipe. Fields are populated based on Kind.
// Parameter placeholders of the form {{name}} are substituted at run time.
type Step struct {
	Kind StepKind `json:"kind"`

	// open_app
	AppName string `json:"app_name,omitempty"`

	// press_key
	Key string `json:"key,omitempty"`

	// type_text
	Text string `json:"text,omitempty"`

	// palette
	Shortcut string `json:"shortcut,omitempty"`
	Command  string `json:"command,omitempty"`

	// sleep_ms
	Ms int `json:"ms,omitempty"`

	// focus_field
	Label string `json:"label,omitempty"`
}

// Recipe is a named sequence of steps for performing a reusable task.
type Recipe struct {
	Name        string   `json:"-"`
	Description string   `json:"description,omitempty"`
	Parameters  []string `json:"parameters,omitempty"`
	Steps       []Step   `json:"steps"`
}

// File is the on-disk format for ~/.nullhand/recipes.json.
type File struct {
	Version int               `json:"version"`
	Recipes map[string]Recipe `json:"recipes"`
}
