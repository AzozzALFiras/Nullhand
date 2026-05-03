package auth

import (
	"regexp"
	"sync"
	"testing"
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
