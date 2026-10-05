package command

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	macsvc "github.com/AzozzALFiras/Nullhand/internal/service/mac"
	tgfmt "github.com/AzozzALFiras/Nullhand/internal/view/telegram"
)

// The macOS-only commands. They stay registered on every platform so a Linux
// user gets an explanation instead of "Unknown command", and each handler
// validates its arguments before asking the platform — that way a usage reply
// is identical on both.

// notOnMac explains that a command needs macOS.
func notOnMac(cmd string) Result {
	return Result{Text: fmt.Sprintf("📱 <b>%s</b> works on macOS only.", tgfmt.Escape(cmd))}
}

// ---- Shortcuts ----

func (vm *ViewModel) shortcuts(args []string) Result {
	if len(args) > 0 && strings.EqualFold(args[0], "run") {
		if len(args) < 2 {
			return Result{Text: "Usage: /shortcuts run `name` [input]"}
		}
		if !macsvc.Available() {
			return notOnMac("/shortcuts")
		}
		// Everything after "run" is the name unless the user separated input
		// with "--", since Shortcut names routinely contain spaces.
		name, input := splitShortcutArgs(args[1:])
		out, err := macsvc.RunShortcut(name, input)
		if err != nil {
			return Result{Text: tgfmt.FailWith("shortcut "+name, err)}
		}
		if out == "" {
			return Result{Text: tgfmt.OKWith(fmt.Sprintf("Ran shortcut %q", name))}
		}
		return Result{Text: tgfmt.OKWith(fmt.Sprintf("Ran shortcut %q", name)) + "\n" + tgfmt.Code(out)}
	}

	if !macsvc.Available() {
		return notOnMac("/shortcuts")
	}
	names, err := macsvc.ListShortcuts()
	if err != nil {
		return Result{Text: tgfmt.FailWith("shortcuts", err)}
	}
	if len(names) == 0 {
		return Result{Text: "No Shortcuts found. Create one in the Shortcuts app first."}
	}
	return Result{Text: fmt.Sprintf("🧩 <b>%d Shortcut(s)</b>\n%s\n\nRun one with /shortcuts run `name`",
		len(names), tgfmt.List(names))}
}

// splitShortcutArgs separates the Shortcut name from its optional input. The
// name comes first and may contain spaces; "--" starts the input.
func splitShortcutArgs(args []string) (name, input string) {
	for i, a := range args {
		if a == "--" {
			return strings.Join(args[:i], " "), strings.Join(args[i+1:], " ")
		}
	}
	return strings.Join(args, " "), ""
}

// ---- Audio & media ----

func (vm *ViewModel) volume(args []string) Result {
	action, level, err := parseVolumeArg(args)
	if err != nil {
		return Result{Text: "Usage: /volume [`0-100` | `+10` | `-10` | up | down | mute | unmute]"}
	}
	if !macsvc.Available() {
		return notOnMac("/volume")
	}

	var state macsvc.VolumeState
	switch action {
	case volumeShow:
		state, err = macsvc.Volume()
	case volumeSet:
		state, err = macsvc.SetVolume(level)
	case volumeStep:
		state, err = macsvc.AdjustVolume(level)
	case volumeMute:
		state, err = macsvc.SetMuted(true)
	case volumeUnmute:
		state, err = macsvc.SetMuted(false)
	}
	if err != nil {
		return Result{Text: tgfmt.FailWith("volume", err)}
	}
	return Result{Text: state.Summary()}
}

type volumeAction int

const (
	volumeShow volumeAction = iota
	volumeSet
	volumeStep
	volumeMute
	volumeUnmute
)

// volumeStepSize is how far "up"/"down" move the volume.
const volumeStepSize = 10

// parseVolumeArg interprets the /volume argument. level carries the target
// level for volumeSet and the delta for volumeStep.
func parseVolumeArg(args []string) (action volumeAction, level int, err error) {
	if len(args) == 0 {
		return volumeShow, 0, nil
	}
	arg := strings.ToLower(strings.TrimSpace(args[0]))
	switch arg {
	case "", "show", "status":
		return volumeShow, 0, nil
	case "up", "+":
		return volumeStep, volumeStepSize, nil
	case "down", "-":
		return volumeStep, -volumeStepSize, nil
	case "mute":
		return volumeMute, 0, nil
	case "unmute":
		return volumeUnmute, 0, nil
	}
	n, convErr := strconv.Atoi(arg)
	if convErr != nil {
		return volumeShow, 0, fmt.Errorf("not a volume: %q", args[0])
	}
	if strings.HasPrefix(arg, "+") || strings.HasPrefix(arg, "-") {
		return volumeStep, n, nil
	}
	return volumeSet, n, nil
}

