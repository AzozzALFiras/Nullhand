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

	// Newer step kinds — verify state and click flexibly.
	StepWaitForWindow  StepKind = "wait_for_window"  // wait until window title contains Text
	StepWaitForText    StepKind = "wait_for_text"    // wait until OCR sees Text
	StepWaitForElement StepKind = "wait_for_element" // wait until AT-SPI element with Label exists
	StepClickText      StepKind = "click_text"       // OCR-based: locate Text and click its center
	StepClickFuzzy     StepKind = "click_fuzzy"      // AT-SPI fuzzy click by Label, OCR fallback
	StepClearField     StepKind = "clear_field"      // Ctrl+A + Delete
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

	// sleep_ms / timeout for wait_* steps (in milliseconds)
	Ms int `json:"ms,omitempty"`

	// focus_field / wait_for_element / click_fuzzy
	Label string `json:"label,omitempty"`

	// wait_for_text uses Text (substring to match in OCR)
	// wait_for_window uses Text (substring to match in window title)
	// click_text uses Text (substring to locate via OCR)
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
