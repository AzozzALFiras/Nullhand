//go:build darwin

package mac

import (
	"errors"
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