func (vm *ViewModel) media(args []string) Result {
	action := "info"
	if len(args) > 0 {
		action = strings.ToLower(strings.TrimSpace(args[0]))
	}
	if !macsvc.Available() {
		return notOnMac("/media")
	}

	if action == "info" || action == "now" || action == "nowplaying" {
		track, err := macsvc.CurrentTrack()
		if err != nil {
			return Result{Text: tgfmt.FailWith("now playing", err)}
		}
		return Result{Text: track.Summary()}
	}

	app, err := macsvc.Media(action)
	if err != nil {
		return Result{Text: tgfmt.FailWith("media "+action, err)}
	}
	return Result{Text: tgfmt.OKWith(fmt.Sprintf("%s → %s", app, action))}
}

func (vm *ViewModel) say(args []string) Result {
	text := strings.TrimSpace(strings.Join(args, " "))
	if text == "" {
		return Result{Text: "Usage: /say `text to speak out loud`"}
	}
	if !macsvc.Available() {
		return notOnMac("/say")
	}
	if err := macsvc.Say(text); err != nil {
		return Result{Text: tgfmt.FailWith("say", err)}
	}
	return Result{Text: "🗣 Speaking on the Mac."}
}

func (vm *ViewModel) notify(args []string) Result {
	text := strings.TrimSpace(strings.Join(args, " "))
	if text == "" {
		return Result{Text: "Usage: /notify `message to show on the Mac`"}
	}
	if !macsvc.Available() {
		return notOnMac("/notify")
	}
	if err := macsvc.Notify("Nullhand", text); err != nil {
		return Result{Text: tgfmt.FailWith("notify", err)}
	}
	return Result{Text: "🔔 Notification shown on the Mac."}
}

// ---- Power & appearance ----

func (vm *ViewModel) battery() Result {
	if !macsvc.Available() {
		return notOnMac("/battery")
	}
	b, err := macsvc.BatteryStatus()
	if err != nil {
		return Result{Text: tgfmt.FailWith("battery", err)}
	}
	return Result{Text: b.Summary()}
}

func (vm *ViewModel) lockScreen() Result {
	if !macsvc.Available() {
		return notOnMac("/lock")
	}
	if err := macsvc.LockScreen(); err != nil {
		return Result{Text: tgfmt.FailWith("lock", err)}
	}
	return Result{Text: "🔒 Screen locked."}
}

func (vm *ViewModel) sleepMac() Result {
	if !macsvc.Available() {
		return notOnMac("/sleep")
	}
	if err := macsvc.SleepNow(); err != nil {
		return Result{Text: tgfmt.FailWith("sleep", err)}
	}
	return Result{Text: "😴 Going to sleep."}
}

func (vm *ViewModel) awake(args []string) Result {
	d, stop, err := parseAwakeArg(args)
	if err != nil {
		return Result{Text: "Usage: /awake [`90` | `90m` | `2h` | off]"}
	}
	if !macsvc.Available() {
		return notOnMac("/awake")
	}

	if stop {
		if macsvc.StopKeepAwake() {
			return Result{Text: "💤 The Mac can sleep again."}
		}
		return Result{Text: "The Mac was not being kept awake."}
	}
	if d == 0 && len(args) == 0 {
		if until, ok := macsvc.KeepAwakeUntil(); ok {
			if until.IsZero() {
				return Result{Text: "☕ Kept awake until cancelled (/awake off)."}
			}
			return Result{Text: fmt.Sprintf("☕ Kept awake for another %s.", tgfmt.Duration(time.Until(until).Round(time.Minute)))}
		}
		return Result{Text: "💤 Not holding the Mac awake. Try /awake 90m."}
	}

	until, err := macsvc.KeepAwake(d)
	if err != nil {
		return Result{Text: tgfmt.FailWith("awake", err)}
	}
	if until.IsZero() {
		return Result{Text: "☕ Staying awake until you send /awake off."}
	}
	return Result{Text: fmt.Sprintf("☕ Staying awake for %s (until %s).", tgfmt.Duration(d), until.Format("15:04"))}
}

