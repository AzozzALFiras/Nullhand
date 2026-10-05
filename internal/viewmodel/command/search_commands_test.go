package command

import (
	"strings"
	"testing"
)

func TestSplitFindArgs(t *testing.T) {
	query, folder := splitFindArgs([]string{"tax", "report.pdf"})
	if query != "tax report.pdf" || folder != "" {
		t.Errorf("got query=%q folder=%q", query, folder)
	}
	query, folder = splitFindArgs([]string{"report.pdf", "in", "~/Documents"})
	if query != "report.pdf" || folder != "~/Documents" {
		t.Errorf("got query=%q folder=%q", query, folder)
	}
	// A trailing "in" is part of the name, not a folder marker.
	query, folder = splitFindArgs([]string{"notes", "in"})
	if query != "notes in" || folder != "" {
		t.Errorf("got query=%q folder=%q", query, folder)
	}
}

// Both commands answer a bad invocation with usage help, on every platform.
func TestSearchCommandsValidateArguments(t *testing.T) {
	vm := New()
	for name, result := range map[string]Result{
		"/find":    vm.find(nil),
		"/preview": vm.preview(nil),
	} {
		if !strings.Contains(result.Text, "Usage:") {
			t.Errorf("%s should reply with usage help, got %q", name, result.Text)
		}
	}
}
