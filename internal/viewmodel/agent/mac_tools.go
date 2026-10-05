package agent

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	macsvc "github.com/AzozzALFiras/Nullhand/internal/service/mac"
)

// The macOS-only tools. They are only offered to the AI on macOS (see
// macToolDefinitions), but the handlers below compile everywhere so a cloud
// model that invents one on Linux gets a clear answer rather than a crash.

// infoPrefix marks a tool result the user is meant to read. The offline
// parser replies "Done." unless a tool result starts with one of its known
// markers, which would otherwise swallow answers like the battery level.
const infoPrefix = "ℹ️ "

// infoParts returns a user-facing tool result.
func infoParts(s string) []aimodel.MessagePart { return textParts(infoPrefix + s) }

// failParts returns a tool error in the form the offline parser surfaces.
func failParts(err error) []aimodel.MessagePart { return textParts("⚠️ " + err.Error()) }

// executeMacTool handles the macOS tools. handled is false when the tool name
// belongs to someone else.
func (vm *ViewModel) executeMacTool(tc aimodel.ToolCall, sendPhoto PhotoFunc) (parts []aimodel.MessagePart, err error, handled bool) {
	args := tc.Arguments

	switch tc.ToolName {
	case "list_shortcuts":
		names, err := macsvc.ListShortcuts()
		if err != nil {
			return failParts(err), err, true
		}
		if len(names) == 0 {
			return infoParts("No Shortcuts exist on this Mac yet."), nil, true
		}
		return infoParts("Shortcuts:\n" + strings.Join(names, "\n")), nil, true

	case "run_shortcut":
		name := strings.TrimSpace(args["name"])
		if name == "" {
			e := fmt.Errorf("run_shortcut needs the shortcut name")
			return failParts(e), e, true
		}
		out, err := macsvc.RunShortcut(name, args["input"])
		if err != nil {
			return failParts(err), err, true
		}
		if out == "" {
			return textParts(fmt.Sprintf("ran shortcut %q", name)), nil, true
		}
		return infoParts(fmt.Sprintf("%s → %s", name, out)), nil, true

	case "control_audio":
		state, err := controlAudio(args)
		if err != nil {
			return failParts(err), err, true
		}
		return infoParts(state.Summary()), nil, true

	case "control_media":
		action := strings.TrimSpace(args["action"])
		if action == "" || action == "info" {
			track, err := macsvc.CurrentTrack()
			if err != nil {
				return failParts(err), err, true
			}
			return infoParts(track.Summary()), nil, true
		}
		app, err := macsvc.Media(action)
		if err != nil {
			return failParts(err), err, true
		}
		return textParts(fmt.Sprintf("%s: %s", app, action)), nil, true

	case "say_text":
		if err := macsvc.Say(args["text"]); err != nil {
			return failParts(err), err, true
		}
		return textParts("speaking on the Mac"), nil, true

	case "show_notification":
		title := args["title"]
		if title == "" {
			title = "Nullhand"
		}
		if err := macsvc.Notify(title, args["text"]); err != nil {
			return failParts(err), err, true
		}
		return textParts("notification shown on the Mac"), nil, true

	case "mac_power":
		parts, err := macPower(args)
		return parts, err, true

	case "reveal_in_finder":
		path := strings.TrimSpace(args["path"])
		if path == "" {
			e := fmt.Errorf("reveal_in_finder needs a path")
			return failParts(e), e, true
		}
		if err := macsvc.Reveal(path); err != nil {
			return failParts(err), err, true
		}
		return textParts("selected in Finder: " + path), nil, true

	case "file_info":
		path := strings.TrimSpace(args["path"])
		if path == "" {
			e := fmt.Errorf("file_info needs a path")
			return failParts(e), e, true
		}
		info, err := macsvc.Info(path)
		if err != nil {
			return failParts(err), err, true
		}
		return infoParts(info.Summary()), nil, true

	case "manage_trash":
		parts, err := manageTrash(args)
		return parts, err, true
	}

	return nil, nil, false
}

