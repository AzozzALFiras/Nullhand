package command

import (
	"strings"
	"testing"
	"time"
)

func TestParseVolumeArg(t *testing.T) {
	cases := []struct {
		args   []string
		action volumeAction
		level  int
	}{
		{nil, volumeShow, 0},
		{[]string{"status"}, volumeShow, 0},
		{[]string{"40"}, volumeSet, 40},
		{[]string{"0"}, volumeSet, 0},
		{[]string{"+15"}, volumeStep, 15},
		{[]string{"-15"}, volumeStep, -15},
		{[]string{"up"}, volumeStep, volumeStepSize},
		{[]string{"DOWN"}, volumeStep, -volumeStepSize},
		{[]string{"mute"}, volumeMute, 0},
		{[]string{"unmute"}, volumeUnmute, 0},
	}
	for _, c := range cases {
		action, level, err := parseVolumeArg(c.args)
		if err != nil {
			t.Errorf("%q: unexpected error %v", c.args, err)
			continue
		}
		if action != c.action || level != c.level {
			t.Errorf("%q: got (%v, %d), want (%v, %d)", c.args, action, level, c.action, c.level)
		}
	}

	if _, _, err := parseVolumeArg([]string{"loud"}); err == nil {
		t.Error("a non-numeric level must be rejected so the user gets usage help")
	}
}

func TestParseAwakeArg(t *testing.T) {
	cases := []struct {
		args []string
		d    time.Duration
		stop bool
	}{
		{nil, 0, false},
		{[]string{"off"}, 0, true},
		{[]string{"stop"}, 0, true},
		{[]string{"90"}, 90 * time.Minute, false},
		{[]string{"90m"}, 90 * time.Minute, false},
		{[]string{"2h"}, 2 * time.Hour, false},
		{[]string{"on"}, 0, false},
	}
	for _, c := range cases {
		d, stop, err := parseAwakeArg(c.args)
		if err != nil {
			t.Errorf("%q: unexpected error %v", c.args, err)
			continue
		}
		if d != c.d || stop != c.stop {
			t.Errorf("%q: got (%s, %v), want (%s, %v)", c.args, d, stop, c.d, c.stop)
		}
	}

	for _, bad := range [][]string{{"soon"}, {"0"}, {"-5"}} {
		if _, _, err := parseAwakeArg(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

func TestSplitShortcutArgs(t *testing.T) {
	// Shortcut names routinely contain spaces, so everything before "--" is
	// the name and the rest is input.
	name, input := splitShortcutArgs([]string{"Airplane", "Mode", "Lock"})
	if name != "Airplane Mode Lock" || input != "" {
		t.Errorf("got name=%q input=%q", name, input)
	}
	name, input = splitShortcutArgs([]string{"Add", "Note", "--", "buy", "milk"})
	if name != "Add Note" || input != "buy milk" {
		t.Errorf("got name=%q input=%q", name, input)
	}
}

// Every macOS command must answer a bad invocation with usage help before it
// touches the platform, so the reply is the same on macOS and Linux.
func TestMacCommandsValidateBeforePlatform(t *testing.T) {
	vm := New()
	cases := []struct {
		name   string
		result Result
	}{
		{"/volume loud", vm.volume([]string{"loud"})},
		{"/say", vm.say(nil)},
		{"/notify", vm.notify(nil)},
		{"/shortcuts run", vm.shortcuts([]string{"run"})},
		{"/awake soon", vm.awake([]string{"soon"})},
		{"/dark bogus", vm.darkMode([]string{"bogus"})},
		{"/reveal", vm.reveal(nil)},
		{"/info", vm.fileInfo(nil)},
	}
	for _, c := range cases {
		if !strings.Contains(c.result.Text, "Usage:") {
			t.Errorf("%s should reply with usage help, got %q", c.name, c.result.Text)
		}
	}
}

func TestNotOnMacMentionsTheCommand(t *testing.T) {
	got := notOnMac("/volume").Text
	if !strings.Contains(got, "/volume") || !strings.Contains(got, "macOS") {
		t.Errorf("the reply should name the command and the platform, got %q", got)
	}
}
