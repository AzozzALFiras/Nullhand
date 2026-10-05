//go:build darwin

package mac

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These run against the real macOS APIs, so they stay read-only: nothing here
// locks the screen, toggles the appearance, changes the volume level, or runs
// one of the user's Shortcuts.

func TestVolumeReadsAndSetsWithoutChangingLevel(t *testing.T) {
	before, err := Volume()
	if err != nil {
		t.Skipf("no software-controlled audio output on this machine: %v", err)
	}
	if before.Output < 0 || before.Output > 100 {
		t.Fatalf("volume out of range: %+v", before)
	}

	// Setting the current level exercises the write path without the user
	// hearing a difference.
	after, err := SetVolume(before.Output)
	if err != nil {
		t.Fatalf("SetVolume: %v", err)
	}
	if after.Output != before.Output {
		t.Errorf("volume changed: before %d, after %d", before.Output, after.Output)
	}
}

func TestDarkModeReads(t *testing.T) {
	if _, err := DarkMode(); err != nil {
		t.Fatalf("reading the appearance requires Automation permission for System Events: %v", err)
	}
}

func TestBatteryStatusReads(t *testing.T) {
	b, err := BatteryStatus()
	if err != nil {
		t.Fatalf("BatteryStatus: %v", err)
	}
	if b.Present && (b.Percent < 1 || b.Percent > 100) {
		t.Errorf("implausible battery level: %+v", b)
	}
	if b.Summary() == "" {
		t.Error("Summary must always render something for /battery")
	}
}

func TestListShortcuts(t *testing.T) {
	names, err := ListShortcuts()
	if err != nil {
		t.Fatalf("ListShortcuts: %v", err)
	}
	for _, n := range names {
		if strings.TrimSpace(n) == "" {
			t.Error("blank shortcut name should have been filtered out")
		}
	}
}

func TestRunShortcutRequiresName(t *testing.T) {
	if _, err := RunShortcut("  ", ""); err == nil {
		t.Error("an empty name must fail before the CLI is invoked")
	}
}

// Spotlight has nothing indexed for many developer folders, so this doubles as
// a test of the `find` fallback: the file exists and must be found either way.
func TestFindLocatesFileInRepo(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))

	hits, err := Find("mac_other.go", repoRoot, 10)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	found := false
	for _, h := range hits {
		if strings.HasSuffix(h, "mac_other.go") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find mac_other.go under %s, got %q", repoRoot, hits)
	}
}

func TestFindRejectsEmptyQuery(t *testing.T) {
	if _, err := Find("", "", 10); err == nil {
		t.Error("an empty query must return an error instead of listing everything")
	}
}

func TestFindRespectsLimit(t *testing.T) {
	wd, _ := os.Getwd()
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))
	hits, err := Find(".go", repoRoot, 3)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(hits) > 3 {
		t.Errorf("limit ignored: got %d hits", len(hits))
	}
}

func TestPreviewProducesPNG(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(path, []byte("hello from nullhand\n"), 0600); err != nil {
		t.Fatal(err)
	}

	data, err := Preview(path)
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		t.Errorf("expected PNG bytes, got %d bytes starting with %q", len(data), data[:min(8, len(data))])
	}
}

func TestPreviewRejectsFolderAndMissingFile(t *testing.T) {
	if _, err := Preview(t.TempDir()); err == nil {
		t.Error("a folder has no Quick Look thumbnail to send")
	}
	if _, err := Preview(filepath.Join(t.TempDir(), "nope.pdf")); err == nil {
		t.Error("a missing file must be reported before qlmanage runs")
	}
}

func TestKeepAwakeStartsAndStops(t *testing.T) {
	if _, ok := KeepAwakeUntil(); ok {
		t.Skip("a keep-awake is already running; leaving it alone")
	}

	until, err := KeepAwake(time.Minute)
	if err != nil {
		t.Fatalf("KeepAwake: %v", err)
	}
	if time.Until(until) > time.Minute+time.Second || time.Until(until) < 50*time.Second {
		t.Errorf("deadline %s is not about a minute away", until)
	}
	if _, ok := KeepAwakeUntil(); !ok {
		t.Error("KeepAwakeUntil should report the running keep-awake")
	}

	if !StopKeepAwake() {
		t.Error("StopKeepAwake should report that it stopped one")
	}
	if _, ok := KeepAwakeUntil(); ok {
		t.Error("nothing should be holding the Mac awake after StopKeepAwake")
	}
	if StopKeepAwake() {
		t.Error("a second StopKeepAwake has nothing to stop")
	}
}

func TestCurrentTrackWithoutMediaApp(t *testing.T) {
	if appRunning("Music") || appRunning("Spotify") {
		t.Skip("a media app is running; skipping the no-app path")
	}
	if _, err := CurrentTrack(); !errors.Is(err, ErrNoMediaApp) {
		t.Errorf("expected ErrNoMediaApp, got %v", err)
	}
	if _, err := Media("next"); !errors.Is(err, ErrNoMediaApp) {
		t.Errorf("transport commands need a running app, got %v", err)
	}
}

func TestMediaRejectsUnknownActionBeforeTouchingApps(t *testing.T) {
	if _, err := Media("eject"); err == nil || errors.Is(err, ErrNoMediaApp) {
		t.Errorf("an unknown action must be rejected on its own merits, got %v", err)
	}
}

func TestHealthLinesDescribeTooling(t *testing.T) {
	lines := strings.Join(HealthLines(), "\n")
	for _, want := range []string{"macOS extras", "Shortcuts", "Quick Look", "Spotlight index"} {
		if !strings.Contains(lines, want) {
			t.Errorf("health output missing %q:\n%s", want, lines)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