// controlAudio applies a volume action and returns the resulting state.
func controlAudio(args map[string]string) (macsvc.VolumeState, error) {
	switch strings.TrimSpace(args["action"]) {
	case "", "info", "status":
		return macsvc.Volume()
	case "set":
		percent, err := strconv.Atoi(strings.TrimSpace(args["percent"]))
		if err != nil {
			return macsvc.VolumeState{}, fmt.Errorf("control_audio: percent must be a number 0-100")
		}
		return macsvc.SetVolume(percent)
	case "up":
		return macsvc.AdjustVolume(volumeStep)
	case "down":
		return macsvc.AdjustVolume(-volumeStep)
	case "mute":
		return macsvc.SetMuted(true)
	case "unmute":
		return macsvc.SetMuted(false)
	default:
		return macsvc.VolumeState{}, fmt.Errorf("control_audio: unknown action %q (use set, up, down, mute, unmute, info)", args["action"])
	}
}

// volumeStep matches the step size the /volume command uses.
const volumeStep = 10

// macPower handles the battery, lock, sleep, keep-awake and appearance actions.
func macPower(args map[string]string) ([]aimodel.MessagePart, error) {
	switch strings.TrimSpace(args["action"]) {
	case "battery", "":
		b, err := macsvc.BatteryStatus()
		if err != nil {
			return failParts(err), err
		}
		return infoParts(b.Summary()), nil

	case "lock":
		if err := macsvc.LockScreen(); err != nil {
			return failParts(err), err
		}
		return textParts("screen locked"), nil

	case "sleep":
		if err := macsvc.SleepNow(); err != nil {
			return failParts(err), err
		}
		return textParts("the Mac is going to sleep"), nil

	case "awake":
		minutes, _ := strconv.Atoi(strings.TrimSpace(args["minutes"]))
		until, err := macsvc.KeepAwake(time.Duration(minutes) * time.Minute)
		if err != nil {
			return failParts(err), err
		}
		if until.IsZero() {
			return infoParts("The Mac will stay awake until cancelled."), nil
		}
		return infoParts(fmt.Sprintf("The Mac will stay awake until %s.", until.Format("15:04"))), nil

	case "awake_off":
		if macsvc.StopKeepAwake() {
			return infoParts("The Mac can sleep again."), nil
		}
		return infoParts("The Mac was not being kept awake."), nil

	case "dark_toggle", "dark_on", "dark_off":
		dark, err := setAppearance(args["action"])
		if err != nil {
			return failParts(err), err
		}
		if dark {
			return infoParts("Dark mode is on."), nil
		}
		return infoParts("Dark mode is off."), nil

	default:
		err := fmt.Errorf("mac_power: unknown action %q (use battery, lock, sleep, awake, awake_off, dark_toggle)", args["action"])
		return failParts(err), err
	}
}

func setAppearance(action string) (bool, error) {
	switch action {
	case "dark_on":
		return true, macsvc.SetDarkMode(true)
	case "dark_off":
		return false, macsvc.SetDarkMode(false)
	default:
		return macsvc.ToggleDarkMode()
	}
}

// manageTrash reports the Trash or moves an item into it. Emptying is
// refused: it cannot be undone, so it needs the /trash + /yes flow where the
// user sees what would be lost first.
func manageTrash(args map[string]string) ([]aimodel.MessagePart, error) {
	switch strings.TrimSpace(args["action"]) {
	case "", "status":
		state, err := macsvc.TrashStatus()
		if err != nil {
			return failParts(err), err
		}
		return infoParts(state.Summary()), nil

	case "move":
		path := strings.TrimSpace(args["path"])
		if path == "" {
			err := fmt.Errorf("manage_trash move needs a path")
			return failParts(err), err
		}
		if err := macsvc.MoveToTrash(path); err != nil {
			return failParts(err), err
		}
		return textParts("moved to the Trash (recoverable): " + path), nil

	case "empty":
		err := fmt.Errorf("emptying the Trash cannot be undone — tell the user to send /trash and confirm with /yes")
		return failParts(err), err

	default:
		err := fmt.Errorf("manage_trash: unknown action %q (use status, move or empty)", args["action"])
		return failParts(err), err
	}
}

