package telegram

import (
	"fmt"
	"strings"
	"time"
)

// OK returns a simple success reply.
func OK() string { return "✅" }

// OKWith returns a success reply with a message.
func OKWith(msg string) string { return "✅ " + msg }

// Fail formats an error reply.
func Fail(err error) string {
	return "❌ " + escapeHTML(err.Error())
}

// FailWith formats an error reply with context label.
func FailWith(label string, err error) string {
	return fmt.Sprintf("❌ <b>%s</b>: %s", escapeHTML(label), escapeHTML(err.Error()))
}

// Code wraps text in an HTML code block for Telegram.
func Code(text string) string {
	return "<pre>" + escapeHTML(text) + "</pre>"
}

// Bold wraps text in HTML bold for Telegram.
func Bold(text string) string {
	return "<b>" + escapeHTML(text) + "</b>"
}

// List formats a string slice as a numbered list.
func List(items []string) string {
	if len(items) == 0 {
		return "(empty)"
	}
	var sb strings.Builder
	for i, item := range items {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, escapeHTML(item)))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// Progress formats a step update during a long AI task.
func Progress(step, message string) string {
	return fmt.Sprintf("⏳ <b>%s</b>\n%s", escapeHTML(step), escapeHTML(message))
}

// AgentDone formats the final reply after an AI task completes.
// The result is external AI output and must be escaped before being sent
// with parse_mode=HTML.
func AgentDone(result string) string {
	if result == "" {
		return "✅ Done."
	}
	return escapeHTML(result)
}

// Confirm formats a dangerous-action confirmation prompt.
func Confirm(description string) string {
	return fmt.Sprintf(
		"⚠️ <b>Confirm action</b>\n\n%s\n\nSend /yes to proceed or /no to cancel.",
		escapeHTML(description),
	)
}

// ConfirmCommand formats the prompt shown before a destructive shell command
// runs.
func ConfirmCommand(cmdLine, reason string, expiresIn time.Duration) string {
	return fmt.Sprintf(
		"⚠️ <b>Confirm destructive command</b>\n\n<code>%s</code>\n%s.\n\nSend /yes within %s to run it, or /no to cancel.",
		escapeHTML(cmdLine), escapeHTML(reason), Duration(expiresIn),
	)
}

// Duration renders d compactly for chat: whole minutes as "2 min", anything
// else in time.Duration's own format ("45s", "1m30s").
func Duration(d time.Duration) string {
	if d >= time.Minute && d%time.Minute == 0 {
		return fmt.Sprintf("%d min", d/time.Minute)
	}
	return d.String()
}

// Help returns the welcome + command reference shown by /start and /help.
func Help() string {
	return `<b>👋 Welcome to Nullhand</b>

Your invisible hand on the Linux machine.

<b>Two ways to control your machine:</b>

<b>1. Manual commands</b> (instant, no AI cost):
/screenshot — capture screen
/status — CPU, memory, active app
/apps — list running apps
/open &lt;app&gt; — open an application
/ls &lt;path&gt; — list directory
/read &lt;path&gt; — read a file
/shell &lt;cmd&gt; — run a whitelisted shell command (destructive ones ask for /yes)
/click &lt;x&gt; &lt;y&gt; — click at coordinates
/type &lt;text&gt; — type text
/key &lt;shortcut&gt; — press key (e.g. cmd+t)
/paste — get clipboard
/copy &lt;text&gt; — set clipboard
/find &lt;name&gt; [in &lt;folder&gt;] — search for files
/preview &lt;path&gt; — see a file as an image or excerpt

<b>2. AI agent mode</b> (natural language):
Just type a task in plain English or Arabic. Example:
<i>Open Safari and take a screenshot</i>
<i>افتح VS Code واعرض محتوى Desktop</i>

<b>Controls:</b>
/stop — cancel current AI task
/help — show this message

Tip: use /screenshot often — it is your eyes on the Linux machine.`
}

// MacHelp lists the macOS-only commands. It is appended to Help() on macOS.
func MacHelp() string {
	return `<b>🍎 macOS extras:</b>
/shortcuts — list Shortcuts; /shortcuts run &lt;name&gt; to run one
/volume &lt;0-100|up|down|mute&gt; — system volume
/media &lt;info|play|pause|next|prev&gt; — Music / Spotify
/battery — charge level and time left
/say &lt;text&gt; — speak out loud on the Mac
/notify &lt;text&gt; — show a notification on the Mac
/info &lt;path&gt; — kind, size, dates
/reveal &lt;path&gt; — select the item in Finder
/trash [&lt;path&gt;] — move to Trash, or empty it (asks for /yes)
/dark [on|off] — switch appearance
/lock, /sleep, /awake &lt;90m|off&gt; — screen and power`
}

// StatusReport formats a system status reply.
func StatusReport(cpu, mem, activeApp, screenSize string) string {
	return fmt.Sprintf(
		"<b>System Status</b>\n\nCPU: %s\nMemory: %s\nActive app: %s\nScreen: %s",
		escapeHTML(cpu), escapeHTML(mem), escapeHTML(activeApp), escapeHTML(screenSize),
	)
}

// Escape makes arbitrary text safe to embed in an HTML-mode message.
func Escape(s string) string { return escapeHTML(s) }

// escapeHTML escapes <, >, & for Telegram HTML parse mode.
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
