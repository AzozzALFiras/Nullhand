package common

import (
	"regexp"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
)

func init() {
	intents.RegisterSmart(
		// ── Browse folder ────────────────────────────────────────────
		// Extremely flexible matching - any mention of browsing/listing + a path
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:` +
				// English variations (including typos)
				`(?:list|show|open|browse|browser|brows|get|view|display|see|check)\s+` +
				`(?:(?:the\s+)?(?:folders?|files?|directory|dir|contents?|items?)\s+(?:in|of|at|from|for)\s+)?` +
				`|` +
				// "folders in X" / "files in X"
				`(?:folders?|files?|directory|dir)\s+(?:in|of|at)\s+` +
				`|` +
				// Arabic variations
				`(?:تصفح|استعرض|اعرض|عرض|افتح|شوف|وريني)\s+(?:(?:المجلدات|الملفات|محتوى|قائمة)\s+(?:في|بـ|من)\s+)?` +
				`|` +
				// "قائمة المجلدات في X"
				`قائمة\s+(?:المجلدات|الملفات)\s+(?:في|بـ|من)\s+` +
				`)(.+?)$`),
			Build: func(m []string) []aimodel.ToolCall {
				path := strings.TrimSpace(m[1])
				return []aimodel.ToolCall{intents.ToolCall("browse_folder", map[string]string{"path": path})}
			},
		},
	)

	intents.RegisterSimple(
		// ── Screenshot ───────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:take\s+(?:a\s+)?screenshot|screenshot|snap(?:shot)?|لقطة(?:\s*شاشة)?|خذ\s+لقطة(?:\s+شاشة)?|التقط\s+(?:لقطة|شاشة|صورة)|ارسل\s+(?:لقطة|سكرين|صورة)|سكرين\s*شوت|سكرين)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("take_screenshot", nil)}
			},
		},

		// ── Clipboard: paste / get ───────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:paste|الصق|اعرض\s+الحافظة|clipboard)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("get_clipboard", nil)}
			},
		},

		// ── Clipboard: copy <text> ───────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:copy|انسخ)\s+(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("set_clipboard", map[string]string{"text": strings.TrimSpace(m[1])})}
			},
		},

		// ── Send / submit ────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:send|submit|ارسل|أرسل|اضغط\s+(?:ارسال|enter|return|إدخال))\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("press_key", map[string]string{"key": "return"})}
			},
		},

		// ── Cancel (ctrl+c) ──────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:cancel|stop|abort|الغ|الغاء|اوقف|ctrl\+c)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("press_key", map[string]string{"key": "ctrl+c"})}
			},
		},

		// ── Press key ────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:press|hit|اضغط)\s+(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("press_key", map[string]string{"key": strings.TrimSpace(m[1])})}
			},
		},

		// ── Open app ─────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:open|launch|start|افتح|شغّل\s+تطبيق|شغل)\s+(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				app := intents.ResolveAppName(m[1])
				return []aimodel.ToolCall{intents.ToolCall("open_app", map[string]string{"app_name": app})}
			},
		},

		// ── Type / write ─────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:type|write|اكتب|أكتب)\s+(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				text := intents.StripQuotes(strings.TrimSpace(m[1]))
				return []aimodel.ToolCall{intents.ToolCall("type_text", map[string]string{"text": text})}
			},
		},

		// ── Click at coordinates ─────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:click|انقر|اضغط)\s+(?:at\s+)?(\d+)\s*[,\s]\s*(\d+)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("click", map[string]string{"x": m[1], "y": m[2]})}
			},
		},

		// ── Click UI element by label ────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:click|press\s+button|انقر|اضغط\s+(?:على\s+)?زر)\s+(?:on\s+)?(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				label := intents.StripQuotes(strings.TrimSpace(m[1]))
				return []aimodel.ToolCall{intents.ToolCall("click_ui_element", map[string]string{"app_name": "", "label": label})}
			},
		},

		// ── Scroll ───────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:scroll|مرر)\s+(up|down|left|right|فوق|تحت|يمين|يسار)(?:\s+(\d+))?\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
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
				return []aimodel.ToolCall{intents.ToolCall("scroll", map[string]string{"direction": dir, "steps": steps})}
			},
		},

		// ── Wait ─────────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:wait|sleep|انتظر)\s+(\d+)(?:\s*(?:ms|milliseconds|ملي|ثانية))?\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("wait", map[string]string{"ms": m[1]})}
			},
		},

		// ── List recipes / help ──────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:list\s+recipes|recipes|الوصفات|اعرض\s+الوصفات|help|مساعدة|اوامر|الاوامر)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("list_recipes", nil)}
			},
		},

		// ── Run recipe ───────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:recipe|run\s+recipe|وصفة|نفذ\s+وصفة)\s+(\S+)(?:\s+(.+))?\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				name := strings.TrimSpace(m[1])
				args := map[string]string{"name": name}
				if m[2] != "" {
					args["params_json"] = m[2]
				}
				return []aimodel.ToolCall{intents.ToolCall("run_recipe", args)}
			},
		},

		// ── Read file ────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:read|show|cat|اقرأ|اعرض\s+محتوى)\s+(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("read_file", map[string]string{"path": strings.TrimSpace(m[1])})}
			},
		},

		// ── List directory ───────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:list|ls|dir|اعرض|محتوى\s+مجلد)\s+(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("list_directory", map[string]string{"path": strings.TrimSpace(m[1])})}
			},
		},

		// ── Run shell ────────────────────────────────────────────────
		intents.Intent{
			Re: regexp.MustCompile(`(?i)^(?:run|shell|exec|execute|شغل|نفذ)\s+(.+?)\.?$`),
			Build: func(m []string) []aimodel.ToolCall {
				return []aimodel.ToolCall{intents.ToolCall("run_shell", map[string]string{"command": strings.TrimSpace(m[1])})}
			},
		},
	)
}
