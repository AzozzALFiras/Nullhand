package setup

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// WizardView handles CLI input/output for the first-run setup wizard.
type WizardView struct {
	reader *bufio.Reader
}

// New creates a WizardView reading from stdin.
func New() *WizardView {
	return &WizardView{reader: bufio.NewReader(os.Stdin)}
}

// PrintBanner prints the Nullhand welcome banner.
func (v *WizardView) PrintBanner() {
	fmt.Println()
	fmt.Println("  ███╗   ██╗██╗   ██╗██╗     ██╗     ██╗  ██╗ █████╗ ███╗   ██╗██████╗ ")
	fmt.Println("  ████╗  ██║██║   ██║██║     ██║     ██║  ██║██╔══██╗████╗  ██║██╔══██╗")
	fmt.Println("  ██╔██╗ ██║██║   ██║██║     ██║     ███████║███████║██╔██╗ ██║██║  ██║")
	fmt.Println("  ██║╚██╗██║██║   ██║██║     ██║     ██╔══██║██╔══██║██║╚██╗██║██║  ██║")
	fmt.Println("  ██║ ╚████║╚██████╔╝███████╗███████╗██║  ██║██║  ██║██║ ╚████║██████╔╝")
	fmt.Println("  ╚═╝  ╚═══╝ ╚═════╝ ╚══════╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═════╝ ")
	fmt.Println()
	fmt.Println("  Your invisible hand on the machine.")
	fmt.Println()
}

// PrintStep prints a numbered setup step header.
func (v *WizardView) PrintStep(n int, title string) {
	fmt.Printf("\n  [%d] %s\n", n, title)
	fmt.Println("  " + strings.Repeat("─", 50))
}

// Ask prints a prompt and reads one line of input.
func (v *WizardView) Ask(prompt string) (string, error) {
	fmt.Printf("  %s: ", prompt)
	line, err := v.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

// AskSecret prints a prompt but does not echo — the input is still read
// from stdin (no terminal magic to keep zero deps).
func (v *WizardView) AskSecret(prompt string) (string, error) {
	fmt.Printf("  %s (hidden): ", prompt)
	line, err := v.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	fmt.Println()
	return strings.TrimSpace(line), nil
}

// Choose presents a numbered menu and returns the chosen index (0-based).
func (v *WizardView) Choose(prompt string, options []string) (int, error) {
	fmt.Printf("  %s\n\n", prompt)
	for i, o := range options {
		fmt.Printf("    %d) %s\n", i+1, o)
	}
	fmt.Println()

	for {
		raw, err := v.Ask("Enter number")
		if err != nil {
			return 0, err
		}
		var n int
		if _, err := fmt.Sscanf(raw, "%d", &n); err == nil && n >= 1 && n <= len(options) {
			return n - 1, nil
		}
		fmt.Printf("  Please enter a number between 1 and %d.\n", len(options))
	}
}

// PrintSuccess prints a success message.
func (v *WizardView) PrintSuccess(msg string) {
	fmt.Printf("\n  ✓ %s\n", msg)
}

// PrintError prints an error message.
func (v *WizardView) PrintError(msg string) {
	fmt.Fprintf(os.Stderr, "\n  ✗ %s\n", msg)
}

// PrintInfo prints an informational message.
func (v *WizardView) PrintInfo(msg string) {
	fmt.Printf("  → %s\n", msg)
}

// PrintDone prints the final ready message.
func (v *WizardView) PrintDone(botUsername string) {
	fmt.Println()
	fmt.Println("  ─────────────────────────────────────────────")
	fmt.Println("  Nullhand is ready.")
	fmt.Println()
	if botUsername != "" {
		fmt.Printf("  Open Telegram and message @%s to get started.\n", botUsername)
	}
	fmt.Println("  Tip: send /screenshot to see your screen.")
	fmt.Println("  ─────────────────────────────────────────────")
	fmt.Println()
}
