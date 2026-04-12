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

// connectorRe splits a single utterance into sequential segments.
var connectorRe = regexp.MustCompile(`(?i)(?:\s*,\s*(?:and|then)?\s*|\s+and\s+|\s+then\s+|\s+ثم\s+)`)

// Parse turns a user utterance into a list of tool calls.
func Parse(text string) []aimodel.ToolCall {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// First try smart intents on the FULL text.
	for _, it := range intents.SmartIntents() {
		if m := it.Re.FindStringSubmatch(text); m != nil {
			return it.Build(m)
		}
	}

	// Fall back to splitting by connectors and matching simple intents.
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
