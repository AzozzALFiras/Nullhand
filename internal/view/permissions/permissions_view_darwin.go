//go:build darwin

package permissions

import (
	"fmt"

	permsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/permissions"
)

// Header prints the section header shown before capability checks.
func Header() {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  🔐 Checking macOS capabilities")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// Report prints the status of each capability.
func Report(s permsvc.Status) {
	fmt.Printf("  %s Screen Recording (for /screenshot, OCR, click_text)\n", mark(s.ScreenRecording))
	fmt.Printf("  %s Accessibility    (for /click, /type, /key, AT-SPI lookups)\n", mark(s.Accessibility))
	fmt.Println()
}

// Missing prints guidance when one or more macOS capabilities are not granted.
func Missing(s permsvc.Status) {
	fmt.Println("⚠️  Some macOS capabilities are missing. Nullhand needs them to control your desktop.")
	fmt.Println()
	fmt.Println("How to grant them:")
	fmt.Println()

	if !s.ScreenRecording {
		fmt.Println("  1. Opening System Settings → Privacy & Security → Screen Recording…")
		fmt.Println("     → Toggle ON the entry for the process running nullhand")
		fmt.Println("       (typically Terminal.app, iTerm.app, or the nullhand binary itself)")
		fmt.Println()
	}
	if !s.Accessibility {
		fmt.Println("  2. Opening System Settings → Privacy & Security → Accessibility…")
		fmt.Println("     → Toggle ON the entry for the process running nullhand")
		fmt.Println()
	}

	fmt.Println("After granting, fully quit and relaunch nullhand.")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

// Granted prints a success footer when all capabilities are present.
func Granted() {
	fmt.Println("✅ All macOS capabilities granted.")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

func mark(ok bool) string {
	if ok {
		return "✅"
	}
	return "❌"
}
