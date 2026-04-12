package local

import (
	"context"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
)

// fallbackHelp is returned when the parser cannot understand the request.
const fallbackHelp = `I did not understand. The local parser supports:

English: open <app>, type <text>, send, press <key>, click <label>,
         screenshot, list <path>, read <path>, run <shell>, copy <text>, paste
Arabic:  افتح <app>، اكتب <text>، ارسل، اضغط <key>، انقر <label>،
         لقطة شاشة، اعرض <path>، اقرأ <path>، شغل <cmd>، انسخ <text>، الصق

Combine steps with "and" / "then" / "," — or Arabic "ثم".
Example: open Safari and type hello and send`

// Provider is a zero-dependency, zero-cost AI that parses user text into
// tool calls using pattern matching and entity extraction.
type Provider struct {
	// sessionCtx holds the current session context for context-aware parsing.
	// Set by SetSessionContext before Chat is called.
	sessionCtx *SessionContext
}

// SupportsVision returns false — the local parser is text-only.
func (p *Provider) SupportsVision() bool { return false }

// New creates a local rule-based provider.
func New() *Provider { return &Provider{} }

// SetSessionContext sets the session context for the next Chat call.
// This allows the bot to pass context (e.g. "we're in terminal mode")
// so the parser can handle ambiguous commands like bare "ls".
func (p *Provider) SetSessionContext(app, mode string) {
	if app == "" && mode == "" {
		p.sessionCtx = nil
		return
	}
	p.sessionCtx = &SessionContext{ActiveApp: app, ActiveMode: mode}
}

// Chat inspects the conversation history, parses the latest user message,
// and returns either tool calls to execute or a final text reply.
func (p *Provider) Chat(_ context.Context, history []aimodel.Message, _ []aimodel.ToolDefinition) (*aimodel.Response, error) {
	// If the last message is a tool result, the agent already executed our
	// tool calls — we just need to finish the task.
	if len(history) > 0 && history[len(history)-1].Role == aimodel.RoleTool {
		return &aimodel.Response{Text: "Done.", Done: true}, nil
	}

	userText := latestUserText(history)
	if userText == "" {
		return &aimodel.Response{Text: fallbackHelp, Done: true}, nil
	}

	calls := ParseWithContext(userText, p.sessionCtx)
	if len(calls) == 0 {
		return &aimodel.Response{Text: fallbackHelp, Done: true}, nil
	}

	return &aimodel.Response{ToolCalls: calls}, nil
}

// latestUserText returns the text of the most recent user message in history.
func latestUserText(history []aimodel.Message) string {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Role != aimodel.RoleUser {
			continue
		}
		for _, part := range history[i].Parts {
			if part.Type == aimodel.ContentTypeText && part.Text != "" {
				return part.Text
			}
		}
	}
	return ""
}
