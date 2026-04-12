package session

import (
	"sync"
	"time"
)

const contextTimeout = 10 * time.Minute

// Mode constants for active context.
const (
	ModeTerminal = "terminal" // macOS Terminal or IDE terminal
	ModeClaude   = "claude"   // Claude/AI chat in IDE
	ModeBrowser  = "browser"  // web browser active
	ModeEditor   = "editor"   // code editor active
)

// Context tracks what app and mode the user is currently working in.
type Context struct {
	ActiveApp   string    // "Visual Studio Code", "Terminal", "Safari", etc.
	ActiveMode  string    // terminal, claude, browser, editor
	WorkingPath string    // current project/directory path
	UpdatedAt   time.Time // auto-expire after timeout
}

// Manager tracks session context per chat.
type Manager struct {
	mu       sync.Mutex
	contexts map[int64]*Context // chatID → context
}

// NewManager creates a session manager.
func NewManager() *Manager {
	return &Manager{
		contexts: make(map[int64]*Context),
	}
}

// Get returns the current context for a chat, or nil if expired/missing.
func (m *Manager) Get(chatID int64) *Context {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx, ok := m.contexts[chatID]
	if !ok {
		return nil
	}
	if time.Since(ctx.UpdatedAt) > contextTimeout {
		delete(m.contexts, chatID)
		return nil
	}
	return ctx
}

// Set stores a new context for a chat.
func (m *Manager) Set(chatID int64, app, mode, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.contexts[chatID] = &Context{
		ActiveApp:   app,
		ActiveMode:  mode,
		WorkingPath: path,
		UpdatedAt:   time.Now(),
	}
}

// Touch refreshes the timestamp of the current context.
func (m *Manager) Touch(chatID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ctx, ok := m.contexts[chatID]; ok {
		ctx.UpdatedAt = time.Now()
	}
}

// Clear removes the context for a chat.
func (m *Manager) Clear(chatID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.contexts, chatID)
}

// IsActive returns true if a context is active for this chat.
func (m *Manager) IsActive(chatID int64) bool {
	return m.Get(chatID) != nil
}

// InferContextFromAction determines what context to set based on
// what the user just did. Called after tool execution.
func InferContextFromAction(toolName string, args map[string]string, currentApp string) (app, mode string) {
	switch toolName {
	case "run_recipe":
		recipeName := args["name"]
		switch {
		case recipeName == "vscode_open_terminal" || recipeName == "vscode_terminal_run" || recipeName == "vscode_new_terminal":
			return "Visual Studio Code", ModeTerminal
		case recipeName == "vscode_claude_chat_focus" || recipeName == "vscode_new_claude_chat" || recipeName == "vscode_type_in_claude":
			return "Visual Studio Code", ModeClaude
		case recipeName == "terminal_run_command":
			return "Terminal", ModeTerminal
		case recipeName == "cursor_chat_focus":
			return "Cursor", ModeClaude
		case hasPrefix(recipeName, "browser_"):
			return currentApp, ModeBrowser
		}

	case "open_app":
		appName := args["app_name"]
		switch appName {
		case "Terminal", "iTerm":
			return appName, ModeTerminal
		case "Safari", "Google Chrome", "Firefox", "Brave Browser", "Arc":
			return appName, ModeBrowser
		case "Visual Studio Code", "Cursor", "Antigravity":
			return appName, ModeEditor
		}
	}

	return "", ""
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
