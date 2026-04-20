package gemini

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
	apiBase      = "https://generativelanguage.googleapis.com/v1beta/models"
	defaultModel = "gemini-1.5-pro"
)

// Provider implements ai.Provider for Google Gemini.
type Provider struct {
	apiKey string
	model  string
	client *http.Client
}

// SupportsVision returns true — Gemini supports inline image data.
func (p *Provider) SupportsVision() bool { return true }

// New creates a Gemini provider.
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

type part struct {
	Text       string      `json:"text,omitempty"`
	InlineData *inlineData `json:"inline_data,omitempty"`
	FunctionCall *funcCall `json:"function_call,omitempty"`
	FunctionResponse *funcResp `json:"function_response,omitempty"`
}

type inlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"` // base64
}

type funcCall struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

type funcResp struct {
	Name     string         `json:"name"`
	Response map[string]any `json:"response"`
}

type wireContent struct {
	Role  string `json:"role"`
	Parts []part `json:"parts"`
}

type functionDecl struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type toolDecl struct {
	FunctionDeclarations []functionDecl `json:"function_declarations"`
}

type requestBody struct {
	Contents []wireContent `json:"contents"`
	Tools    []toolDecl    `json:"tools,omitempty"`
}

type candidate struct {
	Content wireContent `json:"content"`
}

type responseBody struct {
	Candidates []candidate `json:"candidates"`
}

// Chat sends the conversation to Gemini and returns the response.
func (p *Provider) Chat(ctx context.Context, history []aimodel.Message, tools []aimodel.ToolDefinition) (*aimodel.Response, error) {
	contents := convertHistory(history)
	toolDecls := convertTools(tools)

	body := requestBody{Contents: contents, Tools: toolDecls}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gemini: marshal: %w", err)
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", apiBase, p.model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gemini: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: http: %w", err)
	}
	defer resp.Body.Close()

	var rb responseBody
	if err := json.NewDecoder(resp.Body).Decode(&rb); err != nil {
		return nil, fmt.Errorf("gemini: decode: %w", err)
	}
	if len(rb.Candidates) == 0 {
		return nil, fmt.Errorf("gemini: no candidates in response")
	}

	return parseResponse(&rb.Candidates[0].Content), nil
}

func convertHistory(history []aimodel.Message) []wireContent {
	contents := make([]wireContent, 0, len(history))
	for _, m := range history {
		if m.Role == aimodel.RoleSystem {
			continue // handled via system_instruction if needed
		}
		role := m.Role
		if role == aimodel.RoleAssistant {
			role = "model"
		}
		if role == aimodel.RoleTool {
			role = "user" // Gemini expects tool results from user turn
		}

		var parts []part
		for _, mp := range m.Parts {
			switch mp.Type {
			case aimodel.ContentTypeText:
				parts = append(parts, part{Text: mp.Text})
			case aimodel.ContentTypeImage:
				parts = append(parts, part{InlineData: &inlineData{
					MimeType: mp.MimeType,
					Data:     mp.ImageBase64,
				}})
			}
		}
		contents = append(contents, wireContent{Role: role, Parts: parts})
	}
	return contents
}

func convertTools(tools []aimodel.ToolDefinition) []toolDecl {
	if len(tools) == 0 {
		return nil
	}
	decls := make([]functionDecl, 0, len(tools))
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
		decls = append(decls, functionDecl{
			Name:        t.Name,
			Description: t.Description,
			Parameters: map[string]any{
				"type":       "object",
				"properties": props,
				"required":   required,
			},
		})
	}
	return []toolDecl{{FunctionDeclarations: decls}}
}

func parseResponse(content *wireContent) *aimodel.Response {
	var text string
	var toolCalls []aimodel.ToolCall

	for _, p := range content.Parts {
		if p.Text != "" {
			text += p.Text
		}
		if p.FunctionCall != nil {
			args := make(map[string]string, len(p.FunctionCall.Args))
			for k, v := range p.FunctionCall.Args {
				args[k] = fmt.Sprintf("%v", v)
			}
			toolCalls = append(toolCalls, aimodel.ToolCall{
				ID:        p.FunctionCall.Name,
				ToolName:  p.FunctionCall.Name,
				Arguments: args,
			})
		}
	}

	return &aimodel.Response{
		Text:      text,
		ToolCalls: toolCalls,
		Done:      len(toolCalls) == 0,
	}
}
