//go:build darwin

package mac

import (
	"fmt"
	"os/exec"
	"strings"
)

// Notify shows a macOS notification banner. An empty title becomes "Nullhand".
//
// This needs notification permission for whatever runs the bot (Terminal,
// iTerm, the binary itself); macOS asks once, and osascript reports an error
// if it was denied.
func Notify(title, body string) error {
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("notification text is empty")
	}
	if strings.TrimSpace(title) == "" {
		title = "Nullhand"
	}
	_, err := osa(fmt.Sprintf("display notification %s with title %s",
		QuoteAppleScript(body), QuoteAppleScript(title)))
	return err
}

// Say speaks text through the Mac's speakers — handy for getting someone's
// attention in the room where the Mac is.
//
// The command is started and left to run: `say` blocks until it has finished
// speaking, and the caller is the Telegram handler goroutine.
func Say(text string) error {
	spoken, err := sayText(text)
	if err != nil {
		return err
	}
	cmd := exec.Command("say", spoken)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("say: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
