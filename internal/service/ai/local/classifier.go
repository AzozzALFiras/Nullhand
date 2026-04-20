package local

import (
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/antigravity"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/cursor"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/local/intents/vscode"
)

// Intent types returned by the classifier.
const (
	IntentAppFeature = "app_feature"  // open terminal OF vs code
	IntentAppCommand = "app_command"  // open terminal and run ls
	IntentMessaging  = "messaging"    // whatsapp Azozz say hello
	IntentBrowserNav = "browser_nav"  // open safari and go to X
	IntentFileBrowse = "file_browse"  // browse documents
	IntentGitAction  = "git_action"   // git push in vs code
	IntentOpenApp    = "open_app"     // open safari
	IntentSimple     = "simple"       // screenshot, send, press key
)

// ClassifiedIntent is the result of classification.
type ClassifiedIntent struct {
	Type    string
	App     string // primary app
	Feature string // sub-feature (terminal, claude, search...)
	Command string // command to run
	Message string // message to send
	Contact string // contact name
	URL     string // URL to navigate
	Query   string // search query
	Path    string // file/directory path
	GitOp   string // git operation (push, pull, commit...)
}

// SessionContext is a minimal context passed from the session manager.
// Avoids importing the session package (which would cause cycles).
type SessionContext struct {
	ActiveApp  string // "Visual Studio Code", "Terminal", etc.
	ActiveMode string // "terminal", "claude", "browser", "editor"
}

// Classify analyzes extracted entities and determines the user's intent.
// If ctx is non-nil, it provides session context for ambiguous commands.
func Classify(e *Entities) *ClassifiedIntent {
	return ClassifyWithContext(e, nil)
}

// ClassifyWithContext analyzes entities with optional session context.
func ClassifyWithContext(e *Entities, ctx *SessionContext) *ClassifiedIntent {
	ci := &ClassifiedIntent{}

	// ── Priority 1: App Feature (modifier links two things) ──────────
	// "open terminal of vs code" / "search X in vs code" / "git push in vs code"
	if e.Modifier != nil && len(e.Apps) > 0 {
		// Check if modifier links a feature TO an IDE
		for _, app := range e.Apps {
			if IsIDE(app.Name) {
				feature := detectIDEFeature(e, app.Name)
				if feature != "" {
					ci.Type = IntentAppFeature
					ci.App = app.Name
					ci.Feature = feature
					ci.Query = e.TextAfterApps()
					ci.Command = e.TextAfterApps()
					if e.Message != "" {
						ci.Message = e.Message
					}
					return ci
				}
			}
		}
	}

	// ── Priority 2: Git Action ───────────────────────────────────────
	if gitOp := e.HasGitAction(); gitOp != "" {
		ci.Type = IntentGitAction
		ci.GitOp = gitOp
		ci.App = e.PrimaryApp()
		if ci.App == "" {
			ci.App = "Terminal"
		}
		ci.Path = e.PrimaryPath()
		return ci
	}

	// ── Priority 3: App Command (terminal/IDE + command) ─────────────
	// "open terminal and run ls" / "terminal ls -la" / "افتح التيرمنل ونفذ ls"
	if len(e.Apps) > 0 {
		app := e.PrimaryApp()

		if IsTerminal(app) && (e.HasAnyAction("run", "open") || !e.HasAnyAction("browse")) {
			cmd := e.TextAfterApps()
			if cmd != "" {
				ci.Type = IntentAppCommand
				ci.App = app
				ci.Command = cmd
				return ci
			}
		}

		// IDE + run action → run in IDE terminal
		if IsIDE(app) && e.HasAction("run") {
			cmd := e.TextAfterApps()
			if cmd != "" {
				ci.Type = IntentAppFeature
				ci.App = app
				ci.Feature = "terminal_run"
				ci.Command = cmd
				return ci
			}
		}
	}

	// ── Priority 4: Messaging ────────────────────────────────────────
	if len(e.Apps) > 0 && IsMessaging(e.PrimaryApp()) && e.HasAction("send") {
		ci.Type = IntentMessaging
		ci.App = e.PrimaryApp()
		ci.Contact = e.Contact
		ci.Message = e.Message
		if ci.Message == "" {
			ci.Message = e.TextAfterApps()
		}
		return ci
	}

	// ── Priority 5: Browser Navigation ───────────────────────────────
	if len(e.Apps) > 0 && IsBrowserApp(e.PrimaryApp()) {
		if len(e.URLs) > 0 {
			ci.Type = IntentBrowserNav
			ci.App = e.PrimaryApp()
			ci.URL = e.URLs[0]
			return ci
		}
		if e.HasAction("search") {
			ci.Type = IntentBrowserNav
			ci.App = e.PrimaryApp()
			ci.Query = e.TextAfterApps()
			return ci
		}
	}

	// Direct search without browser specified
	if e.HasAction("search") && len(e.Apps) == 0 {
		ci.Type = IntentBrowserNav
		ci.App = "Safari"
		ci.Query = e.TextAfterApps()
		if ci.Query != "" {
			return ci
		}
	}

	// ── Priority 6: File Browse ──────────────────────────────────────
	if e.HasAction("browse") || (len(e.Paths) > 0 && !e.HasAction("open")) {
		ci.Type = IntentFileBrowse
		ci.Path = e.PrimaryPath()
		if ci.Path == "" {
			// Use remaining text as path
			remaining := e.TextAfterApps()
			if remaining != "" {
				ci.Path = remaining
			} else {
				ci.Path = "~"
			}
		}
		return ci
	}

	// Browse by keywords: "list/show folders/files in X"
	if e.HasAnyAction("browse") && len(e.Paths) > 0 {
		ci.Type = IntentFileBrowse
		ci.Path = e.PrimaryPath()
		return ci
	}

	// ── Priority 7: Open App ─────────────────────────────────────────
	if len(e.Apps) > 0 && e.HasAction("open") {
		ci.Type = IntentOpenApp
		ci.App = e.PrimaryApp()
		return ci
	}

	// ── Priority 8: If we have an app but no clear action ────────────
	if len(e.Apps) > 0 {
		ci.Type = IntentOpenApp
		ci.App = e.PrimaryApp()
		return ci
	}

	// ── No classification → fall through to simple intents ───────────
	// But first check if session context can help
	ci.Type = IntentSimple
	return ci
}

