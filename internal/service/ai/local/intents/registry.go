// Package intents provides the intent registration system.
// Each app package registers its intents via Register() in init().
package intents

import (
	"regexp"
	"sort"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
)

// Priority levels for smart intents. Higher numbers match first when multiple
// patterns could match the same text. Use these to disambiguate overlapping
// patterns across packages (e.g. settings vs. generic browser search).
const (
	PriorityVeryHigh = 100 // exact-context matches: "ابحث في الإعدادات", "اضغط زر X"
	PriorityHigh     = 80  // app-specific: messaging, IDE features
	PriorityNormal   = 50  // default for most patterns
	PriorityLow      = 20  // generic catch-alls (browse folder, generic search)
)

// Intent is a pattern that matches a user utterance and produces tool calls.
// Priority defaults to PriorityNormal when zero. Higher priority intents are
// tried first by matchSmart.
type Intent struct {
	Re       *regexp.Regexp
	Build    func(matches []string) []aimodel.ToolCall
	Priority int
}

// smartRegistry holds multi-step/complex intents (matched on full text first).
var (
	smartRegistry  []Intent
	smartSorted    []Intent
	smartSortDirty bool
)

// simpleRegistry holds single-step intents (matched per segment after splitting).
var simpleRegistry []Intent

// RegisterSmart adds complex intents that match on the full text.
func RegisterSmart(intents ...Intent) {
	smartRegistry = append(smartRegistry, intents...)
	smartSortDirty = true
}

// RegisterSimple adds simple intents that match on individual segments.
func RegisterSimple(intents ...Intent) {
	simpleRegistry = append(simpleRegistry, intents...)
}

// SmartIntents returns all registered smart intents sorted by priority desc.
// Intents with the same priority preserve their registration order.
func SmartIntents() []Intent {
	if smartSortDirty || smartSorted == nil {
		out := make([]Intent, len(smartRegistry))
		copy(out, smartRegistry)
		// Stable sort by priority desc; default priority is PriorityNormal.
		sort.SliceStable(out, func(i, j int) bool {
			pi := out[i].Priority
			if pi == 0 {
				pi = PriorityNormal
			}
			pj := out[j].Priority
			if pj == 0 {
				pj = PriorityNormal
			}
			return pi > pj
		})
		smartSorted = out
		smartSortDirty = false
	}
	return smartSorted
}

// SimpleIntents returns all registered simple intents.
func SimpleIntents() []Intent {
	return simpleRegistry
}
