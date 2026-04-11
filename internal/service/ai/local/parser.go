package local

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
)

// connectorRe splits a single utterance into sequential segments.
// It matches English "and" / "then" / "," and Arabic "ثم".
// Note: Go's RE2 `\b` is ASCII-only, so we require explicit whitespace around
// the word connectors to avoid matching substrings inside Arabic words.
var connectorRe = regexp.MustCompile(`(?i)(?:\s*,\s*(?:and|then)?\s*|\s+and\s+|\s+then\s+|\s+ثم\s+)`)

// Parse turns a user utterance into a list of tool calls. Returns nil when
// no known intent matches any segment.
func Parse(text string) []aimodel.ToolCall {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	segments := splitConnectors(text)

	var calls []aimodel.ToolCall
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}
		matched := matchSegment(seg)
		if len(matched) == 0 {
			// Any unknown segment fails the whole parse so the caller can
			// reply with the "not understood" fallback.
			return nil
		}
		calls = append(calls, matched...)
	}
	return calls
}

// splitConnectors breaks text on "and"/"then"/"," and Arabic equivalents.
func splitConnectors(text string) []string {
	return connectorRe.Split(text, -1)
}

// matchSegment walks the intent table and returns the first match.
func matchSegment(segment string) []aimodel.ToolCall {
	for _, it := range intents {
		if m := it.re.FindStringSubmatch(segment); m != nil {
			return it.build(m)
		}
	}
	return nil
}

// toolCall builds an aimodel.ToolCall with a unique ID.
func toolCall(name string, args map[string]string) aimodel.ToolCall {
	if args == nil {
		args = map[string]string{}
	}
	return aimodel.ToolCall{
		ID:        newID(),
		ToolName:  name,
		Arguments: args,
	}
}

// newID returns a short random hex identifier for a tool call.
func newID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return "local_" + hex.EncodeToString(b[:])
}
