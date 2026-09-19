package auth

import (
	"regexp"
	"sync"
	"testing"
	"time"
)

func TestNewGateIsLocked(t *testing.T) {
	g := NewOTPGate()
	if g.IsUnlocked() {
		t.Fatal("freshly created gate must start locked")
	}
	if g.CurrentCode() == "" {
		t.Fatal("freshly created gate must have a code")
	}
}

func TestCodeFormat(t *testing.T) {
	g := NewOTPGate()
	re := regexp.MustCompile(`^\d{6}$`)
	if !re.MatchString(g.CurrentCode()) {
		t.Fatalf("code %q is not 6 digits", g.CurrentCode())
	}
}

func TestTryUnlockCorrect(t *testing.T) {
	g := NewOTPGate()
	code := g.CurrentCode()
	if !g.TryUnlock(code) {
		t.Fatal("correct code must unlock")
	}
	if !g.IsUnlocked() {
		t.Fatal("gate must report unlocked after correct code")
	}
}

func TestTryUnlockWrong(t *testing.T) {
	g := NewOTPGate()
	if g.TryUnlock("000000") {
		// 000000 is the only 6-digit value that cannot be produced by
		// generateCode (which adds 100000), so it is guaranteed wrong.
		t.Fatal("wrong code must not unlock")
	}
	if g.IsUnlocked() {
		t.Fatal("gate must remain locked after wrong attempt")
	}
}

func TestTryUnlockIdempotentAfterUnlock(t *testing.T) {
	g := NewOTPGate()
	_ = g.TryUnlock(g.CurrentCode())
	// After unlocking, any input should keep the session unlocked: callers
	// rely on this so a stale OTP message doesn't accidentally re-lock.
	if !g.TryUnlock("not-the-code") {
		t.Fatal("once unlocked, TryUnlock must keep returning true")
	}
}

func TestLockResets(t *testing.T) {
	g := NewOTPGate()
	original := g.CurrentCode()
	_ = g.TryUnlock(original)

	g.Lock()
	if g.IsUnlocked() {
		t.Fatal("Lock must re-lock the session")
	}
	if g.CurrentCode() == original {
		t.Fatal("Lock must rotate the code")
	}
	if !g.TryUnlock(g.CurrentCode()) {
		t.Fatal("the new code must unlock")
	}
}

func TestConcurrentTryUnlock(t *testing.T) {
	// Race detector check: many goroutines hitting the gate concurrently
	// must never panic and at most one of them sees the "first unlock"
	// transition (the others see already-unlocked, both accepted).
	g := NewOTPGate()
	code := g.CurrentCode()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			g.TryUnlock(code)
			_ = g.IsUnlocked()
			_ = g.CurrentCode()
		}()
	}
	wg.Wait()
	if !g.IsUnlocked() {
		t.Fatal("gate must end unlocked after concurrent correct attempts")
	}
}

// unlockedGateWithClock returns an unlocked gate whose clock the test controls.
func unlockedGateWithClock(t *testing.T, idle time.Duration) (*OTPGate, *time.Time) {
	t.Helper()
	clock := time.Date(2026, 9, 19, 9, 0, 0, 0, time.UTC)
	g := NewOTPGate()
	g.now = func() time.Time { return clock }
	g.SetIdleTimeout(idle)
	if !g.TryUnlock(g.CurrentCode()) {
		t.Fatal("setup: unlock failed")
	}
	return g, &clock
}

func TestIdleLockDisabledWhenTimeoutIsZero(t *testing.T) {
	g, clock := unlockedGateWithClock(t, 0)
	*clock = clock.Add(365 * 24 * time.Hour)
	if g.LockIfIdle() || !g.IsUnlocked() {
		t.Fatal("with no idle timeout the session must stay unlocked")
	}
}

func TestIdleLockAfterInactivity(t *testing.T) {
	g, clock := unlockedGateWithClock(t, 30*time.Minute)
	oldCode := g.CurrentCode()

	*clock = clock.Add(30*time.Minute - time.Second)
	if g.LockIfIdle() {
		t.Fatal("must not lock before the idle timeout")
	}
	*clock = clock.Add(time.Second)
	if !g.LockIfIdle() {
		t.Fatal("must lock once the idle timeout is reached")
	}
	if g.IsUnlocked() {
		t.Error("gate must be locked after an idle lock")
	}
	if g.CurrentCode() == oldCode {
		t.Error("idle lock must rotate the code so the old one is useless")
	}
}

func TestTouchRestartsIdleCountdown(t *testing.T) {
	g, clock := unlockedGateWithClock(t, 30*time.Minute)

	*clock = clock.Add(20 * time.Minute)
	g.Touch()
	*clock = clock.Add(20 * time.Minute) // 40 min since unlock, 20 since last activity
	if g.LockIfIdle() {
		t.Fatal("activity must restart the idle countdown")
	}
	*clock = clock.Add(10 * time.Minute)
	if !g.LockIfIdle() {
		t.Fatal("30 min after the last activity the session must lock")
	}
}

func TestLockIfIdleReportsOnlyOnce(t *testing.T) {
	g, clock := unlockedGateWithClock(t, time.Minute)
	*clock = clock.Add(time.Hour)
	if !g.LockIfIdle() {
		t.Fatal("first call must perform the lock")
	}
	if g.LockIfIdle() {
		t.Error("an already-locked gate must not report a second idle lock")
	}
}

func TestIdleLockAnnouncesNewCode(t *testing.T) {
	g, clock := unlockedGateWithClock(t, time.Minute)
	var announced string
	g.onCodeChanged = func(code string) { announced = code }

	*clock = clock.Add(time.Hour)
	g.LockIfIdle()
	if announced == "" || announced != g.CurrentCode() {
		t.Errorf("new code must be announced so it gets printed to the terminal, got %q want %q", announced, g.CurrentCode())
	}
}

func TestSetIdleTimeoutClampsNegative(t *testing.T) {
	g := NewOTPGate()
	g.SetIdleTimeout(-5 * time.Minute)
	if g.IdleTimeout() != 0 {
		t.Errorf("negative timeout must disable idle locking, got %s", g.IdleTimeout())
	}
}
