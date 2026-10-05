//go:build !darwin

package mac

import "time"

// On non-macOS builds every entry point reports ErrUnsupported so the shared
// viewmodels can offer the command and explain it needs a Mac, rather than
// the Linux build failing to compile.

// Available reports whether this build can run the macOS features.
func Available() bool { return false }

func ListShortcuts() ([]string, error)                       { return nil, ErrUnsupported }
func RunShortcut(name, input string) (string, error)         { return "", ErrUnsupported }
func Volume() (VolumeState, error)                           { return VolumeState{}, ErrUnsupported }
func SetVolume(percent int) (VolumeState, error)             { return VolumeState{}, ErrUnsupported }
func AdjustVolume(delta int) (VolumeState, error)            { return VolumeState{}, ErrUnsupported }
func SetMuted(muted bool) (VolumeState, error)               { return VolumeState{}, ErrUnsupported }
func Media(action string) (string, error)                    { return "", ErrUnsupported }
func CurrentTrack() (NowPlaying, error)                      { return NowPlaying{}, ErrUnsupported }
func Say(text string) error                                  { return ErrUnsupported }
func Notify(title, body string) error                        { return ErrUnsupported }
func Find(query, folder string, limit int) ([]string, error) { return nil, ErrUnsupported }
func Preview(path string) ([]byte, error)                    { return nil, ErrUnsupported }
func BatteryStatus() (Battery, error)                        { return Battery{}, ErrUnsupported }
func LockScreen() error                                      { return ErrUnsupported }
func SleepNow() error                                        { return ErrUnsupported }
func KeepAwake(d time.Duration) (time.Time, error)           { return time.Time{}, ErrUnsupported }
func StopKeepAwake() bool                                    { return false }
func KeepAwakeUntil() (time.Time, bool)                      { return time.Time{}, false }
func DarkMode() (bool, error)                                { return false, ErrUnsupported }
func SetDarkMode(on bool) error                              { return ErrUnsupported }
func ToggleDarkMode() (bool, error)                          { return false, ErrUnsupported }
func HealthLines() []string                                  { return nil }
