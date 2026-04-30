//go:build darwin

package apps

import "strings"

// nativeAXAppsMac is the allowlist of macOS applications whose UI is exposed
// to the macOS Accessibility API (i.e. native Cocoa apps). For these apps,
// accessibility.FocusField / Click via AppleScript "System Events" works.
//
// Apps NOT in this list are typically Electron-based (VS Code, Slack, Discord,
// WhatsApp, Notion) — the agent should reach them via focus_via_palette,
// run_recipe, or OCR-based click_text instead.
var nativeAXAppsMac = map[string]struct{}{
	"safari":             {},
	"finder":             {},
	"mail":               {},
	"messages":           {},
	"facetime":           {},
	"calendar":           {},
	"contacts":           {},
	"notes":              {},
	"reminders":          {},
	"music":              {},
	"podcasts":           {},
	"tv":                 {},
	"photos":             {},
	"preview":            {},
	"textedit":           {},
	"calculator":         {},
	"system settings":    {},
	"system preferences": {},
	"terminal":           {},
	"app store":          {},
	"xcode":              {},
	"keychain access":    {},
	"font book":          {},
	"chess":              {},
	"stocks":             {},
	"weather":            {},
	"home":               {},
	"news":               {},
	"voice memos":        {},
	"books":              {},
	"maps":               {},
}

// IsNativeAX reports whether the given app name is in the native macOS AX
// allowlist. Comparison is case-insensitive and tolerates a trailing .app.
func IsNativeAX(appName string) bool {
	name := strings.ToLower(strings.TrimSpace(appName))
	name = strings.TrimSuffix(name, ".app")
	_, ok := nativeAXAppsMac[name]
	return ok
}