// ApplyContext uses session context to handle unrecognized text.
// Called when Parse() gets no results from both classifier and simple intents.
// Returns tool calls if context applies, nil otherwise.
func ApplyContext(text string, ctx *SessionContext) []aimodel.ToolCall {
	if ctx == nil || ctx.ActiveMode == "" {
		return nil
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	switch ctx.ActiveMode {
	case "terminal":
		// In terminal mode: anything unrecognized = type it as a command
		return []aimodel.ToolCall{
			intents.ToolCall("type_text", map[string]string{"text": text}),
			intents.ToolCall("press_key", map[string]string{"key": "return"}),
		}

	case "claude":
		// In claude/chat mode: type the message and send
		return []aimodel.ToolCall{
			intents.ToolCall("type_text", map[string]string{"text": text}),
			intents.ToolCall("press_key", map[string]string{"key": "return"}),
		}

	case "browser":
		// In browser mode: if looks like URL, navigate; otherwise search
		if isLikelyURL(text) {
			return []aimodel.ToolCall{
				intents.ToolCall("press_key", map[string]string{"key": "cmd+l"}),
				intents.ToolCall("wait", map[string]string{"ms": "200"}),
				intents.ToolCall("type_text", map[string]string{"text": text}),
				intents.ToolCall("press_key", map[string]string{"key": "return"}),
			}
		}
		return []aimodel.ToolCall{
			intents.ToolCall("press_key", map[string]string{"key": "cmd+l"}),
			intents.ToolCall("wait", map[string]string{"ms": "200"}),
			intents.ToolCall("type_text", map[string]string{"text": text}),
			intents.ToolCall("press_key", map[string]string{"key": "return"}),
		}
	}

	return nil
}

func isLikelyURL(text string) bool {
	return strings.Contains(text, ".") && !strings.Contains(text, " ")
}

// detectIDEFeature determines what IDE feature is being requested.
func detectIDEFeature(e *Entities, ideName string) string {
	lower := e.LowerText

	// Check for terminal-related words
	terminalWords := []string{"terminal", "term", "تيرمنل", "الطرفية", "طرفية"}
	for _, tw := range terminalWords {
		if strings.Contains(lower, tw) {
			// "open terminal of vs code" → terminal
			return "terminal"
		}
	}

	// Check for Claude/AI chat words
	claudeWords := []string{"claude", "chat", "كلود", "شات", "agent", "ai"}
	for _, cw := range claudeWords {
		if strings.Contains(lower, cw) {
			return "claude"
		}
	}

	// Check for search
	if e.HasAction("search") {
		return "search"
	}

	// Check for git operations
	if gitOp := e.HasGitAction(); gitOp != "" {
		return "git_" + gitOp
	}

	// Check for settings
	if strings.Contains(lower, "settings") || strings.Contains(lower, "اعدادات") {
		return "settings"
	}

	// Check for extensions
	if strings.Contains(lower, "extension") || strings.Contains(lower, "اضافات") {
		return "extensions"
	}

	return ""
}

// BuildToolCalls converts a classified intent into executable tool calls.
func BuildToolCalls(ci *ClassifiedIntent) []aimodel.ToolCall {
	switch ci.Type {

	case IntentAppFeature:
		return buildAppFeatureCalls(ci)

	case IntentAppCommand:
		if IsTerminal(ci.App) {
			return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
				"name":        "terminal_run_command",
				"params_json": intents.MustJSON(map[string]string{"command": ci.Command}),
			})}
		}
		return []aimodel.ToolCall{
			intents.ToolCall("open_app", map[string]string{"app_name": ci.App}),
			intents.ToolCall("wait", map[string]string{"ms": "500"}),
			intents.ToolCall("type_text", map[string]string{"text": ci.Command}),
			intents.ToolCall("press_key", map[string]string{"key": "return"}),
		}

	case IntentMessaging:
		return buildMessagingCalls(ci)

	case IntentBrowserNav:
		return buildBrowserCalls(ci)

	case IntentFileBrowse:
		path := ci.Path
		if path == "" {
			path = "~"
		}
		return []aimodel.ToolCall{intents.ToolCall("browse_folder", map[string]string{"path": path})}

	case IntentGitAction:
		return buildGitCalls(ci)

	case IntentOpenApp:
		return []aimodel.ToolCall{intents.ToolCall("open_app", map[string]string{"app_name": ci.App})}

	default:
		return nil
	}
}

