package local

import (
	"regexp"
	"strings"

	aimodel "github.com/iamakillah/Nullhand_Linux/internal/model/ai"
	"github.com/iamakillah/Nullhand_Linux/internal/service/ai/local/intents"

	// Import all intent packages to trigger their init() registration.
	_ "github.com/iamakillah/Nullhand_Linux/internal/service/ai/local/intents/browser"
	_ "github.com/iamakillah/Nullhand_Linux/internal/service/ai/local/intents/common"
	_ "github.com/iamakillah/Nullhand_Linux/internal/service/ai/local/intents/discord"
	_ "github.com/iamakillah/Nullhand_Linux/internal/service/ai/local/intents/finder"
	_ "github.com/iamakillah/Nullhand_Linux/internal/service/ai/local/intents/messages"
	_ "github.com/iamakillah/Nullhand_Linux/internal/service/ai/local/intents/slack"
	_ "github.com/iamakillah/Nullhand_Linux/internal/service/ai/local/intents/system"
	_ "github.com/iamakillah/Nullhand_Linux/internal/service/ai/local/intents/terminal"
	_ "github.com/iamakillah/Nullhand_Linux/internal/service/ai/local/intents/vscode"
	_ "github.com/iamakillah/Nullhand_Linux/internal/service/ai/local/intents/whatsapp"
)

// connectorRe splits text on "and"/"then"/"," and Arabic "ثم".
var connectorRe = regexp.MustCompile(`(?i)(?:\s*,\s*(?:and|then)?\s*|\s+and\s+|\s+then\s+|\s+ثم\s+)`)

// Parse turns user text into tool calls using the 3-phase pipeline.
// No session context — each message is independent.
func Parse(text string) []aimodel.ToolCall {
	return ParseWithContext(text, nil)
}

// ParseWithContext turns user text into tool calls with optional session context.
// If the smart classifier and simple intents both fail, the context is used
// as a fallback (e.g. "ls" in terminal mode → type ls + enter).
func ParseWithContext(text string, ctx *SessionContext) []aimodel.ToolCall {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// If we're in terminal/claude mode, check context FIRST.
	// This prevents "cd docs" from being interpreted as "browse documents"
	// when the user is clearly typing in a terminal.
	if ctx != nil && (ctx.ActiveMode == "terminal" || ctx.ActiveMode == "claude") {
		// Only escape to normal parsing if it's a clearly recognized command
		// (screenshot, open app, browse, etc.) — otherwise treat as terminal input.
		entities := Extract(text)
		classified := ClassifyWithContext(entities, ctx)

		// Check if it's a known simple command (screenshot, send, help, etc.)
		isSimpleKnown := matchSimple(text) != nil

		// These intent types are "escape" commands — the user wants to do
		// something different, not type in the terminal.
		isEscapeIntent := isSimpleKnown ||
			classified.Type == IntentAppFeature ||
			classified.Type == IntentOpenApp ||
			classified.Type == IntentFileBrowse ||
			classified.Type == IntentBrowserNav ||
			classified.Type == IntentMessaging

		if !isEscapeIntent {
			// Not an escape command → type it in the active terminal/chat
			return ApplyContext(text, ctx)
		}

		// It's an escape command → continue with normal classification
		if classified.Type != IntentSimple {
			calls := BuildToolCalls(classified)
			if len(calls) > 0 {
				return calls
			}
		}
	}

	// Phase 1+2+3: Smart classification on FULL text
	entities := Extract(text)
	classified := ClassifyWithContext(entities, ctx)

	if classified.Type != IntentSimple {
		calls := BuildToolCalls(classified)
		if len(calls) > 0 {
			return calls
		}
	}

	// Fallback: split by connectors and match simple intents
	segments := connectorRe.Split(text, -1)
	var calls []aimodel.ToolCall
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		matched := matchSimple(seg)
		if len(matched) == 0 {
			// No simple match either — try session context fallback
			if ctx != nil {
				contextCalls := ApplyContext(text, ctx)
				if len(contextCalls) > 0 {
					return contextCalls
				}
			}
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
