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

	case "find_files":
		query := strings.TrimSpace(args["query"])
		if query == "" {
			e := fmt.Errorf("find_files needs a query")
			return failParts(e), e, true
		}
		limit, _ := strconv.Atoi(args["limit"])
		if limit <= 0 {
			limit = 20
		}
		hits, err := macsvc.Find(query, args["folder"], limit)
		if err != nil {
			return failParts(err), err, true
		}
		if len(hits) == 0 {
			return infoParts(fmt.Sprintf("No files matching %q.", query)), nil, true
		}
		return infoParts(fmt.Sprintf("%d match(es) for %q:\n%s", len(hits), query, strings.Join(hits, "\n"))), nil, true

	case "preview_file":
		if sendPhoto == nil {
			e := fmt.Errorf("preview delivery is not available in this context")
			return failParts(e), e, true
		}
		path := strings.TrimSpace(args["path"])
		png, err := macsvc.Preview(path)
		if err != nil {
			return failParts(err), err, true
		}
		if err := sendPhoto(png, "🖼 "+path); err != nil {
			return failParts(err), err, true
		}
		return textParts("preview delivered to the user (the AI cannot see it)"), nil, true

	case "mac_power":
		parts, err := macPower(args)
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
			Name: "find_files",
			Description: "Find files by name using Spotlight, falling back to a file walk when " +
				"the folder is not indexed. Returns matching paths.",
			Parameters: []aimodel.ToolParameter{
				{Name: "query", Type: "string", Description: "Part of the file name", Required: true},
				{Name: "folder", Type: "string", Description: "Folder to search (default: the home folder)"},
				{Name: "limit", Type: "string", Description: "Maximum number of results (default 20)"},
			},
		},
		{
			Name: "preview_file",
			Description: "Render a Quick Look preview of a file (PDF, image, document) and deliver it " +
				"to the user's Telegram chat as a photo. You will NOT see the image.",
			Parameters: []aimodel.ToolParameter{
				{Name: "path", Type: "string", Description: "Path to the file", Required: true},
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
