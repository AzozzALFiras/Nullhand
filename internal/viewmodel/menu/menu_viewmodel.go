package menu

import (
	"sync"
	"time"

	tgsvc "github.com/AzozzALFiras/nullhand/internal/service/telegram"
)

const menuTimeout = 5 * time.Minute

// TelegramSender abstracts the Telegram client methods needed by menus.
type TelegramSender interface {
	SendMessageWithKeyboard(chatID int64, text string, keyboard *tgsvc.InlineKeyboardMarkup) (int, error)
	EditMessage(chatID int64, messageID int, text string, keyboard *tgsvc.InlineKeyboardMarkup) error
	AnswerCallbackQuery(callbackID string, text string) error
	SendMessage(chatID int64, text string) error
}

// State tracks where a user is in a menu flow.
type State struct {
	Type        string    // "file_browser", "app_chooser", "git_actions", "vscode_actions"
	CurrentPath string    // current directory path
	History     []string  // navigation stack for back button
	MessageID   int       // telegram message ID to edit
	ExpiresAt   time.Time // auto-cleanup
}

// ViewModel manages interactive menus per chat.
type ViewModel struct {
	mu     sync.Mutex
	states map[int64]*State // chatID → menu state
}

// New creates a menu ViewModel.
func New() *ViewModel {
	return &ViewModel{
		states: make(map[int64]*State),
	}
}

// GetState returns the current menu state for a chat, or nil.
func (vm *ViewModel) GetState(chatID int64) *State {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	s, ok := vm.states[chatID]
	if !ok {
		return nil
	}
	if time.Now().After(s.ExpiresAt) {
		delete(vm.states, chatID)
		return nil
	}
	return s
}

// SetState stores a menu state for a chat.
func (vm *ViewModel) SetState(chatID int64, state *State) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	state.ExpiresAt = time.Now().Add(menuTimeout)
	vm.states[chatID] = state
}

// ClearState removes the menu state for a chat.
func (vm *ViewModel) ClearState(chatID int64) {
	vm.mu.Lock()
	defer vm.mu.Unlock()
	delete(vm.states, chatID)
}

// IsActive returns true if a menu is active for this chat.
func (vm *ViewModel) IsActive(chatID int64) bool {
	return vm.GetState(chatID) != nil
}
