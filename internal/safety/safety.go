package safety

import (
	"sync"
	"time"
)

// PendingTTL is how long a parked action waits for /yes before it is
// discarded, so a stale confirmation can never fire minutes later because the
// user typed /yes for something else.
const PendingTTL = 2 * time.Minute

// Guard enforces the allowed-user whitelist and tracks pending confirmations.
type Guard struct {
	allowed map[int64]struct{}

	mu      sync.Mutex
	pending map[int64]*pendingAction // keyed by chat ID
	now     func() time.Time
}

type pendingAction struct {
	execute   func() (string, error)
	expiresAt time.Time
}

// New creates a Guard that allows the given Telegram user IDs. Zero-value
// IDs are ignored. If no valid IDs are provided, the guard rejects every
// sender — call NewMulti(...) explicitly with the IDs you want.
func New(allowedUserIDs ...int64) *Guard {
	g := &Guard{
		allowed: make(map[int64]struct{}),
		pending: make(map[int64]*pendingAction),
		now:     time.Now,
	}
	for _, id := range allowedUserIDs {
		if id != 0 {
			g.allowed[id] = struct{}{}
		}
	}
	return g
}

// IsAllowed reports whether the sender is on the whitelist.
func (g *Guard) IsAllowed(userID int64) bool {
	_, ok := g.allowed[userID]
	return ok
}

// AllowedUserIDs returns a snapshot of the whitelist (unordered).
func (g *Guard) AllowedUserIDs() []int64 {
	out := make([]int64, 0, len(g.allowed))
	for id := range g.allowed {
		out = append(out, id)
	}
	return out
}

// SetPending parks a dangerous action for chatID until /yes confirms it or
// PendingTTL passes. A newer action replaces any older one for the same chat;
// other chats are unaffected, so one whitelisted user can never confirm an
// action another user requested.
func (g *Guard) SetPending(chatID int64, execute func() (string, error)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pending[chatID] = &pendingAction{execute: execute, expiresAt: g.now().Add(PendingTTL)}
}

// ClearPending discards the chat's pending action without executing it and
// reports whether there was a live one to discard.
func (g *Guard) ClearPending(chatID int64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.takeLocked(chatID) != nil
}

// ConfirmPending executes the chat's pending action and clears it.
// Returns ("", false, nil) if nothing is pending or it has expired.
// The action runs after the lock is released, so a slow command cannot
// block other chats' confirmations.
func (g *Guard) ConfirmPending(chatID int64) (string, bool, error) {
	g.mu.Lock()
	p := g.takeLocked(chatID)
	g.mu.Unlock()
	if p == nil {
		return "", false, nil
	}
	result, err := p.execute()
	return result, true, err
}

// HasPending reports whether a live confirmation is waiting for chatID.
func (g *Guard) HasPending(chatID int64) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	p, ok := g.pending[chatID]
	return ok && g.now().Before(p.expiresAt)
}

// takeLocked removes and returns the chat's pending action, or nil when
// there is none or it has expired. Callers must hold g.mu.
func (g *Guard) takeLocked(chatID int64) *pendingAction {
	p, ok := g.pending[chatID]
	if !ok {
		return nil
	}
	delete(g.pending, chatID)
	if !g.now().Before(p.expiresAt) {
		return nil
	}
	return p
}
