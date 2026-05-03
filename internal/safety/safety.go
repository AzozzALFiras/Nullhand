package safety

import "sync"

// Guard enforces the allowed-user whitelist and tracks pending confirmations.
type Guard struct {
	allowed map[int64]struct{}

	mu      sync.Mutex
	pending *pendingAction
}

type pendingAction struct {
	chatID  int64
	execute func() (string, error)
}

// New creates a Guard that allows the given Telegram user IDs. Zero-value
// IDs are ignored. If no valid IDs are provided, the guard rejects every
// sender — call NewMulti(...) explicitly with the IDs you want.
func New(allowedUserIDs ...int64) *Guard {
	g := &Guard{allowed: make(map[int64]struct{})}
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

// SetPending stores a dangerous action awaiting /yes confirmation.
func (g *Guard) SetPending(chatID int64, execute func() (string, error)) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pending = &pendingAction{chatID: chatID, execute: execute}
}

// ClearPending discards any pending action without executing it.
func (g *Guard) ClearPending() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pending = nil
}

// ConfirmPending executes the stored action and clears it.
// Returns ("", false) if there is nothing pending.
func (g *Guard) ConfirmPending() (string, bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.pending == nil {
		return "", false, nil
	}
	exec := g.pending.execute
	g.pending = nil
	result, err := exec()
	return result, true, err
}

// HasPending reports whether a confirmation is waiting.
func (g *Guard) HasPending() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.pending != nil
}
