package local

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
)

// intent is a pattern that matches a user utterance and produces tool calls.
type intent struct {
	re    *regexp.Regexp
	build func(matches []string) []aimodel.ToolCall
}

// ── App name normalization ──────────────────────────────────────────────
// Maps common names (in any language) to the correct macOS app name.
var appNameMap = map[string]string{
	// English
	"terminal": "Terminal", "term": "Terminal", "iterm": "iTerm",
	"safari": "Safari", "chrome": "Google Chrome", "google chrome": "Google Chrome",
	"firefox": "Firefox", "brave": "Brave Browser",
	"vscode": "Visual Studio Code", "vs code": "Visual Studio Code", "code": "Visual Studio Code",
	"cursor": "Cursor",
	"whatsapp": "WhatsApp", "wa": "WhatsApp",
	"slack": "Slack", "discord": "Discord",
	"telegram": "Telegram", "tg": "Telegram",
	"messages": "Messages", "imessage": "Messages",
	"mail": "Mail", "notes": "Notes", "finder": "Finder",
	"music": "Music", "spotify": "Spotify",
	"xcode": "Xcode", "photos": "Photos",
	"preview": "Preview", "calculator": "Calculator",
	"activity monitor": "Activity Monitor",
	"system settings": "System Settings", "settings": "System Settings",
	// Arabic
	"التيرمنل": "Terminal", "الطرفية": "Terminal", "تيرمنل": "Terminal",
	"سفاري": "Safari", "كروم": "Google Chrome", "قوقل كروم": "Google Chrome",
	"واتساب": "WhatsApp", "واتس": "WhatsApp", "الواتساب": "WhatsApp", "الواتس": "WhatsApp",
	"سلاك": "Slack", "ديسكورد": "Discord",
	"تلقرام": "Telegram", "تليقرام": "Telegram",
	"الرسائل": "Messages", "فايندر": "Finder",
	"الملاحظات": "Notes", "البريد": "Mail",
	"الاعدادات": "System Settings", "الضبط": "System Settings",
}

// resolveAppName normalizes an app name using the map, or returns it as-is.
func resolveAppName(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if mapped, ok := appNameMap[lower]; ok {
		return mapped
	}
	return strings.TrimSpace(raw)
}

