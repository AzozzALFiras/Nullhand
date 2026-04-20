package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
)

const (
	apiURL        = "https://api.anthropic.com/v1/messages"
	anthropicVer  = "2023-06-01"
	defaultModel  = "claude-opus-4-6"
	maxTokens     = 4096
)

// Provider implements ai.Provider for Anthropic Claude.
type Provider struct {
	apiKey string
	model  string
	client *http.Client
}

// SupportsVision returns true — Claude supports image content blocks.
func (p *Provider) SupportsVision() bool { return true }

// New creates a Claude provider. If model is empty, uses the default.
func New(apiKey, model string) *Provider {
	if model == "" {
		model = defaultModel
	}
	return &Provider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

// ---- wire types ----

type contentBlock struct {
	Type       string          `json:"type"`
	Text       string          `json:"text,omitempty"`
	Source     *imageSource    `json:"source,omitempty"`
	ID         string          `json:"id,omitempty"`
	Name       string          `json:"name,omitempty"`
	Input      json.RawMessage `json:"input,omitempty"`
	ToolUseID  string          `json:"tool_use_id,omitempty"`
	Content    string          `json:"content,omitempty"`
}

type imageSource struct {
	Type      string `json:"type"` // "base64"
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type wireMessage struct {
	Role    string         `json:"role"`
	Content []contentBlock `json:"content"`
}

type toolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type requestBody struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	System    string        `json:"system,omitempty"`
	Messages  []wireMessage `json:"messages"`
	Tools     []toolDef     `json:"tools,omitempty"`
}

type responseBody struct {
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
}

// Chat sends the conversation to Claude and returns the response.
func (p *Provider) Chat(ctx context.Context, history []aimodel.Message, tools []aimodel.ToolDefinition) (*aimodel.Response, error) {
	msgs, systemPrompt := convertHistory(history)
	wireDefs := convertTools(tools)

	body := requestBody{
		Model:     p.model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  msgs,
		Tools:     wireDefs,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("claude: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("claude: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", anthropicVer)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("claude: http: %w", err)
	}
	defer resp.Body.Close()

	var rb responseBody
	if err := json.NewDecoder(resp.Body).Decode(&rb); err != nil {
		return nil, fmt.Errorf("claude: decode response: %w", err)
	}

	return parseResponse(&rb), nil
}

// convertHistory maps internal messages to Claude wire format.
func convertHistory(history []aimodel.Message) ([]wireMessage, string) {
	var systemPrompt string
	var msgs []wireMessage

	for _, m := range history {
		if m.Role == aimodel.RoleSystem {
			for _, p := range m.Parts {
				systemPrompt += p.Text
			}
			continue
		}

		var blocks []contentBlock
		for _, part := range m.Parts {
			switch part.Type {
			case aimodel.ContentTypeText:
				blocks = append(blocks, contentBlock{Type: "text", Text: part.Text})
			case aimodel.ContentTypeImage:
				blocks = append(blocks, contentBlock{
					Type: "image",
					Source: &imageSource{
						Type:      "base64",
						MediaType: part.MimeType,
						Data:      part.ImageBase64,
					},
				})
			}
		}
		msgs = append(msgs, wireMessage{Role: m.Role, Content: blocks})
	}
	return msgs, systemPrompt
}

// convertTools maps ToolDefinition to Claude's tool schema format.
func convertTools(tools []aimodel.ToolDefinition) []toolDef {
	defs := make([]toolDef, 0, len(tools))
	for _, t := range tools {
		schema := buildInputSchema(t.Parameters)
		raw, _ := json.Marshal(schema)
		defs = append(defs, toolDef{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: raw,
		})
	}
	return defs
}

// buildInputSchema produces a JSON Schema object for the tool parameters.
func buildInputSchema(params []aimodel.ToolParameter) map[string]any {
	properties := map[string]any{}
	required := []string{}
	for _, p := range params {
		properties[p.Name] = map[string]string{
			"type":        p.Type,
			"description": p.Description,
		}
		if p.Required {
			required = append(required, p.Name)
		}
	}
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

// parseResponse converts the raw Claude response into the internal Response type.
func parseResponse(rb *responseBody) *aimodel.Response {
	var textParts []string
	var toolCalls []aimodel.ToolCall

	for _, block := range rb.Content {
		switch block.Type {
		case "text":
			textParts = append(textParts, block.Text)
		case "tool_use":
			var args map[string]string
			_ = json.Unmarshal(block.Input, &args)
			toolCalls = append(toolCalls, aimodel.ToolCall{
				ID:        block.ID,
				ToolName:  block.Name,
				Arguments: args,
			})
		}
	}

	text := ""
	for _, t := range textParts {
		text += t
	}

	return &aimodel.Response{
		Text:      text,
		ToolCalls: toolCalls,
		Done:      rb.StopReason == "end_turn" && len(toolCalls) == 0,
	}
}
