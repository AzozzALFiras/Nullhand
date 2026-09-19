package safety

import (
	"errors"
	"testing"
	"time"
)

func TestGuardAllowsListedUsers(t *testing.T) {
	g := New(100, 200, 300)
	for _, id := range []int64{100, 200, 300} {
		if !g.IsAllowed(id) {
			t.Errorf("user %d should be allowed", id)
		}
	}
	if g.IsAllowed(999) {
		t.Error("unlisted user should be rejected")
	}
}

func TestGuardIgnoresZeroID(t *testing.T) {
	g := New(0, 100)
	if g.IsAllowed(0) {
		t.Error("zero ID must never be allowed (default-zero config trap)")
	}
	if !g.IsAllowed(100) {
		t.Error("real ID alongside zero should still pass")
	}
}

func TestGuardRejectsAllWhenEmpty(t *testing.T) {
	g := New()
	if g.IsAllowed(123) {
		t.Error("empty allowlist must reject everyone")
	}
}

// fakeClock lets tests move time forward without sleeping.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newGuardWithClock() (*Guard, *fakeClock) {
	clock := &fakeClock{t: time.Date(2026, 9, 19, 9, 0, 0, 0, time.UTC)}
	g := New(100)
	g.now = clock.now
	return g, clock
}

func TestConfirmPendingRunsActionOnce(t *testing.T) {
	g, _ := newGuardWithClock()
	runs := 0
	g.SetPending(1, func() (string, error) { runs++; return "done", nil })

	result, had, err := g.ConfirmPending(1)
	if !had || err != nil || result != "done" {
		t.Fatalf("first confirm should run the action: result=%q had=%v err=%v", result, had, err)
	}
	if _, had, _ := g.ConfirmPending(1); had {
		t.Error("a confirmed action must not be runnable twice")
	}
	if runs != 1 {
		t.Errorf("action ran %d times, want 1", runs)
	}
}

func TestConfirmPendingPropagatesError(t *testing.T) {
	g, _ := newGuardWithClock()
	g.SetPending(1, func() (string, error) { return "", errors.New("boom") })
	if _, had, err := g.ConfirmPending(1); !had || err == nil {
		t.Errorf("action error must be returned: had=%v err=%v", had, err)
	}
}

func TestPendingIsScopedPerChat(t *testing.T) {
	g, _ := newGuardWithClock()
	ran := false
	g.SetPending(1, func() (string, error) { ran = true; return "", nil })

	if _, had, _ := g.ConfirmPending(2); had {
		t.Fatal("chat 2 must not be able to confirm chat 1's action")
	}
	if ran {
		t.Fatal("action ran from the wrong chat")
	}
	if g.ClearPending(2) {
		t.Error("chat 2 has nothing to cancel")
	}
	if !g.HasPending(1) {
		t.Error("chat 1's action must survive chat 2's /yes and /no")
	}
}

func TestPendingExpires(t *testing.T) {
	g, clock := newGuardWithClock()
	ran := false
	g.SetPending(1, func() (string, error) { ran = true; return "", nil })

	clock.advance(PendingTTL - time.Second)
	if !g.HasPending(1) {
		t.Fatal("action should still be pending just before the TTL")
	}
	clock.advance(time.Second)
	if g.HasPending(1) {
		t.Error("action should expire exactly at the TTL")
	}
	if _, had, _ := g.ConfirmPending(1); had || ran {
		t.Error("an expired action must never run")
	}
}

func TestClearPendingDiscardsAction(t *testing.T) {
	g, _ := newGuardWithClock()
	ran := false
	g.SetPending(1, func() (string, error) { ran = true; return "", nil })

	if !g.ClearPending(1) {
		t.Fatal("ClearPending should report the discarded action")
	}
	if _, had, _ := g.ConfirmPending(1); had || ran {
		t.Error("a cancelled action must never run")
	}
	if g.ClearPending(1) {
		t.Error("second ClearPending has nothing to discard")
	}
}

func TestSetPendingReplacesOlderAction(t *testing.T) {
	g, _ := newGuardWithClock()
	g.SetPending(1, func() (string, error) { return "old", nil })
	g.SetPending(1, func() (string, error) { return "new", nil })
	if result, _, _ := g.ConfirmPending(1); result != "new" {
		t.Errorf("/yes should confirm the most recent request, got %q", result)
	}
}

// The action must run without the guard's lock held; otherwise a long
// command would block every other chat's /yes and /no.
func TestConfirmPendingRunsOutsideLock(t *testing.T) {
	g, _ := newGuardWithClock()
	g.SetPending(1, func() (string, error) {
		g.HasPending(2) // would deadlock if the lock were held
		return "ok", nil
	})

	done := make(chan struct{})
	go func() {
		g.ConfirmPending(1)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ConfirmPending deadlocked: action ran while holding the lock")
	}
}
