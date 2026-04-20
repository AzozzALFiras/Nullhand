package deepseek

import (
	"context"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	"github.com/AzozzALFiras/Nullhand/internal/service/ai/openai"
)

const (
	apiURL       = "https://api.deepseek.com/chat/completions"
	defaultModel = "deepseek-chat"
)

// Provider implements ai.Provider for DeepSeek using the OpenAI-compatible API.
type Provider struct {
	inner *openai.Provider
}

// New creates a DeepSeek provider.
func New(apiKey, model string) *Provider {
	if model == "" {
		model = defaultModel
	}
	return &Provider{inner: openai.New(apiKey, model, apiURL)}
}

// SupportsVision returns false — DeepSeek chat does not support image inputs.
func (p *Provider) SupportsVision() bool { return false }

// Chat delegates to the OpenAI-compatible implementation.
func (p *Provider) Chat(ctx context.Context, history []aimodel.Message, tools []aimodel.ToolDefinition) (*aimodel.Response, error) {
	return p.inner.Chat(ctx, history, tools)
}
