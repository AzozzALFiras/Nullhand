//go:build darwin

package mac

import (
	"errors"
	"fmt"
	"time"
)

// Volume returns the system output volume and mute state.
func Volume() (VolumeState, error) {
	out, err := osa("get volume settings")
	if err != nil {
		return VolumeState{}, err
	}
	return parseVolumeSettings(out)
}

// SetVolume sets the output volume (0-100, clamped) and returns the new state.
func SetVolume(percent int) (VolumeState, error) {
	p := clampPercent(percent)
	if _, err := osa(fmt.Sprintf("set volume output volume %d", p)); err != nil {
		return VolumeState{}, err
	}
	// Raising the level while muted leaves the Mac silent, which looks like
	// the command did nothing — so asking for a level also unmutes.
	if p > 0 {
		_, _ = osa("set volume without output muted")
	}
	return Volume()
}

// AdjustVolume moves the volume by delta percentage points.
func AdjustVolume(delta int) (VolumeState, error) {
	current, err := Volume()
	if err != nil {
		return VolumeState{}, err
	}
	return SetVolume(current.Output + delta)
}

// SetMuted mutes or unmutes the system output.
func SetMuted(muted bool) (VolumeState, error) {
	script := "set volume without output muted"
	if muted {
		script = "set volume with output muted"
	}
	if _, err := osa(script); err != nil {
		return VolumeState{}, err
	}
	return Volume()
}

// mediaApps are tried in this order; whichever is already running handles the
// command, so a user with both apps installed controls the one in use.
var mediaApps = []string{"Spotify", "Music"}

// mediaLaunchWait is how long to wait for a freshly launched Music to appear.
const mediaLaunchWait = 8 * time.Second

// waitForMediaApp polls until a media app is running or the deadline passes.
func waitForMediaApp(limit time.Duration) (string, error) {
	deadline := time.Now().Add(limit)
	for {
		if app, err := runningMediaApp(); err == nil {
			// The app's process exists before it can answer AppleScript.
			time.Sleep(700 * time.Millisecond)
			return app, nil
		}
		if time.Now().After(deadline) {
			return "", ErrNoMediaApp
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func runningMediaApp() (string, error) {
	for _, app := range mediaApps {
		if appRunning(app) {
			return app, nil
		}
	}
	return "", ErrNoMediaApp
}

// Media sends a transport command (play, pause, next, previous) to whichever
// media app is running and returns that app's name.
//
// A play request with nothing running launches Music first: "play music" from
// a phone should start the music, not report that no player is open. The other
// commands still need a running app — there is nothing to skip or pause.
func Media(action string) (string, error) {
	command, err := normalizeMediaAction(action)
	if err != nil {
		return "", err
	}
	app, err := runningMediaApp()
	if errors.Is(err, ErrNoMediaApp) && isPlayCommand(command) {
		if _, openErr := run(defaultTimeout, "open", "-a", "Music"); openErr != nil {
			return "", err // report the original "nothing is running"
		}
		app, err = waitForMediaApp(mediaLaunchWait)
	}
	if err != nil {
		return "", err
	}
	if _, err := osa(fmt.Sprintf("tell application %s to %s", QuoteAppleScript(app), command)); err != nil {
		return app, err
	}
	return app, nil
}

// CurrentTrack reports what the running media app is playing. Music and
// Spotify share enough of their AppleScript dictionary for one script to
// cover both; a stopped player has no current track, hence the try block.
func CurrentTrack() (NowPlaying, error) {
	app, err := runningMediaApp()
	if err != nil {
		return NowPlaying{}, err
	}

	// The fields come back joined by ASCII 31, which no track title contains.
	script := fmt.Sprintf(`tell application %s
	set playerState to player state as text
	try
		set theTrack to current track
		return playerState & (ASCII character 31) & (name of theTrack) & (ASCII character 31) & (artist of theTrack) & (ASCII character 31) & (album of theTrack)
	on error
		return playerState & (ASCII character 31) & ""
	end try
end tell`, QuoteAppleScript(app))

	out, err := osa(script)
	if err != nil {
		return NowPlaying{}, err
	}
	return parseTrack(app, out)
}
