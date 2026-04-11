package apps

import "strings"

// nativeAXApps is the allowlist of macOS applications whose UI is exposed
// to the Accessibility tree via AppleScript `System Events`. For these apps,
// accessibility.FocusField / Click work correctly.
//
// Apps NOT in this list are assumed to be Electron-based or otherwise
// opaque to AX, and the agent should reach them via focus_via_palette or
// run_recipe instead.
//
// Keys are lowercased process/display names. We check both the user-supplied
// name and common variants.
var nativeAXApps = map[string]struct{}{
	"safari":          {},
	"mail":            {},
	"messages":        {},
	"finder":          {},
	"terminal":        {},
	"iterm":           {},
	"iterm2":          {},
	"textedit":        {},
	"notes":           {},
	"xcode":           {},
	"system settings": {},
	"system preferences": {},
	"preview":         {},
	"calculator":      {},
	"reminders":       {},
	"calendar":        {},
	"contacts":        {},
	"music":           {},
	"podcasts":        {},
	"photos":          {},
	"maps":            {},
	"weather":         {},
	"dictionary":      {},
	"automator":       {},
	"script editor":   {},
	"activity monitor": {},
	"disk utility":    {},
	"keychain access": {},
}

// IsNativeAX reports whether the given app name is in the native AX allowlist.
// The comparison is case-insensitive and tolerant of the .app suffix.
func IsNativeAX(appName string) bool {
	name := strings.ToLower(strings.TrimSpace(appName))
	name = strings.TrimSuffix(name, ".app")
	_, ok := nativeAXApps[name]
	return ok
}
