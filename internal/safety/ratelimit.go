package safety

import (
	"sync"
	"time"
)

// RateLimiter enforces a per-user token bucket on incoming messages so a
// runaway client (or an accidental loop) cannot saturate the bot or burn AI
// tokens. The bucket refills continuously so a quiet user is never penalised
// for a previous burst.
type RateLimiter struct {
	capacity   float64
	refillRate float64 // tokens per second

	mu      sync.Mutex
	buckets map[int64]*bucket
}

type bucket struct {
	tokens     float64
	lastRefill time.Time
	notified   bool // true after we've told the user they were throttled
}

// NewRateLimiter creates a limiter that allows up to `burst` messages
// instantly and refills `perMinute` messages per minute over time. Pass
// burst=0 or perMinute=0 to disable.
func NewRateLimiter(burst, perMinute int) *RateLimiter {
	return &RateLimiter{
		capacity:   float64(burst),
		refillRate: float64(perMinute) / 60.0,
		buckets:    make(map[int64]*bucket),
	}
}

// Allow consumes one token for userID. It returns ok=true when the message
// should be processed. notify=true means this is the first denial since the
// user last had headroom — callers can use it to send a single "slow down"
// reply rather than spamming the user with repeated warnings.
func (r *RateLimiter) Allow(userID int64) (ok bool, notify bool) {
	if r == nil || r.capacity <= 0 || r.refillRate <= 0 {
		return true, false
	}

	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()

	b, found := r.buckets[userID]
	if !found {
		b = &bucket{tokens: r.capacity, lastRefill: now}
		r.buckets[userID] = b
	} else {
		elapsed := now.Sub(b.lastRefill).Seconds()
		b.tokens += elapsed * r.refillRate
		if b.tokens > r.capacity {
			b.tokens = r.capacity
		}
		b.lastRefill = now
	}

	if b.tokens >= 1 {
		b.tokens--
		b.notified = false
		return true, false
	}

	notify = !b.notified
	b.notified = true
	return false, notify
}
