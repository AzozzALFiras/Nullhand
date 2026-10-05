// Package mac registers the natural-language intents for the macOS-only
// features: Shortcuts, audio and media, Spotlight search, power and
// appearance, and screen alerts.
//
// Registration happens in mac_darwin.go only. On Linux the package registers
// nothing, so phrases like "lock" keep falling through to the Linux intents
// in the system package instead of resolving to a tool that cannot run there.
package mac
