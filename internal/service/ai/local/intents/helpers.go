package intents

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
)

// ToolCall builds an aimodel.ToolCall with a unique ID.
func ToolCall(name string, args map[string]string) aimodel.ToolCall {
	if args == nil {
		args = map[string]string{}
	}
	return aimodel.ToolCall{
		ID:        newID(),
		ToolName:  name,
		Arguments: args,
	}
}

// MustJSON marshals a map to a JSON string.
func MustJSON(m map[string]string) string {
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// StripQuotes removes a matching pair of surrounding quotes from s.
func StripQuotes(s string) string {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') ||
			(first == '\'' && last == '\'') ||
			(first == '`' && last == '`') {
			return s[1 : len(s)-1]
		}
		if strings.HasPrefix(s, "\u201c") && strings.HasSuffix(s, "\u201d") {
			return strings.TrimSuffix(strings.TrimPrefix(s, "\u201c"), "\u201d")
		}
	}
	return s
}

// IsBrowser returns true if the app name is a known browser.
func IsBrowser(app string) bool {
	switch app {
	case "Safari", "Google Chrome", "Firefox", "Brave Browser", "Arc":
		return true
	}
	return false
}

// newID returns a short random hex identifier for a tool call.
func newID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return "local_" + hex.EncodeToString(b[:])
}