// macToolDefinitions describes the macOS tools for the AI. They are added to
// the tool list only on macOS, so a Linux run neither pays for the tokens nor
// lets the model call something that cannot work.
func macToolDefinitions() []aimodel.ToolDefinition {
	return []aimodel.ToolDefinition{
		{
			Name: "list_shortcuts",
			Description: "List the macOS Shortcuts available on this Mac. Call this before " +
				"run_shortcut when the user's wording may not match a shortcut name exactly.",
		},
		{
			Name: "run_shortcut",
			Description: "Run a macOS Shortcut by name and return any text it produces. " +
				"Shortcuts can control HomeKit devices, toggle system settings and run multi-step automations.",
			Parameters: []aimodel.ToolParameter{
				{Name: "name", Type: "string", Description: "Exact Shortcut name", Required: true},
				{Name: "input", Type: "string", Description: "Optional text input for the Shortcut"},
			},
		},
		{
			Name:        "control_audio",
			Description: "Read or change the Mac's system output volume.",
			Parameters: []aimodel.ToolParameter{
				{Name: "action", Type: "string", Description: "info, set, up, down, mute or unmute", Required: true},
				{Name: "percent", Type: "string", Description: "Target level 0-100, required when action is set"},
			},
		},
		{
			Name:        "control_media",
			Description: "Control Music or Spotify, whichever is running: info (what is playing), play, pause, next, previous.",
			Parameters: []aimodel.ToolParameter{
				{Name: "action", Type: "string", Description: "info, play, pause, next or previous", Required: true},
			},
		},
		{
			Name:        "say_text",
			Description: "Speak text out loud through the Mac's speakers. Use it to get the attention of someone near the Mac.",
			Parameters: []aimodel.ToolParameter{
				{Name: "text", Type: "string", Description: "What to say", Required: true},
			},
		},
		{
			Name:        "show_notification",
			Description: "Show a macOS notification banner on the Mac's screen.",
			Parameters: []aimodel.ToolParameter{
				{Name: "text", Type: "string", Description: "Notification body", Required: true},
				{Name: "title", Type: "string", Description: "Notification title (default Nullhand)"},
			},
		},
		{
			Name:        "reveal_in_finder",
			Description: "Open a Finder window on the Mac with the given file or folder selected.",
			Parameters: []aimodel.ToolParameter{
				{Name: "path", Type: "string", Description: "Path to reveal", Required: true},
			},
		},
		{
			Name:        "file_info",
			Description: "Describe a file or folder: kind, size (folders are totalled), item count, and the created and modified dates.",
			Parameters: []aimodel.ToolParameter{
				{Name: "path", Type: "string", Description: "Path to describe", Required: true},
			},
		},
		{
			Name: "manage_trash",
			Description: "Report what is in the Trash (status) or move a file into it (move), which is " +
				"recoverable and the safe way to delete something. Emptying the Trash is not available here.",
			Parameters: []aimodel.ToolParameter{
				{Name: "action", Type: "string", Description: "status or move", Required: true},
				{Name: "path", Type: "string", Description: "Path to move to the Trash, for action move"},
			},
		},
		{
			Name: "mac_power",
			Description: "Battery and power control on the Mac: battery (charge level), lock (lock the screen), " +
				"sleep, awake (keep awake), awake_off, dark_toggle, dark_on, dark_off.",
			Parameters: []aimodel.ToolParameter{
				{Name: "action", Type: "string", Description: "battery, lock, sleep, awake, awake_off, dark_toggle, dark_on or dark_off", Required: true},
				{Name: "minutes", Type: "string", Description: "How long to stay awake, for action awake (0 = until cancelled)"},
			},
		},
	}
}