// ── Smart intents (multi-step, context-aware) ──────────────────────────
// These match complex sentences BEFORE the simple intents.
var smartIntents = []intent{
	// ── Terminal: run command ─────────────────────────────────────────
	// "open terminal and run ls -la" / "افتح التيرمنل ونفذ ls -la"
	{
		re: regexp.MustCompile(`(?i)^(?:open|launch|افتح|شغل)\s+(.+?)\s+(?:and\s+(?:run|execute|type)|then\s+(?:run|execute|type)|و\s*(?:نفذ|شغل|اكتب|نفّذ)|ثم\s+(?:نفذ|شغل|اكتب))\s+(.+?)$`),
		build: func(m []string) []aimodel.ToolCall {
			app := resolveAppName(m[1])
			cmd := strings.TrimSpace(m[2])
			if app == "Terminal" || app == "iTerm" {
				return []aimodel.ToolCall{toolCall("run_recipe", map[string]string{
					"name":        "terminal_run_command",
					"params_json": mustJSON(map[string]string{"command": cmd}),
				})}
			}
			// VS Code terminal
			if app == "Visual Studio Code" || app == "Cursor" {
				return []aimodel.ToolCall{toolCall("run_recipe", map[string]string{
					"name":        "vscode_terminal_run",
					"params_json": mustJSON(map[string]string{"command": cmd}),
				})}
			}
			// Generic: open app, type, send
			return []aimodel.ToolCall{
				toolCall("open_app", map[string]string{"app_name": app}),
				toolCall("wait", map[string]string{"ms": "300"}),
				toolCall("type_text", map[string]string{"text": cmd}),
				toolCall("press_key", map[string]string{"key": "return"}),
			}
		},
	},

	// ── WhatsApp/Slack/Discord: send message to contact ──────────────
	// "open whatsapp and send Azozz a message hello" / "افتح واتساب وارسل لعزوز مرحبا"
	{
		re: regexp.MustCompile(`(?i)^(?:open|افتح)\s+(?:whatsapp|واتساب|واتس|الواتساب|الواتس)\s+(?:and\s+(?:send|message|write)|و\s*(?:ارسل|اكتب|راسل))\s+(?:to\s+|ل|لـ\s*)?(.+?)\s+(?:a\s+message\s+|message\s+|رسالة\s+|وقل(?:\s+له)?\s+|قل(?:\s+له)?\s+)(.+?)$`),
		build: func(m []string) []aimodel.ToolCall {
			contact := strings.TrimSpace(m[1])
			message := stripQuotes(strings.TrimSpace(m[2]))
			return []aimodel.ToolCall{toolCall("run_recipe", map[string]string{
				"name":        "whatsapp_send_message",
				"params_json": mustJSON(map[string]string{"contact": contact, "message": message}),
			})}
		},
	},

	// ── WhatsApp: open chat (no message) ─────────────────────────────
	// "open whatsapp chat with Azozz" / "افتح واتساب محادثة عزوز"
	{
		re: regexp.MustCompile(`(?i)^(?:open|افتح)\s+(?:whatsapp|واتساب|واتس|الواتساب|الواتس)\s+(?:chat\s+(?:with\s+)?|محادثة\s+)(.+?)$`),
		build: func(m []string) []aimodel.ToolCall {
			contact := strings.TrimSpace(m[1])
			return []aimodel.ToolCall{toolCall("run_recipe", map[string]string{
				"name":        "whatsapp_new_message",
				"params_json": mustJSON(map[string]string{"contact": contact}),
			})}
		},
	},

	// ── Browser: open URL ────────────────────────────────────────────
	// "open safari and go to github.com" / "افتح سفاري وروح لـ github.com"
	{
		re: regexp.MustCompile(`(?i)^(?:open|افتح)\s+(.+?)\s+(?:and\s+(?:go\s+to|navigate\s+to|open)|و\s*(?:روح|اذهب|انتقل)\s+(?:ل|لـ|الى|إلى)\s*)(.+?)$`),
		build: func(m []string) []aimodel.ToolCall {
			app := resolveAppName(m[1])
			url := strings.TrimSpace(m[2])
			if isBrowser(app) {
				return []aimodel.ToolCall{toolCall("run_recipe", map[string]string{
					"name":        "browser_open_url",
					"params_json": mustJSON(map[string]string{"browser": app, "url": url}),
				})}
			}
			return []aimodel.ToolCall{
				toolCall("open_app", map[string]string{"app_name": app}),
			}
		},
	},

	// ── Browser: search ──────────────────────────────────────────────
	// "open safari and search for X" / "افتح سفاري وابحث عن X"
	{
		re: regexp.MustCompile(`(?i)^(?:open|افتح)\s+(.+?)\s+(?:and\s+(?:search|google|look)\s+(?:for\s+)?|و\s*(?:ابحث|بحث)\s+(?:عن\s+)?)(.+?)$`),
		build: func(m []string) []aimodel.ToolCall {
			app := resolveAppName(m[1])
			query := strings.TrimSpace(m[2])
			if isBrowser(app) {
				return []aimodel.ToolCall{toolCall("run_recipe", map[string]string{
					"name":        "browser_google_search",
					"params_json": mustJSON(map[string]string{"browser": app, "query": query}),
				})}
			}
			return []aimodel.ToolCall{
				toolCall("open_app", map[string]string{"app_name": app}),
			}
		},
	},

	// ── Search Google directly ───────────────────────────────────────
	// "search for X" / "ابحث عن X" / "google X"
	{
		re: regexp.MustCompile(`(?i)^(?:search|google|ابحث|بحث)\s+(?:for\s+|about\s+|عن\s+)?(.+?)$`),
		build: func(m []string) []aimodel.ToolCall {
			query := strings.TrimSpace(m[1])
			return []aimodel.ToolCall{toolCall("run_recipe", map[string]string{
				"name":        "browser_google_search",
				"params_json": mustJSON(map[string]string{"browser": "Safari", "query": query}),
			})}
		},
	},

	// ── VS Code Claude: type and send ────────────────────────────────
	// Many patterns: "in vs code write/type/send/tell ... claude/box ..."
	{
		re: regexp.MustCompile(`(?i)^(?:(?:in\s+)?(?:vs\s*code|vscode|visual\s+studio\s+code)\s+.*?(?:write|type|send|told?|tell)\s+.*?(?:box|claude|chat|message)\s+.*?[""\"](.+?)[""\"]\s*(?:and\s+send)?.*|open\s+(?:vs\s*code|vscode|visual\s+studio\s+code)\s+and\s+(?:write|type|send|tell)\s+(?:in\s+|to\s+)?(?:claude|the\s+box)\s+[""\"]?(.+?)[""\"]?\s*(?:and\s+send)?.*|tell\s+claude\s+(?:in\s+)?(?:vs\s*code|vscode)\s+[""\"]?(.+?)[""\"]?\s*(?:and\s+send)?.*)$`),
		build: func(m []string) []aimodel.ToolCall {
			// Find first non-empty capture group
			message := ""
			for _, g := range m[1:] {
				if strings.TrimSpace(g) != "" {
					message = stripQuotes(strings.TrimSpace(g))
					break
				}
			}
			if message == "" {
				message = "hello"
			}
			return []aimodel.ToolCall{toolCall("run_recipe", map[string]string{
				"name":        "vscode_type_in_claude",
				"params_json": mustJSON(map[string]string{"message": message}),
			})}
		},
	},

	// ── VS Code: open terminal ───────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:open|افتح)\s+(?:vs\s*code|vscode|visual\s+studio\s+code)\s+(?:and\s+open\s+)?(?:terminal|تيرمنل|الطرفية)$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("run_recipe", map[string]string{
				"name": "vscode_open_terminal",
			})}
		},
	},

	// ── Open app and type and send ───────────────────────────────────
	// "open X and type Y and send" / "افتح X واكتب Y وارسل"
	{
		re: regexp.MustCompile(`(?i)^(?:open|افتح)\s+(.+?)\s+(?:and\s+(?:type|write)|و\s*(?:اكتب|أكتب))\s+(.+?)\s+(?:and\s+send|و\s*(?:ارسل|أرسل))$`),
		build: func(m []string) []aimodel.ToolCall {
			app := resolveAppName(m[1])
			text := stripQuotes(strings.TrimSpace(m[2]))
			return []aimodel.ToolCall{
				toolCall("open_app", map[string]string{"app_name": app}),
				toolCall("wait", map[string]string{"ms": "300"}),
				toolCall("type_text", map[string]string{"text": text}),
				toolCall("press_key", map[string]string{"key": "return"}),
			}
		},
	},
}

