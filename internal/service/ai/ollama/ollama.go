package ollama

import (
	"context"
	"strings"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
	"github.com/AzozzALFiras/nullhand/internal/service/ai/openai"
)

const (
	// DefaultBaseURL is the Ollama server's OpenAI-compatible endpoint.
	DefaultBaseURL = "http://localhost:11434/v1/chat/completions"
	// DefaultModel is used when the developer does not specify one.
	DefaultModel = "qwen2.5:3b"
)

// Provider implements ai.Provider by talking to a local Ollama server via
// its OpenAI-compatible /v1/chat/completions endpoint. Developers can point
// it at any Ollama instance (local or remote) by setting baseURL in config.
type Provider struct {
	inner *openai.Provider
}

// New creates an Ollama provider.
//
// baseURL may be:
//   - empty → uses DefaultBaseURL (http://localhost:11434/v1/chat/completions)
//   - a host like "http://localhost:11434" → path is appended automatically
//   - a full URL ending in /v1/chat/completions → used as-is
//
// model may be empty → uses DefaultModel.
func New(baseURL, model string) *Provider {
	baseURL = normalizeBaseURL(baseURL)
	if model == "" {
		model = DefaultModel
	}
	// Ollama does not check the API key but openai.New requires a non-empty one.
	return &Provider{inner: openai.New("ollama", model, baseURL)}
}

// Chat delegates to the embedded OpenAI-compatible client.
func (p *Provider) Chat(ctx context.Context, history []aimodel.Message, tools []aimodel.ToolDefinition) (*aimodel.Response, error) {
	return p.inner.Chat(ctx, history, tools)
}

// normalizeBaseURL accepts either a bare host or a full endpoint URL and
// returns a canonical /v1/chat/completions URL suitable for the OpenAI client.
func normalizeBaseURL(raw string) string {
	if raw == "" {
		return DefaultBaseURL
	}
	raw = strings.TrimRight(raw, "/")
	if strings.HasSuffix(raw, "/v1/chat/completions") {
		return raw
	}
	if strings.HasSuffix(raw, "/v1") {
		return raw + "/chat/completions"
	}
	return raw + "/v1/chat/completions"
}
