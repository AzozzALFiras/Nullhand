package safety

import "sync"

// Guard enforces the allowed-user whitelist and tracks pending confirmations.
type Guard struct {
	allowedUserID int64

	mu      sync.Mutex
	pending *pendingAction
}

type pendingAction struct {
	chatID  int64
	execute func() (string, error)
}

// New creates a Guard that only allows the given Telegram user ID.
func New(allowedUserID int64) *Guard {
	return &Guard{allowedUserID: allowedUserID}
}

// IsAllowed reports whether the sender is the authorised user.
func (g *Guard) IsAllowed(userID int64) bool {
	return userID == g.allowedUserID
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
