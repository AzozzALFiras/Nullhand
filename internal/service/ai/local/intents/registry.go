// Package intents provides the intent registration system.
// Each app package registers its intents via Register() in init().
package intents

import (
	"regexp"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
)

// Intent is a pattern that matches a user utterance and produces tool calls.
type Intent struct {
	Re    *regexp.Regexp
	Build func(matches []string) []aimodel.ToolCall
}

// smartRegistry holds multi-step/complex intents (matched on full text first).
var smartRegistry []Intent

// simpleRegistry holds single-step intents (matched per segment after splitting).
var simpleRegistry []Intent

// RegisterSmart adds complex intents that match on the full text.
func RegisterSmart(intents ...Intent) {
	smartRegistry = append(smartRegistry, intents...)
}

// RegisterSimple adds simple intents that match on individual segments.
func RegisterSimple(intents ...Intent) {
	simpleRegistry = append(simpleRegistry, intents...)
}

// SmartIntents returns all registered smart intents.
func SmartIntents() []Intent {
	return smartRegistry
}

// SimpleIntents returns all registered simple intents.
func SimpleIntents() []Intent {
	return simpleRegistry
}
