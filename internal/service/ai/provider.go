package ai

import (
	"context"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
)

// Provider is the common interface all AI backends must implement.
// Chat sends one round of the conversation and returns the AI response.
// history contains the full conversation so far.
// tools contains the definitions of tools the AI may call.
type Provider interface {
	Chat(ctx context.Context, history []aimodel.Message, tools []aimodel.ToolDefinition) (*aimodel.Response, error)
}
