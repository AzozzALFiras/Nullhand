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
// after 2 minutes. An unlocked session can optionally lock itself again after
// a period without activity (see SetIdleTimeout).
type OTPGate struct {
	mu            sync.Mutex
	code          string
	unlocked      bool
	expiryTimer   *time.Timer
	onCodeChanged func(string) // called when a new code is generated

	idleTimeout  time.Duration // 0 = never lock for inactivity
	lastActivity time.Time
	now          func() time.Time
}

// NewOTPGate creates a new OTP gate. The caller must call StartExpiry
// after wiring the onCodeChanged callback.
func NewOTPGate() *OTPGate {
	g := &OTPGate{now: time.Now}
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
		g.lastActivity = g.now()
		if g.expiryTimer != nil {
			g.expiryTimer.Stop()
		}
		return true
	}
	return false
}

// SetIdleTimeout makes an unlocked session lock itself again once d passes
// without activity. d <= 0 disables idle locking.
func (g *OTPGate) SetIdleTimeout(d time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if d < 0 {
		d = 0
	}
	g.idleTimeout = d
}

// IdleTimeout returns the configured inactivity limit (0 = disabled).
func (g *OTPGate) IdleTimeout() time.Duration {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.idleTimeout
}

// Touch records user activity, restarting the idle countdown.
func (g *OTPGate) Touch() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.lastActivity = g.now()
}

// LockIfIdle locks the session if it is unlocked and has been inactive for
// the idle timeout. It returns true only when this call performed the lock,
// so the caller can tell the user why they are locked out.
func (g *OTPGate) LockIfIdle() bool {
	g.mu.Lock()
	if !g.unlocked || g.idleTimeout <= 0 || g.now().Sub(g.lastActivity) < g.idleTimeout {
		g.mu.Unlock()
		return false
	}
	newCode, onChanged := g.relockLocked()
	g.mu.Unlock()

	g.announce(newCode, onChanged)
	return true
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
	fmt.Println("╚══════════════════════════════╝")
	fmt.Println()
	fmt.Println("Enter this code in Telegram to unlock the bot.")
}

// Lock re-locks the session, generates a new OTP, prints it, and restarts the expiry timer.
func (g *OTPGate) Lock() {
	g.mu.Lock()
	newCode, onChanged := g.relockLocked()
	g.mu.Unlock()

	g.announce(newCode, onChanged)
}

// relockLocked locks the session and rotates the code. Callers must hold g.mu
// and pass the results to announce after releasing it.
func (g *OTPGate) relockLocked() (string, func(string)) {
	g.unlocked = false
	g.generateCode()
	return g.code, g.onCodeChanged
}

// announce publishes a freshly rotated code and restarts its expiry timer.
func (g *OTPGate) announce(newCode string, onChanged func(string)) {
	if onChanged != nil {
		onChanged(newCode)
	}
	g.scheduleExpiry()
}
