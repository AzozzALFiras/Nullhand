//go:build darwin

package mac

import (
	"fmt"
	"os/exec"
	"strings"
)

// HealthLines describes the state of the macOS extras for /health: which
// helper binaries are present and whether Spotlight is indexing, since an
// unindexed volume is the usual reason /find falls back to a slow walk.
func HealthLines() []string {
	lines := []string{"", "macOS extras:"}
	for _, tool := range []struct{ bin, feature string }{
		{"shortcuts", "Shortcuts (/shortcuts)"},
		{"qlmanage", "Quick Look previews (/preview)"},
		{"mdfind", "Spotlight search (/find)"},
		{"caffeinate", "Keep awake (/awake)"},
		{"say", "Speech (/say)"},
	} {
		mark := "❌"
		if _, err := exec.LookPath(tool.bin); err == nil {
			mark = "✅"
		}
		lines = append(lines, fmt.Sprintf("  %s %s", mark, tool.feature))
	}
	lines = append(lines, "  "+spotlightIndexLine())
	return lines
}

// spotlightIndexLine reports whether the data volume — where the home folder
// actually lives — is indexed.
func spotlightIndexLine() string {
	out, err := run(defaultTimeout, "mdutil", "-s", "/System/Volumes/Data")
	switch {
	case err != nil:
		return "⚠️ Spotlight index: unknown (" + firstLine(err.Error()) + ")"
	case strings.Contains(out, "Indexing enabled"):
		return "✅ Spotlight index: enabled"
	default:
		return "⚠️ Spotlight index: disabled — /find falls back to a slower file walk"
	}
}
