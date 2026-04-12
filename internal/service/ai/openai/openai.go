package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
)

const (
	apiURL       = "https://api.openai.com/v1/chat/completions"
	defaultModel = "gpt-4o"
)

// Provider implements ai.Provider for OpenAI.
type Provider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// SupportsVision returns true — GPT-4o and compatible models support images.
func (p *Provider) SupportsVision() bool { return true }

// New creates an OpenAI provider. baseURL can be overridden for compatible APIs.
func New(apiKey, model, baseURL string) *Provider {
	if model == "" {
		model = defaultModel
	}
	if baseURL == "" {
		baseURL = apiURL
	}
	return &Provider{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// ---- wire types ----

type contentPart struct {
	Type     string       `json:"type"`
	Text     string       `json:"text,omitempty"`
	ImageURL *imageURLObj `json:"image_url,omitempty"`
}

type imageURLObj struct {
	URL string `json:"url"`
}

type wireMessage struct {
	Role       string        `json:"role"`
	Content    any           `json:"content"` // string or []contentPart
	ToolCallID string        `json:"tool_call_id,omitempty"`
	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
}

type wireToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"` // "function"
	Function wireFuncCall `json:"function"`
}

type wireFuncCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

type toolDef struct {
	Type     string       `json:"type"` // "function"
	Function toolFuncDef  `json:"function"`
}

type toolFuncDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type requestBody struct {
	Model    string        `json:"model"`
	Messages []wireMessage `json:"messages"`
	Tools    []toolDef     `json:"tools,omitempty"`
}

type responseBody struct {
	Choices []struct {
		Message      wireMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
}

// Chat sends the conversation to OpenAI and returns the response.
func (p *Provider) Chat(ctx context.Context, history []aimodel.Message, tools []aimodel.ToolDefinition) (*aimodel.Response, error) {
	msgs := convertHistory(history)
	wireDefs := convertTools(tools)

	body := requestBody{
		Model:    p.model,
		Messages: msgs,
		Tools:    wireDefs,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("openai: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: http: %w", err)
	}
	defer resp.Body.Close()

	var rb responseBody
	if err := json.NewDecoder(resp.Body).Decode(&rb); err != nil {
		return nil, fmt.Errorf("openai: decode: %w", err)
	}
	if len(rb.Choices) == 0 {
		return nil, fmt.Errorf("openai: empty choices")
	}

	return parseResponse(&rb.Choices[0].Message, rb.Choices[0].FinishReason), nil
}

func convertHistory(history []aimodel.Message) []wireMessage {
	msgs := make([]wireMessage, 0, len(history))
	for _, m := range history {
		wm := wireMessage{Role: m.Role}

		if m.Role == aimodel.RoleTool {
			// Tool result: content must be a plain string for OpenAI-compatible APIs.
			text := ""
			for _, p := range m.Parts {
				if p.Type == aimodel.ContentTypeText {
					text += p.Text
				}
			}
			if text == "" {
				text = "(no output)"
			}
			wm.Content = text
			wm.ToolCallID = m.ToolCallID
			msgs = append(msgs, wm)
			continue
		}

		// Assistant messages may carry tool calls from a previous turn.
		if m.Role == aimodel.RoleAssistant && len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				argsJSON, err := json.Marshal(tc.Arguments)
				if err != nil {
					argsJSON = []byte("{}")
				}
				wm.ToolCalls = append(wm.ToolCalls, wireToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: wireFuncCall{
						Name:      tc.ToolName,
						Arguments: string(argsJSON),
					},
				})
			}
		}

		var parts []contentPart
		for _, part := range m.Parts {
			switch part.Type {
			case aimodel.ContentTypeText:
				if part.Text != "" {
					parts = append(parts, contentPart{Type: "text", Text: part.Text})
				}
			case aimodel.ContentTypeImage:
				parts = append(parts, contentPart{
					Type: "image_url",
					ImageURL: &imageURLObj{
						URL: "data:" + part.MimeType + ";base64," + part.ImageBase64,
					},
				})
			}
		}
		switch {
		case len(parts) == 0:
			// Assistant with only tool_calls must still send content — use empty string.
			wm.Content = ""
		case len(parts) == 1 && parts[0].Type == "text":
			wm.Content = parts[0].Text
		default:
			wm.Content = parts
		}
		msgs = append(msgs, wm)
	}
	return msgs
}

func convertTools(tools []aimodel.ToolDefinition) []toolDef {
	defs := make([]toolDef, 0, len(tools))
	for _, t := range tools {
		props := map[string]any{}
		required := []string{}
		for _, p := range t.Parameters {
			props[p.Name] = map[string]string{
				"type":        p.Type,
				"description": p.Description,
			}
			if p.Required {
				required = append(required, p.Name)
			}
		}
		defs = append(defs, toolDef{
			Type: "function",
			Function: toolFuncDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters: map[string]any{
					"type":       "object",
					"properties": props,
					"required":   required,
				},
			},
		})
	}
	return defs
}

func parseResponse(msg *wireMessage, finishReason string) *aimodel.Response {
	var text string
	switch v := msg.Content.(type) {
	case string:
		text = v
	}

	var toolCalls []aimodel.ToolCall
	for _, tc := range msg.ToolCalls {
		var args map[string]string
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
		toolCalls = append(toolCalls, aimodel.ToolCall{
			ID:        tc.ID,
			ToolName:  tc.Function.Name,
			Arguments: args,
		})
	}

	return &aimodel.Response{
		Text:      text,
		ToolCalls: toolCalls,
		Done:      finishReason == "stop" && len(toolCalls) == 0,
	}
}
