//go:build linux

package permissions

import (
	"fmt"

	permsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/permissions"
)

// Header prints the section header shown before capability checks.
func Header() {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  🔐 Checking Linux capabilities")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// Report prints the status of each capability.
func Report(s permsvc.Status) {
	fmt.Printf("  %s Screen Recording (for /screenshot)\n", mark(s.ScreenRecording))
	fmt.Printf("  %s Accessibility    (for /click, /type, /key)\n", mark(s.Accessibility))
	fmt.Println()
}

// Missing prints guidance when one or more capabilities are not available.
func Missing(s permsvc.Status) {
	fmt.Println("⚠️  Some capabilities are missing. Nullhand needs them to control your desktop.")
	fmt.Println()
	fmt.Println("How to grant them:")
	fmt.Println()

	if !s.ScreenRecording {
		fmt.Println("  1. Opening GNOME Privacy settings...")
		fmt.Println("     → Ensure screen sharing / screenshot permissions are enabled")
		fmt.Println()
	}
	if !s.Accessibility {
		fmt.Println("  2. Opening Universal Access settings...")
		fmt.Println("     → Ensure assistive technologies / accessibility is enabled")
		fmt.Println()
	}

	fmt.Println("After granting, fully quit and relaunch nullhand.")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

// Granted prints a success footer when all capabilities are present.
func Granted() {
	fmt.Println("✓ All capabilities granted.")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

func mark(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}
