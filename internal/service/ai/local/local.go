package local

import (
	"context"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
)

// fallbackHelp is returned when the parser cannot understand the request.
const fallbackHelp = `I did not understand. The local parser supports:

Basic:
  English:  open <app>, type <text>, send, press <key>, click <label>,
            screenshot, list <path>, read <path>, run <shell>, copy <text>, paste
  Arabic:   افتح <app>، اكتب <text>، ارسل، اضغط <key>، انقر <label>،
            لقطة شاشة، اعرض <path>، اقرأ <path>، شغل <cmd>، انسخ <text>، الصق

Messaging (WhatsApp / Slack / Discord):
  open whatsapp and send azozz a message hello
  ارسل لعزوز في الواتساب: مرحبا
  واتساب عزوز: مرحبا

Browser:
  open firefox and go to github.com
  افتح فايرفوكس وروح إلى github.com
  اكتب google.com في شريط العنوان
  ابحث عن "go programming"
  ارجع / تحديث / علامة تبويب جديدة

Settings:
  search settings for wifi
  ابحث في الإعدادات عن WiFi
  open WiFi settings / افتح إعدادات WiFi

Click buttons:
  click the send button
  press button OK
  اضغط زر إرسال
  انقر على زر حفظ
  اضغط على Send في WhatsApp

Recipes:
  /recipes                    — list all available recipes
  /recipes show <name>        — see steps of one recipe
  recipe <name>               — run a recipe
  save this as recipe X: ...  — author your own recipe
  احفظ هذا كروتين X: ...        — same in Arabic

Preview / dry-run (see what would happen, NO execution):
  preview: open whatsapp and send azozz hi
  dry-run: open firefox and go to github.com
  معاينة: افتح فايرفوكس وروح إلى github.com

Diagnostics:
  /health                     — system, OCR, AI provider, schedule status

Combine steps with "and" / "then" / "," — or Arabic "ثم".
Example: open firefox and go to github.com and search for golang`

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

// SetSessionMemory updates the conversation-memory fields on the current
// session context. Empty strings leave the matching field unchanged.
// Creates an empty SessionContext if one wasn't set yet.
func (p *Provider) SetSessionMemory(lastBrowser, lastContact, lastURL, lastQuery string) {
	if p.sessionCtx == nil {
		p.sessionCtx = &SessionContext{}
	}
	if lastBrowser != "" {
		p.sessionCtx.LastBrowser = lastBrowser
	}
	if lastContact != "" {
		p.sessionCtx.LastContact = lastContact
	}
	if lastURL != "" {
		p.sessionCtx.LastURL = lastURL
	}
	if lastQuery != "" {
		p.sessionCtx.LastQuery = lastQuery
	}
}

// Chat inspects the conversation history, parses the latest user message,
// and returns either tool calls to execute or a final text reply.
func (p *Provider) Chat(_ context.Context, history []aimodel.Message, _ []aimodel.ToolDefinition) (*aimodel.Response, error) {
	// If the last message is a tool result, the agent already executed our
	// tool calls. Surface any error from the last batch; otherwise say Done.
	if len(history) > 0 && history[len(history)-1].Role == aimodel.RoleTool {
		for i := len(history) - 1; i >= 0 && history[i].Role == aimodel.RoleTool; i-- {
			for _, p := range history[i].Parts {
				if p.Type == aimodel.ContentTypeText &&
					(strings.HasPrefix(p.Text, "❌") || strings.HasPrefix(p.Text, "⚠️")) {
					return &aimodel.Response{Text: p.Text, Done: true}, nil
				}
			}
		}
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
