package safety

import "testing"

func TestRateLimiterAllowsBurst(t *testing.T) {
	rl := NewRateLimiter(3, 60)
	for i := 0; i < 3; i++ {
		ok, _ := rl.Allow(1)
		if !ok {
			t.Fatalf("burst message %d unexpectedly denied", i+1)
		}
	}
	ok, notify := rl.Allow(1)
	if ok {
		t.Fatal("expected 4th message to be denied")
	}
	if !notify {
		t.Fatal("first denial should set notify=true")
	}
	_, notify = rl.Allow(1)
	if notify {
		t.Fatal("subsequent denials should not re-notify")
	}
}

func TestRateLimiterPerUserIsolated(t *testing.T) {
	rl := NewRateLimiter(1, 60)
	if ok, _ := rl.Allow(1); !ok {
		t.Fatal("user 1 first message denied")
	}
	if ok, _ := rl.Allow(2); !ok {
		t.Fatal("user 2 first message denied — buckets must be per-user")
	}
	if ok, _ := rl.Allow(1); ok {
		t.Fatal("user 1 second message should be denied")
	}
}

func TestRateLimiterDisabled(t *testing.T) {
	rl := NewRateLimiter(0, 0)
	for i := 0; i < 100; i++ {
		if ok, _ := rl.Allow(1); !ok {
			t.Fatalf("disabled limiter must always allow (denied at #%d)", i+1)
		}
	}
}

func TestRateLimiterNilSafe(t *testing.T) {
	var rl *RateLimiter
	if ok, _ := rl.Allow(1); !ok {
		t.Fatal("nil limiter must always allow")
	}
}
