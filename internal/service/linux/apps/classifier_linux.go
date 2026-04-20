//go:build linux

package apps

import "strings"

// nativeAXApps is the allowlist of Linux applications whose UI is exposed
// to the accessibility tree via AT-SPI2. For these apps,
// accessibility.FocusField / Click work correctly.
//
// Apps NOT in this list are assumed to be Electron-based or otherwise
// opaque to AT-SPI, and the agent should reach them via focus_via_palette or
// run_recipe instead.
//
// Keys are lowercased process/display names.
var nativeAXApps = map[string]struct{}{
	// Browsers (non-snap .deb builds expose AT-SPI)
	"firefox":   {},
	"epiphany":  {},
	// Mail / comms
	"thunderbird": {},
	"geary":       {},
	// Editors / IDEs
	"gedit":             {},
	"gnome-text-editor": {},
	"text editor":       {},
	"kate":              {},
	"mousepad":          {},
	// Terminal emulators
	"gnome-terminal": {},
	"terminal":       {},
	"konsole":        {},
	"xterm":          {},
	"tilix":          {},
	// File managers
	"nautilus": {},
	"files":    {},
	"dolphin":  {},
	"thunar":   {},
	// Office / productivity
	"libreoffice":          {},
	"libreoffice writer":   {},
	"libreoffice calc":     {},
	"libreoffice impress":  {},
	// Viewers
	"evince":  {},
	"eog":     {},
	"totem":   {},
	"shotwell": {},
	// System / utilities
	"gnome-calculator":     {},
	"calculator":           {},
	"gnome-calendar":       {},
	"calendar":             {},
	"gnome-contacts":       {},
	"contacts":             {},
	"rhythmbox":            {},
	"gnome-maps":           {},
	"gnome-weather":        {},
	"gnome-disk-utility":   {},
	"baobab":               {},
	"gnome-system-monitor": {},
	"system monitor":       {},
	"seahorse":             {},
	"gnome-control-center": {},
	"settings":             {},
}

// IsNativeAX reports whether the given app name is in the native AX allowlist.
// The comparison is case-insensitive.
func IsNativeAX(appName string) bool {
	name := strings.ToLower(strings.TrimSpace(appName))
	name = strings.TrimSuffix(name, ".app") // tolerate accidental .app suffix
	_, ok := nativeAXApps[name]
	return ok
}
