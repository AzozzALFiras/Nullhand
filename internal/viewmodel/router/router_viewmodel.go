package router

import (
	"strings"

	cmdmodel "github.com/iamakillah/Nullhand_Linux/internal/model/command"
	msgmodel "github.com/iamakillah/Nullhand_Linux/internal/model/message"
)

// RouteType indicates how a message should be handled.
type RouteType int

const (
	RouteManual    RouteType = iota // starts with "/"
	RouteAIAgent                   // natural language
	RouteConfirmYes                 // /yes
	RouteConfirmNo                  // /no
	RouteStop                      // /stop
)

// Route holds the routing decision for an incoming message.
type Route struct {
	Type    RouteType
	Command *cmdmodel.Command // set when Type == RouteManual
	Text    string            // set when Type == RouteAIAgent
}

// ViewModel parses incoming Telegram messages into routing decisions.
type ViewModel struct{}

// New creates a router ViewModel.
func New() *ViewModel { return &ViewModel{} }

// Dispatch inspects the message text and returns the appropriate Route.
func (vm *ViewModel) Dispatch(msg *msgmodel.Message) Route {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return Route{Type: RouteAIAgent, Text: ""}
	}

	lower := strings.ToLower(text)
	switch lower {
	case "/yes":
		return Route{Type: RouteConfirmYes}
	case "/no":
		return Route{Type: RouteConfirmNo}
	case "/stop":
		return Route{Type: RouteStop}
	}

	if strings.HasPrefix(text, "/") {
		return Route{Type: RouteManual, Command: parseCommand(text)}
	}

	return Route{Type: RouteAIAgent, Text: text}
}

// parseCommand splits "/click 100 200" into Command{Name:"click", Args:["100","200"]}.
func parseCommand(text string) *cmdmodel.Command {
	parts := strings.Fields(text)
	name := strings.TrimPrefix(strings.ToLower(parts[0]), "/")
	// Strip @botname suffix if present (e.g. /click@MyBot).
	if idx := strings.Index(name, "@"); idx >= 0 {
		name = name[:idx]
	}
	var args []string
	if len(parts) > 1 {
		args = parts[1:]
	}
	return &cmdmodel.Command{Name: name, Args: args, Raw: text}
}
