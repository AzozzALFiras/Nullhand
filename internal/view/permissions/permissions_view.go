package permissions

import (
	"fmt"

	permsvc "github.com/AzozzALFiras/nullhand/internal/service/macos/permissions"
)

// Header prints the section header shown before permission checks.
func Header() {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  🔐 Checking macOS permissions")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// Report prints the status of each permission.
func Report(s permsvc.Status) {
	fmt.Printf("  %s Screen Recording (for /screenshot)\n", mark(s.ScreenRecording))
	fmt.Printf("  %s Accessibility    (for /click, /type, /key)\n", mark(s.Accessibility))
	fmt.Println()
}

// Missing prints guidance when one or more permissions are not granted.
func Missing(s permsvc.Status) {
	fmt.Println("⚠️  Some permissions are missing. Nullhand needs them to control your Mac.")
	fmt.Println()
	fmt.Println("How to grant them:")
	fmt.Println()

	if !s.ScreenRecording {
		fmt.Println("  1. Opening System Settings → Privacy → Screen Recording...")
		fmt.Println("     → Enable the entry for this Terminal / nullhand")
		fmt.Println()
	}
	if !s.Accessibility {
		fmt.Println("  2. Opening System Settings → Privacy → Accessibility...")
		fmt.Println("     → Enable the entry for this Terminal / nullhand")
		fmt.Println()
	}

	fmt.Println("After granting, fully quit and relaunch nullhand.")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

// Granted prints a success footer when all permissions are present.
func Granted() {
	fmt.Println("✓ All permissions granted.")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

func mark(ok bool) string {
	if ok {
		return "✓"
	}
	return "✗"
}
