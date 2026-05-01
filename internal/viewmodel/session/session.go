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

// Context tracks what app and mode the user is currently working in, plus a
// short-term memory of the last entities the user referenced (so "go to
// github" after "open Firefox" remembers Firefox; so "send hi" after "open
// whatsapp chat with Azozz" remembers Azozz).
type Context struct {
	ActiveApp   string    // "Visual Studio Code", "Terminal", "Safari", etc.
	ActiveMode  string    // terminal, claude, browser, editor
	WorkingPath string    // current project/directory path

	// Conversation memory — populated by InferContextFromAction.
	LastBrowser string // last browser used (Firefox, Google Chrome, ...)
	LastContact string // last messaging contact (Azozz, alice, #general, ...)
	LastURL     string // last URL navigated to
	LastQuery   string // last search query

	UpdatedAt time.Time // auto-expire after timeout
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

// Set stores a new context for a chat. Conversation-memory fields (LastX) are
// preserved from the previous context if the new entry doesn't override them.
func (m *Manager) Set(chatID int64, app, mode, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	prev := m.contexts[chatID]
	c := &Context{
		ActiveApp:   app,
		ActiveMode:  mode,
		WorkingPath: path,
		UpdatedAt:   time.Now(),
	}
	if prev != nil && time.Since(prev.UpdatedAt) <= contextTimeout {
		c.LastBrowser = prev.LastBrowser
		c.LastContact = prev.LastContact
		c.LastURL = prev.LastURL
		c.LastQuery = prev.LastQuery
	}
	m.contexts[chatID] = c
}

// Remember updates the conversation-memory fields without changing the active
// app/mode. Pass an empty string to leave a field unchanged.
func (m *Manager) Remember(chatID int64, browser, contact, url, query string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := m.contexts[chatID]
	if c == nil {
		c = &Context{UpdatedAt: time.Now()}
		m.contexts[chatID] = c
	}
	if browser != "" {
		c.LastBrowser = browser
	}
	if contact != "" {
		c.LastContact = contact
	}
	if url != "" {
		c.LastURL = url
	}
	if query != "" {
		c.LastQuery = query
	}
	c.UpdatedAt = time.Now()
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

// InferMemoryFromAction extracts conversation-memory fields from a tool call.
// Returns (browser, contact, url, query); each may be empty if not applicable.
//
// For run_recipe calls it inspects params_json (a JSON object literal) for
// the standard parameter names used by built-in recipes.
func InferMemoryFromAction(toolName string, args map[string]string) (browser, contact, url, query string) {
	switch toolName {
	case "open_app":
		app := args["app_name"]
		switch app {
		case "Safari", "Google Chrome", "Firefox", "Brave Browser", "Arc":
			browser = app
		}

	case "run_recipe":
		raw := args["params_json"]
		params := parseSimpleJSONObject(raw)
		if v := params["browser"]; v != "" {
			browser = v
		}
		if v := params["contact"]; v != "" {
			contact = v
		}
		if v := params["recipient"]; v != "" {
			contact = v
		}
		if v := params["channel"]; v != "" {
			contact = v
		}
		if v := params["url"]; v != "" {
			url = v
		}
		if v := params["query"]; v != "" {
			query = v
		}
	}
	return
}

// parseSimpleJSONObject is a tiny string-only JSON object parser (no nested
// objects, arrays, numbers, or booleans). Sufficient for tool-call params.
// Returns an empty map on any parse error.
func parseSimpleJSONObject(s string) map[string]string {
	out := map[string]string{}
	s = trimSpace(s)
	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return out
	}
	body := s[1 : len(s)-1]
	i := 0
	for i < len(body) {
		// skip whitespace
		for i < len(body) && (body[i] == ' ' || body[i] == '\t' || body[i] == '\n' || body[i] == ',') {
			i++
		}
		if i >= len(body) {
			break
		}
		if body[i] != '"' {
			return out
		}
		i++
		keyStart := i
		for i < len(body) && body[i] != '"' {
			i++
		}
		if i >= len(body) {
			return out
		}
		key := body[keyStart:i]
		i++ // closing "
		// skip : and whitespace
		for i < len(body) && (body[i] == ' ' || body[i] == ':') {
			i++
		}
		if i >= len(body) || body[i] != '"' {
			return out
		}
		i++
		valStart := i
		for i < len(body) && body[i] != '"' {
			// support \" inside string
			if body[i] == '\\' && i+1 < len(body) {
				i += 2
				continue
			}
			i++
		}
		if i > len(body) {
			return out
		}
		val := body[valStart:i]
		out[key] = val
		if i < len(body) {
			i++ // closing "
		}
	}
	return out
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t' || s[0] == '\n') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t' || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	return s
}
