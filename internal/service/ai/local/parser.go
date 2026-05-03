package local

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"

	// Import all intent packages to trigger their init() registration.
	// IMPORTANT: import order is the smart-intent matching order. List the most
	// specific patterns first so they match before more generic ones.
	_ "github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/settings"
	_ "github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/buttons"
	_ "github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/whatsapp"
	_ "github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/slack"
	_ "github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/discord"
	_ "github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/messages"
	_ "github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/finder"
	_ "github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/closeapp"
	_ "github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/system"
	_ "github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/terminal"
	_ "github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/vscode"
	_ "github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/browser"
	_ "github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/common"
)

// connectorRe splits text on "and"/"then"/"," and Arabic "ثم".
var connectorRe = regexp.MustCompile(`(?i)(?:\s*,\s*(?:and|then)?\s*|\s+and\s+|\s+then\s+|\s+ثم\s+)`)

// stepConnectorRe is a stricter test for explicit step chaining. Unlike
// connectorRe it does not match a bare comma, since most natural-language
// commands ("send a message, hi") contain commas without intending step
// separation.
var stepConnectorRe = regexp.MustCompile(`(?i)(?:\s+and\s+|\s+then\s+|\s+ثم\s+)`)

func hasStepConnector(text string) bool {
	return stepConnectorRe.MatchString(text)
}

// matchChainedSimple splits the text on step connectors and returns the
// concatenated simple-intent matches if and only if every non-empty segment
// matches a simple intent. Otherwise it returns nil so the caller can fall
// back to the standard pipeline.
func matchChainedSimple(text string) []aimodel.ToolCall {
	segments := stepConnectorRe.Split(text, -1)
	var calls []aimodel.ToolCall
	matchedSegments := 0
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
		matchedSegments++
	}
	if matchedSegments < 2 {
		return nil
	}
	return calls
}

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
		// Or a smart pattern (e.g. whatsapp send, settings search).
		isSmartKnown := matchSmart(text) != nil

		// These intent types are "escape" commands — the user wants to do
		// something different, not type in the terminal.
		isEscapeIntent := isSimpleKnown || isSmartKnown ||
			classified.Type == IntentAppFeature ||
			classified.Type == IntentOpenApp ||
			classified.Type == IntentFileBrowse ||
			classified.Type == IntentBrowserNav ||
			classified.Type == IntentMessaging ||
			classified.Type == IntentClickButton ||
			classified.Type == IntentSettingsSearch ||
			classified.Type == IntentSettingsPanel

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

	// Phase 0a: Explicit step chains. When the user separates steps with
	// "and" / "then" / "ثم" AND every segment is a known simple intent
	// (open / type / press / send / …), honor the literal chain instead of
	// letting the entity classifier or smart patterns collapse it into a
	// single recipe. Falls through unchanged when any segment isn't a clear
	// simple match, so multi-word recipes like "open Safari and go to X"
	// keep the existing browser_open_url path.
	if hasStepConnector(text) {
		if calls := matchChainedSimple(text); len(calls) > 0 {
			return calls
		}
	}

	// Phase 0: Smart regex patterns on full text (whatsapp, browser, settings, buttons).
	// These take priority because they're more specific than entity classification.
	if calls := matchSmart(text); len(calls) > 0 {
		return calls
	}

	// Phase 1+2+3: Smart classification on FULL text via entity extraction.
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

// matchSmart walks the smart intent registry. Returns nil if no pattern matches
// or if the matched pattern's Build returns nil.
func matchSmart(text string) []aimodel.ToolCall {
	for _, it := range intents.SmartIntents() {
		if m := it.Re.FindStringSubmatch(text); m != nil {
			if calls := it.Build(m); len(calls) > 0 {
				return calls
			}
		}
	}
	return nil
}