// intents is the ordered list of simple intent patterns.
var intents = []intent{
	// ── Screenshot ────────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:take\s+(?:a\s+)?screenshot|screenshot|snap(?:shot)?|لقطة(?:\s*شاشة)?|خذ\s+لقطة(?:\s+شاشة)?|التقط\s+(?:لقطة|شاشة|صورة)|ارسل\s+(?:لقطة|سكرين|صورة))\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("take_screenshot", nil)}
		},
	},

	// ── Clipboard: paste / get ────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:paste|الصق|اعرض\s+الحافظة|clipboard)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("get_clipboard", nil)}
		},
	},

	// ── Clipboard: copy <text> ────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:copy|انسخ)\s+(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("set_clipboard", map[string]string{"text": strings.TrimSpace(m[1])})}
		},
	},

	// ── Send / submit (press return) ──────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:send|submit|ارسل|أرسل|اضغط\s+(?:ارسال|enter|return|إدخال))\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("press_key", map[string]string{"key": "return"})}
		},
	},

	// ── Cancel (ctrl+c) ──────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:cancel|stop|abort|الغ|الغاء|اوقف|ctrl\+c)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("press_key", map[string]string{"key": "ctrl+c"})}
		},
	},

	// ── Press key: "press cmd+t" / "اضغط cmd+t" ───────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:press|hit|اضغط)\s+(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("press_key", map[string]string{"key": strings.TrimSpace(m[1])})}
		},
	},

	// ── Open app ──────────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:open|launch|start|افتح|شغّل\s+تطبيق|شغل)\s+(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			app := resolveAppName(m[1])
			return []aimodel.ToolCall{toolCall("open_app", map[string]string{"app_name": app})}
		},
	},

	// ── Type / write ──────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:type|write|اكتب|أكتب)\s+(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			text := stripQuotes(strings.TrimSpace(m[1]))
			return []aimodel.ToolCall{toolCall("type_text", map[string]string{"text": text})}
		},
	},

	// ── Click UI element by label ─────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:click|press\s+button|انقر|اضغط\s+(?:على\s+)?زر)\s+(?:on\s+)?(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			label := stripQuotes(strings.TrimSpace(m[1]))
			return []aimodel.ToolCall{toolCall("click_ui_element", map[string]string{
				"app_name": "",
				"label":    label,
			})}
		},
	},

	// ── Click at coordinates ──────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:click|انقر|اضغط)\s+(?:at\s+)?(\d+)\s*[,\s]\s*(\d+)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("click", map[string]string{"x": m[1], "y": m[2]})}
		},
	},

	// ── Scroll ────────────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:scroll|مرر)\s+(up|down|left|right|فوق|تحت|يمين|يسار)(?:\s+(\d+))?\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			dir := strings.ToLower(m[1])
			switch dir {
			case "فوق":
				dir = "up"
			case "تحت":
				dir = "down"
			case "يمين":
				dir = "right"
			case "يسار":
				dir = "left"
			}
			steps := "3"
			if m[2] != "" {
				steps = m[2]
			}
			return []aimodel.ToolCall{toolCall("scroll", map[string]string{"direction": dir, "steps": steps})}
		},
	},

	// ── Wait ──────────────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:wait|sleep|انتظر)\s+(\d+)(?:\s*(?:ms|milliseconds|ملي))?\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("wait", map[string]string{"ms": m[1]})}
		},
	},

	// ── List recipes ──────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:list\s+recipes|recipes|الوصفات|اعرض\s+الوصفات|help|مساعدة)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("list_recipes", nil)}
		},
	},

	// ── Run recipe ────────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:recipe|run\s+recipe|وصفة|نفذ\s+وصفة)\s+(\S+)(?:\s+(.+))?\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			name := strings.TrimSpace(m[1])
			args := map[string]string{"name": name}
			if m[2] != "" {
				params := parseKeyValuePairs(m[2])
				if params != "" {
					args["params_json"] = params
				}
			}
			return []aimodel.ToolCall{toolCall("run_recipe", args)}
		},
	},

	// ── Status ────────────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:status|حالة|الحالة|معلومات)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("run_shell", map[string]string{"command": "echo CPU: $(top -l 1 | grep 'CPU usage' | head -1) && echo MEM: $(top -l 1 | grep 'PhysMem' | head -1)"})}
		},
	},

	// ── Read file ─────────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:read|show|cat|اقرأ|اعرض\s+محتوى)\s+(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("read_file", map[string]string{"path": strings.TrimSpace(m[1])})}
		},
	},

	// ── List directory ────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:list|ls|dir|اعرض|محتوى\s+مجلد)\s+(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("list_directory", map[string]string{"path": strings.TrimSpace(m[1])})}
		},
	},

	// ── Run shell ─────────────────────────────────────────────────────────
	{
		re: regexp.MustCompile(`(?i)^(?:run|shell|exec|execute|شغل|نفذ)\s+(.+?)\.?$`),
		build: func(m []string) []aimodel.ToolCall {
			return []aimodel.ToolCall{toolCall("run_shell", map[string]string{"command": strings.TrimSpace(m[1])})}
		},
	},
}

