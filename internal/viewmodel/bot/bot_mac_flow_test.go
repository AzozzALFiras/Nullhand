//go:build darwin

package bot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// End-to-end macOS checks through the real handleUpdate pipeline. Read-only:
// nothing here locks the screen, changes the volume level or runs a Shortcut.

// waitForReply polls until a message containing want shows up.
func waitForReply(t *testing.T, fake *fakeTelegram, want string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		for _, msg := range fake.texts() {
			if strings.Contains(msg, want) {
				return msg
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("no reply containing %q within %s; got %q", want, timeout, fake.texts())
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestMacFlowBatteryCommand(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	vm.handleUpdate(textUpdate("/battery"))

	msg := fake.lastText(t)
	if !strings.Contains(msg, "%") && !strings.Contains(msg, "No battery") {
		t.Errorf("/battery should report a charge level, got %q", msg)
	}
}

func TestMacFlowVolumeCommandShowsLevel(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	vm.handleUpdate(textUpdate("/volume"))

	msg := fake.lastText(t)
	if !strings.Contains(msg, "Volume") && !strings.Contains(msg, "Muted") {
		t.Errorf("/volume with no argument should report the current level, got %q", msg)
	}
}

func TestMacFlowFindCommand(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	wd, _ := os.Getwd()
	repoRoot := filepath.Clean(filepath.Join(wd, "..", "..", ".."))

	vm.handleUpdate(textUpdate("/find mac_other.go in " + repoRoot))

	msg := fake.lastText(t)
	if !strings.Contains(msg, "mac_other.go") {
		t.Errorf("/find should list the matching path, got %q", msg)
	}
}

func TestMacFlowPreviewSendsPhoto(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	path := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(path, []byte("preview me"), 0600); err != nil {
		t.Fatal(err)
	}

	vm.handleUpdate(textUpdate("/preview " + path))

	fake.mu.Lock()
	defer fake.mu.Unlock()
	for _, c := range fake.calls {
		if c.method == "sendPhoto" {
			return
		}
	}
	t.Errorf("/preview should deliver a Quick Look image, calls were %+v", fake.calls)
}

func TestMacFlowHelpListsMacCommands(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	vm.handleUpdate(textUpdate("/help"))

	if msg := fake.lastText(t); !strings.Contains(msg, "macOS extras") || !strings.Contains(msg, "/shortcuts") {
		t.Errorf("/help should advertise the macOS commands on a Mac, got %q", msg)
	}
}

// The offline parser path: plain "battery" has to reach the mac_power tool and
// the answer has to make it back to the chat rather than a bare "Done.".
func TestMacFlowNaturalLanguageBattery(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	vm.handleUpdate(textUpdate("battery"))

	msg := waitForReply(t, fake, "%", 15*time.Second)
	if strings.Contains(msg, "Done.") {
		t.Errorf("the battery level must be surfaced, got %q", msg)
	}
}

func TestMacFlowNaturalLanguageArabicBattery(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	vm.handleUpdate(textUpdate("كم البطارية"))
	waitForReply(t, fake, "%", 15*time.Second)
}

// Telegram's command menu should advertise the macOS extras on a Mac.
func TestMacMenuIsAdvertised(t *testing.T) {
	var names []string
	for _, c := range defaultMenu() {
		names = append(names, c.Command)
	}
	joined := strings.Join(names, " ")
	for _, want := range []string{"shortcuts", "volume", "media", "find", "battery"} {
		if !strings.Contains(joined, want) {
			t.Errorf("command menu is missing %q: %v", want, names)
		}
	}
}
