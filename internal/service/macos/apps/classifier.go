package apps

import "strings"

// nativeAXApps is the allowlist of Linux applications whose UI is exposed
// to the Accessibility tree via AT-SPI2. For these apps,
// accessibility.FocusField / Click work correctly.
//
// Apps NOT in this list are assumed to be Electron-based or otherwise
// opaque to AT-SPI, and the agent should reach them via focus_via_palette or
// run_recipe instead.
//
// Keys are lowercased process/display names. We check both the user-supplied
// name and common variants.
var nativeAXApps = map[string]struct{}{
	"firefox":          {},
	"thunderbird":      {},
	"gedit":            {},
	"gnome-terminal":   {},
	"terminal":         {},
	"nautilus":         {},
	"files":            {},
	"gnome-text-editor": {},
	"kate":             {},
	"konsole":          {},
	"dolphin":          {},
	"libreoffice":      {},
	"libreoffice writer": {},
	"libreoffice calc": {},
	"libreoffice impress": {},
	"evince":           {},
	"eog":              {},
	"cheese":           {},
	"gnome-calculator": {},
	"calculator":       {},
	"gnome-calendar":   {},
	"calendar":         {},
	"gnome-contacts":   {},
	"contacts":         {},
	"rhythmbox":        {},
	"totem":            {},
	"shotwell":         {},
	"gnome-maps":       {},
	"gnome-weather":    {},
	"gnome-disk-utility": {},
	"baobab":           {},
	"gnome-system-monitor": {},
	"system monitor":   {},
	"seahorse":         {},
}

// IsNativeAX reports whether the given app name is in the native AX allowlist.
// The comparison is case-insensitive and tolerant of the .app suffix.
func IsNativeAX(appName string) bool {
	name := strings.ToLower(strings.TrimSpace(appName))
	name = strings.TrimSuffix(name, ".app")
	_, ok := nativeAXApps[name]
	return ok
}
