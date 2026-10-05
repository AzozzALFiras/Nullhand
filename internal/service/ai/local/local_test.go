package local

import "testing"

func TestIsUserFacing(t *testing.T) {
	for _, surfaced := range []string{"❌ failed", "⚠️ not run", "ℹ️ 🔋 95%"} {
		if !isUserFacing(surfaced) {
			t.Errorf("%q must become the bot's reply", surfaced)
		}
	}
	for _, quiet := range []string{"clicked at 10,20", "opened Safari", "Done.", ""} {
		if isUserFacing(quiet) {
			t.Errorf("%q is internal progress, not a reply", quiet)
		}
	}
}
