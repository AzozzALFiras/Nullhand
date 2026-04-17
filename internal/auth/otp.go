package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// OTPGate manages a one-time password that must be entered before the bot
// responds to any commands. The code is stored in memory only and expires
// after 2 minutes.
type OTPGate struct {
	mu            sync.Mutex
	code          string
	unlocked      bool
	expiryTimer   *time.Timer
	onCodeChanged func(string) // called when a new code is generated
}

// NewOTPGate creates a new OTP gate. The caller must call StartExpiry
// after wiring the onCodeChanged callback.
func NewOTPGate() *OTPGate {
	g := &OTPGate{}
	g.generateCode()
	return g
}

// generateCode creates a new cryptographically random 6-digit code.
func (g *OTPGate) generateCode() {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	code := fmt.Sprintf("%06d", n.Int64()+100000)
	g.code = code
}

// CurrentCode returns the current OTP (for debugging only).
func (g *OTPGate) CurrentCode() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.code
}

// IsUnlocked returns whether the session has been unlocked with the correct OTP.
func (g *OTPGate) IsUnlocked() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.unlocked
}

// TryUnlock checks if the supplied input matches the current code. If it does,
// the session is unlocked permanently.
func (g *OTPGate) TryUnlock(input string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.unlocked {
		return true
	}
	if input == g.code {
		g.unlocked = true
		if g.expiryTimer != nil {
			g.expiryTimer.Stop()
		}
		return true
	}
	return false
}

// StartExpiry starts the 2-minute countdown. When the timer fires, a new code
// is generated and onCodeChanged is called with the new code so the caller
// can print it.
func (g *OTPGate) StartExpiry(onCodeChanged func(newCode string)) {
	g.mu.Lock()
	g.onCodeChanged = onCodeChanged
	g.mu.Unlock()

	g.scheduleExpiry()
}

func (g *OTPGate) scheduleExpiry() {
	g.mu.Lock()
	if g.expiryTimer != nil {
		g.expiryTimer.Stop()
	}
	g.expiryTimer = time.AfterFunc(2*time.Minute, g.onExpiry)
	g.mu.Unlock()
}

func (g *OTPGate) onExpiry() {
	g.mu.Lock()
	g.generateCode()
	newCode := g.code
	onChanged := g.onCodeChanged
	g.mu.Unlock()

	if onChanged != nil {
		onChanged(newCode)
	}
	g.scheduleExpiry()
}

// PrintCurrentCode prints the current OTP in a large visible box to stdout.
// Call this when a new code is generated.
func (g *OTPGate) PrintCurrentCode() {
	g.mu.Lock()
	code := g.code
	g.mu.Unlock()

	fmt.Println("\n╔══════════════════════════════╗")
	fmt.Printf("║  OTP CODE: %s          ║\n", code)
	fmt.Println("║  Expires in 2 minutes        ║")
	fmt.Println("╚══════════════════════════════╝\n")
	fmt.Println("Enter this code in Telegram to unlock the bot.")
}

// Lock re-locks the session, generates a new OTP, prints it, and restarts the expiry timer.
func (g *OTPGate) Lock() {
	g.mu.Lock()
	g.unlocked = false
	g.generateCode()
	newCode := g.code
	onChanged := g.onCodeChanged
	g.mu.Unlock()

	if onChanged != nil {
		onChanged(newCode)
	}
	g.scheduleExpiry()
}