// parseAwakeArg reads the /awake argument: a bare number is minutes, "off"
// cancels, and no argument asks for the current state.
func parseAwakeArg(args []string) (d time.Duration, stop bool, err error) {
	if len(args) == 0 {
		return 0, false, nil
	}
	arg := strings.ToLower(strings.TrimSpace(args[0]))
	switch arg {
	case "off", "stop", "cancel":
		return 0, true, nil
	case "on", "forever":
		return 0, false, nil
	}

	unit := time.Minute
	value := arg
	switch {
	case strings.HasSuffix(arg, "h"):
		unit, value = time.Hour, strings.TrimSuffix(arg, "h")
	case strings.HasSuffix(arg, "m"):
		value = strings.TrimSuffix(arg, "m")
	}
	n, convErr := strconv.Atoi(value)
	if convErr != nil || n <= 0 {
		return 0, false, fmt.Errorf("not a duration: %q", args[0])
	}
	return time.Duration(n) * unit, false, nil
}

func (vm *ViewModel) darkMode(args []string) Result {
	arg := ""
	if len(args) > 0 {
		arg = strings.ToLower(strings.TrimSpace(args[0]))
	}
	switch arg {
	case "", "toggle", "on", "off", "status":
	default:
		return Result{Text: "Usage: /dark [on | off | toggle | status]"}
	}
	if !macsvc.Available() {
		return notOnMac("/dark")
	}

	var (
		dark bool
		err  error
	)
	switch arg {
	case "status":
		dark, err = macsvc.DarkMode()
	case "on":
		err = macsvc.SetDarkMode(true)
		dark = true
	case "off":
		err = macsvc.SetDarkMode(false)
		dark = false
	default: // no argument means toggle, the common case from a phone
		dark, err = macsvc.ToggleDarkMode()
	}
	if err != nil {
		return Result{Text: tgfmt.FailWith("dark mode", err)}
	}
	if dark {
		return Result{Text: "🌙 Dark mode is on."}
	}
	return Result{Text: "☀️ Dark mode is off."}
}

// ---- Finder ----

func (vm *ViewModel) reveal(args []string) Result {
	path := strings.TrimSpace(strings.Join(args, " "))
	if path == "" {
		return Result{Text: "Usage: /reveal `path to a file or folder`"}
	}
	if !macsvc.Available() {
		return notOnMac("/reveal")
	}
	if err := macsvc.Reveal(path); err != nil {
		return Result{Text: tgfmt.FailWith("reveal", err)}
	}
	return Result{Text: "📂 Selected in Finder: " + tgfmt.Escape(path)}
}

func (vm *ViewModel) fileInfo(args []string) Result {
	path := strings.TrimSpace(strings.Join(args, " "))
	if path == "" {
		return Result{Text: "Usage: /info `path to a file or folder`"}
	}
	if !macsvc.Available() {
		return notOnMac("/info")
	}
	info, err := macsvc.Info(path)
	if err != nil {
		return Result{Text: tgfmt.FailWith("info", err)}
	}
	return Result{Text: tgfmt.Escape(info.Summary())}
}

// trash moves an item to the Trash, which is recoverable and therefore needs
// no confirmation. Emptying the Trash is not recoverable, so "/trash" with no
// path is intercepted by the bot, which asks for /yes first; reaching here
// without a path only reports what is in there.
func (vm *ViewModel) trash(args []string) Result {
	if !macsvc.Available() {
		return notOnMac("/trash")
	}
	path := strings.TrimSpace(strings.Join(args, " "))
	if path == "" {
		state, err := macsvc.TrashStatus()
		if err != nil {
			return Result{Text: tgfmt.FailWith("trash", err)}
		}
		return Result{Text: state.Summary()}
	}
	if err := macsvc.MoveToTrash(path); err != nil {
		return Result{Text: tgfmt.FailWith("trash", err)}
	}
	return Result{Text: "🗑 Moved to the Trash: " + tgfmt.Escape(path) + "\nRecover it from Finder if you need it back."}
}
