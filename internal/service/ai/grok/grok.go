package grok

import (
	"context"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/openai"
)

const (
	apiURL       = "https://api.x.ai/v1/chat/completions"
	defaultModel = "grok-2-vision-1212"
)

// Provider implements ai.Provider for xAI Grok using the OpenAI-compatible API.
type Provider struct {
	inner *openai.Provider
}

// New creates a Grok provider.
func New(apiKey, model string) *Provider {
	if model == "" {
		model = defaultModel
	}
	return &Provider{inner: openai.New(apiKey, model, apiURL)}
}

// SupportsVision returns true — Grok vision model supports images.
func (p *Provider) SupportsVision() bool { return true }

// Chat delegates to the OpenAI-compatible implementation.
func (p *Provider) Chat(ctx context.Context, history []aimodel.Message, tools []aimodel.ToolDefinition) (*aimodel.Response, error) {
	return p.inner.Chat(ctx, history, tools)
}