func buildAppFeatureCalls(ci *ClassifiedIntent) []aimodel.ToolCall {
	switch ci.App {
	case "Visual Studio Code":
		return vscode.BuildFeature(ci.Feature, ci.Command, ci.Message, ci.Query)
	case "Cursor":
		return cursor.BuildFeature(ci.Feature, ci.Command, ci.Message, ci.Query)
	case "Antigravity":
		return antigravity.BuildFeature(ci.Feature, ci.Command, ci.Message, ci.Query)
	default:
		return []aimodel.ToolCall{intents.ToolCall("open_app", map[string]string{"app_name": ci.App})}
	}
}

func buildMessagingCalls(ci *ClassifiedIntent) []aimodel.ToolCall {
	switch ci.App {
	case "WhatsApp":
		if ci.Message != "" && ci.Contact != "" {
			return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
				"name":        "whatsapp_send_message",
				"params_json": intents.MustJSON(map[string]string{"contact": ci.Contact, "message": ci.Message}),
			})}
		}
		if ci.Contact != "" {
			return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
				"name":        "whatsapp_new_message",
				"params_json": intents.MustJSON(map[string]string{"contact": ci.Contact}),
			})}
		}
	case "Slack":
		if ci.Message != "" && ci.Contact != "" {
			return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
				"name":        "slack_send_message",
				"params_json": intents.MustJSON(map[string]string{"channel": ci.Contact, "message": ci.Message}),
			})}
		}
	case "Discord":
		if ci.Message != "" && ci.Contact != "" {
			return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
				"name":        "discord_send_dm",
				"params_json": intents.MustJSON(map[string]string{"recipient": ci.Contact, "message": ci.Message}),
			})}
		}
	}
	return []aimodel.ToolCall{intents.ToolCall("open_app", map[string]string{"app_name": ci.App})}
}

func buildBrowserCalls(ci *ClassifiedIntent) []aimodel.ToolCall {
	if ci.URL != "" {
		return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
			"name":        "browser_open_url",
			"params_json": intents.MustJSON(map[string]string{"browser": ci.App, "url": ci.URL}),
		})}
	}
	if ci.Query != "" {
		return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
			"name":        "browser_google_search",
			"params_json": intents.MustJSON(map[string]string{"browser": ci.App, "query": ci.Query}),
		})}
	}
	return []aimodel.ToolCall{intents.ToolCall("open_app", map[string]string{"app_name": ci.App})}
}

func buildGitCalls(ci *ClassifiedIntent) []aimodel.ToolCall {
	if IsIDE(ci.App) {
		return buildAppFeatureCalls(&ClassifiedIntent{
			Type:    IntentAppFeature,
			App:     ci.App,
			Feature: "git_" + ci.GitOp,
		})
	}
	return []aimodel.ToolCall{intents.ToolCall("run_recipe", map[string]string{
		"name":        "terminal_run_command",
		"params_json": intents.MustJSON(map[string]string{"command": "git " + ci.GitOp}),
	})}
}