// isBrowser returns true if the app name is a known browser.
func isBrowser(app string) bool {
	switch app {
	case "Safari", "Google Chrome", "Firefox", "Brave Browser", "Arc":
		return true
	}
	return false
}

// mustJSON marshals a map to a JSON string. Panics on error (should never happen).
func mustJSON(m map[string]string) string {
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// parseKeyValuePairs converts "key1=val1 key2=val2" to a JSON object string.
func parseKeyValuePairs(s string) string {
	parts := strings.Fields(strings.TrimSpace(s))
	pairs := []string{}
	for _, p := range parts {
		idx := strings.IndexByte(p, '=')
		if idx > 0 {
			key := p[:idx]
			val := p[idx+1:]
			val = stripQuotes(val)
			pairs = append(pairs, fmt.Sprintf("%q:%q", key, val))
		}
	}
	if len(pairs) == 0 {
		return ""
	}
	return "{" + strings.Join(pairs, ",") + "}"
}

// stripQuotes removes a matching pair of surrounding quotes from s.
func stripQuotes(s string) string {
	if len(s) >= 2 {
		first, last := s[0], s[len(s)-1]
		if (first == '"' && last == '"') ||
			(first == '\'' && last == '\'') ||
			(first == '`' && last == '`') {
			return s[1 : len(s)-1]
		}
		if strings.HasPrefix(s, "\u201c") && strings.HasSuffix(s, "\u201d") {
			return strings.TrimSuffix(strings.TrimPrefix(s, "\u201c"), "\u201d")
		}
	}
	return s
}
