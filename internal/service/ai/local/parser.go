package local

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
	"github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents"

	// Import all intent packages to trigger their init() registration.
	_ "github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents/browser"
	_ "github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents/common"
	_ "github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents/discord"
	_ "github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents/finder"
	_ "github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents/messages"
	_ "github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents/slack"
	_ "github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents/system"
	_ "github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents/terminal"
	_ "github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents/vscode"
	_ "github.com/AzozzALFiras/nullhand/internal/service/ai/local/intents/whatsapp"
)

// connectorRe splits text on "and"/"then"/"," and Arabic "ثم".
var connectorRe = regexp.MustCompile(`(?i)(?:\s*,\s*(?:and|then)?\s*|\s+and\s+|\s+then\s+|\s+ثم\s+)`)

// Parse turns user text into tool calls using the 3-phase pipeline:
// 1. Extract entities (apps, paths, actions, modifiers)
// 2. Classify intent (what does the user want?)
// 3. Build tool calls (convert intent to executable actions)
//
// Falls back to simple regex intents for basic commands (screenshot, type, etc.)
func Parse(text string) []aimodel.ToolCall {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// Phase 1+2+3: Smart classification on FULL text
	entities := Extract(text)
	classified := Classify(entities)

	if classified.Type != IntentSimple {
		calls := BuildToolCalls(classified)
		if len(calls) > 0 {
			return calls
		}
	}

	// Fallback: split by connectors and match simple intents (screenshot, type, press, etc.)
	segments := connectorRe.Split(text, -1)
	var calls []aimodel.ToolCall
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		matched := matchSimple(seg)
		if len(matched) == 0 {
			return nil
		}
		calls = append(calls, matched...)
	}
	return calls
}

// matchSimple walks the simple intent registry.
func matchSimple(segment string) []aimodel.ToolCall {
	for _, it := range intents.SimpleIntents() {
		if m := it.Re.FindStringSubmatch(segment); m != nil {
			return it.Build(m)
		}
	}
	return nil
}
