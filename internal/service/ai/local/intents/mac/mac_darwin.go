//go:build darwin

package mac

import (
	"regexp"
	"strconv"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
)

// call is a small helper for the one-tool intents below.
func call(tool string, args map[string]string) []aimodel.ToolCall {
	return []aimodel.ToolCall{intents.ToolCall(tool, args)}
}

func init() {
	// Registered in both registries on purpose: as smart intents they are
	// matched on the full text in phase 0, ahead of the entity classifier
	// that would otherwise read "شغل الموسيقى" as "open the Music app" and
	// "اقفل الشاشة" as "close a window"; as simple intents they stay usable
	// inside a chain such as "mute and lock the screen".
	set := macIntents()
	intents.RegisterSmart(set...)
	intents.RegisterSimple(set...)
}

// macIntents returns the macOS intent set. Everything is PriorityVeryHigh:
// these phrases name a macOS capability outright, so they should win over the
// generic app/browser patterns, and ties are broken by this package being
// imported first.
func macIntents() []intents.Intent {
	return withPriority(
		// ── Volume ───────────────────────────────────────────────────
		intents.Intent{
			// "volume 40", "vol 40%", "الصوت 40", "اضبط الصوت على 40"
			Re: regexp.MustCompile(`(?i)^(?:volume|vol|الصوت|اضبط\s+الصوت(?:\s+على)?|خلي\s+الصوت)\s*(\d{1,3})\s*%?\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("control_audio", map[string]string{"action": "set", "percent": m[1]})
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:volume\s+up|turn\s+(?:it\s+)?up|louder|ارفع\s+الصوت|علي\s+الصوت|زود\s+الصوت)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("control_audio", map[string]string{"action": "up"})
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:volume\s+down|turn\s+(?:it\s+)?down|quieter|اخفض\s+الصوت|نزل\s+الصوت|وطي\s+الصوت)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("control_audio", map[string]string{"action": "down"})
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:mute|اكتم(?:\s+الصوت)?|اسكت(?:\s+الصوت)?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("control_audio", map[string]string{"action": "mute"})
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:unmute|ال(?:غ|غي)\s+الكتم|رجع\s+الصوت)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("control_audio", map[string]string{"action": "unmute"})
			},
		},

		// ── Music / Spotify ──────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:what(?:'s|\s+is)\s+playing|now\s+playing|current\s+(?:song|track)|ما\s+يعمل\s+الان|شنو\s+يشتغل|الاغنية\s+الحالية|الأغنية\s+الحالية)\s*\??\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("control_media", map[string]string{"action": "info"})
			},
		},
		intents.Intent{
			// "next"/"التالي" on their own usually mean a Next button on
			// screen, so a track skip has to say so.
			Re: regexp.MustCompile(`(?i)^(?:next\s+(?:track|song)|skip\s+(?:track|song)|الاغنية\s+التالية|الأغنية\s+التالية|اغنية\s+تالية)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("control_media", map[string]string{"action": "next"})
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:previous\s+(?:track|song)|الاغنية\s+السابقة|الأغنية\s+السابقة)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("control_media", map[string]string{"action": "previous"})
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:pause(?:\s+(?:the\s+)?music)?|stop\s+(?:the\s+)?music|(?:أ|ا)وقف\s+(?:الموسيقى|الاغنية)|وقف\s+الموسيقى)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("control_media", map[string]string{"action": "pause"})
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:play(?:\s+(?:the\s+)?music)?|resume|شغل\s+(?:الموسيقى|الاغنية)|كمل\s+الموسيقى)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("control_media", map[string]string{"action": "play"})
			},
		},

		// ── Shortcuts ────────────────────────────────────────────────
		intents.Intent{
			// Simple intents are matched in registration order, and this
			// package is imported before the generic "run <cmd>" shell intent
			// that would otherwise treat the Shortcut name as a command.
			Re: regexp.MustCompile(`(?i)^(?:run\s+shortcut|shortcut|(?:شغل|نفذ)\s+(?:ال)?اختصار)\s+(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("run_shortcut", map[string]string{"name": intents.StripQuotes(m[1])})
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:shortcuts|list\s+shortcuts|الاختصارات|اعرض\s+الاختصارات)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("list_shortcuts", nil)
			},
		},

		// ── Speech & notifications ───────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:say|speak)\s+(.+)$|^(?:قل|احكي|تكلم)\s+(.+)$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("say_text", map[string]string{"text": intents.StripQuotes(firstNonEmpty(m[1:]))})
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:notify|notification)\s+(.+)$|^(?:نبه|تنبيه|اشعار)\s+(.+)$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("show_notification", map[string]string{"text": intents.StripQuotes(firstNonEmpty(m[1:]))})
			},
		},

		// ── Spotlight & Quick Look ───────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:find(?:\s+files?)?|spotlight|ابحث\s+عن\s+ملف|جد\s+ملف|دور\s+على\s+ملف)\s+(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("find_files", map[string]string{"query": intents.StripQuotes(m[1])})
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:preview|quick\s*look|معاينة\s+ملف|اعرض\s+صورة)\s+(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("preview_file", map[string]string{"path": intents.StripQuotes(m[1])})
			},
		},

		// ── Power & appearance ───────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:battery(?:\s+status)?|البطارية|كم\s+البطارية|نسبة\s+البطارية)\s*\??\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("mac_power", map[string]string{"action": "battery"})
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:lock(?:\s+(?:the\s+)?screen)?|اقفل(?:\s+الشاشة)?|قفل\s+الشاشة)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("mac_power", map[string]string{"action": "lock"})
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:sleep|go\s+to\s+sleep|نام|نوم|ارقد)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("mac_power", map[string]string{"action": "sleep"})
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:dark\s+mode|toggle\s+dark(?:\s+mode)?|light\s+mode|الوضع\s+(?:الليلي|الداكن|المظلم))\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("mac_power", map[string]string{"action": "dark_toggle"})
			},
		},
		intents.Intent{
			// "keep awake", "keep awake 90", "لا تنم ساعتين" → minutes when given
			Re: regexp.MustCompile(`(?i)^(?:keep\s+awake|stay\s+awake|لا\s+تنم|ابق(?:ى)?\s+مستيقظ)(?:\s+(\d{1,4})\s*(m|min|minutes|h|hours|دقيقة|ساعة|ساعات)?)?\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				args := map[string]string{"action": "awake"}
				if m[1] != "" {
					args["minutes"] = strconv.Itoa(toMinutes(m[1], m[2]))
				}
				return call("mac_power", args)
			},
		},
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:awake\s+off|stop\s+keeping\s+awake|يمكنك\s+النوم|ارتاح)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return call("mac_power", map[string]string{"action": "awake_off"})
			},
		},
	)
}

// withPriority stamps PriorityVeryHigh on every intent in the set.
func withPriority(set ...intents.Intent) []intents.Intent {
	for i := range set {
		set[i].Priority = intents.PriorityVeryHigh
	}
	return set
}

// toMinutes converts a captured amount plus optional unit into minutes.
func toMinutes(amount, unit string) int {
	n, err := strconv.Atoi(amount)
	if err != nil || n <= 0 {
		return 0
	}
	switch unit {
	case "h", "hours", "ساعة", "ساعات":
		return n * 60
	default:
		return n
	}
}

// firstNonEmpty returns the first non-empty capture group, which is how the
// bilingual patterns above keep English and Arabic in one regexp.
func firstNonEmpty(groups []string) string {
	for _, g := range groups {
		if g != "" {
			return g
		}
	}
	return ""
}
