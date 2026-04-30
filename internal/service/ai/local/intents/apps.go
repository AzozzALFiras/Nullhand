package intents

import "strings"

// AppNameMap maps common names (any language) to the correct macOS app name.
// Exported so the entity extractor in the parent package can use it.
var AppNameMap = map[string]string{
	// English
	"terminal": "Terminal", "term": "Terminal", "iterm": "iTerm",
	"safari": "Safari", "chrome": "Google Chrome", "google chrome": "Google Chrome",
	"firefox": "Firefox", "brave": "Brave Browser", "arc": "Arc",
	"vscode": "Visual Studio Code", "vs code": "Visual Studio Code", "code": "Visual Studio Code",
	"cursor": "Cursor",
	"antigravity": "Antigravity", "anti gravity": "Antigravity", "ag": "Antigravity",
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
	"app store": "App Store",
	// Arabic
	"التيرمنل": "Terminal", "الطرفية": "Terminal", "تيرمنل": "Terminal", "الترمنل": "Terminal",
	"سفاري": "Safari", "كروم": "Google Chrome", "قوقل كروم": "Google Chrome", "جوجل كروم": "Google Chrome",
	"فايرفوكس": "Firefox", "فاير فوكس": "Firefox", "فيرفوكس": "Firefox",
	"المتصفح": "Firefox", "متصفح": "Firefox",
	"واتساب": "WhatsApp", "واتس": "WhatsApp", "الواتساب": "WhatsApp", "الواتس": "WhatsApp",
	"سلاك": "Slack", "ديسكورد": "Discord", "دسكورد": "Discord",
	"تلقرام": "Telegram", "تليقرام": "Telegram", "تليجرام": "Telegram",
	"الرسائل": "Messages", "رسائل": "Messages",
	"فايندر": "Finder", "الملفات": "Finder",
	"الملاحظات": "Notes", "ملاحظات": "Notes",
	"البريد": "Mail", "بريد": "Mail",
	"الاعدادات": "System Settings", "الضبط": "System Settings", "اعدادات": "System Settings",
	"الموسيقى": "Music", "موسيقى": "Music",
	"الصور": "Photos", "صور": "Photos",
	"الحاسبة": "Calculator", "حاسبة": "Calculator",
	"المتجر": "App Store",
}

// ResolveAppName normalizes an app name using the map, or returns it as-is.
func ResolveAppName(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	if mapped, ok := AppNameMap[lower]; ok {
		return mapped
	}
	return strings.TrimSpace(raw)
}
