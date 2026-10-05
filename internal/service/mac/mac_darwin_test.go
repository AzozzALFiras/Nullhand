//go:build darwin

package mac

import (
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

func TestInfoDescribesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(path, []byte("hello nullhand"), 0600); err != nil {
		t.Fatal(err)
	}

	info, err := Info(path)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if info.Name != "note.txt" || info.IsDir {
		t.Errorf("got %+v", info)
	}
	if info.Size != 14 {
		t.Errorf("size = %d, want 14", info.Size)
	}
	if info.Modified.IsZero() || info.Created.IsZero() {
		t.Errorf("both timestamps should be filled on macOS: %+v", info)
	}
	if info.Summary() == "" {
		t.Error("Summary must always render something for /info")
	}
}

func TestInfoDescribesFolder(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("12345"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	info, err := Info(dir)
	if err != nil {
		t.Fatalf("Info: %v", err)
	}
	if !info.IsDir {
		t.Fatal("expected a folder")
	}
	if info.Items != 3 {
		t.Errorf("items = %d, want 3", info.Items)
	}
	// du reports allocated blocks, so the total is at least the content size.
	if info.Size <= 0 {
		t.Errorf("folder size should be totalled, got %d", info.Size)
	}
}

func TestInfoRejectsMissingFile(t *testing.T) {
	if _, err := Info(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("a missing path must return an error")
	}
}

func TestTrashStatusReads(t *testing.T) {
	state, err := TrashStatus()
	if err != nil {
		if strings.Contains(err.Error(), "not authorized") || strings.Contains(err.Error(), "-1743") {
			t.Skipf("Automation permission for Finder not granted: %v", err)
		}
		t.Fatalf("TrashStatus: %v", err)
	}
	if state.Items < 0 || state.Size < 0 {
		t.Errorf("implausible trash state %+v", state)
	}
	if state.Summary() == "" {
		t.Error("Summary must always render something for /trash")
	}
}

// Reveal and MoveToTrash are not exercised against real files here: one opens
// a Finder window on the user's screen, the other puts things in their Trash.
func TestFinderActionsValidatePaths(t *testing.T) {
	if err := Reveal(""); err == nil {
		t.Error("an empty path must be refused before Finder is asked")
	}
	if err := Reveal(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("a missing path must be refused")
	}
	if err := MoveToTrash(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Error("a missing path must be refused before Finder is asked")
	}
}
