package safety

import "testing"

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
