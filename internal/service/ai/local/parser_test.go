package local

import (
	"context"
	"strings"
	"testing"

	aimodel "github.com/AzozzALFiras/nullhand/internal/model/ai"
)

func TestParser(t *testing.T) {
	cases := []struct {
		input     string
		wantTools []string // ordered list of tool names expected
	}{
		{"open Safari", []string{"open_app"}},
		{"افتح Safari", []string{"open_app"}},
		{"type hello world", []string{"type_text"}},
		{"اكتب مرحبا", []string{"type_text"}},
		{"screenshot", []string{"take_screenshot"}},
		{"خذ لقطة شاشة", []string{"take_screenshot"}},
		{"list ~/Desktop", []string{"list_directory"}},
		{"run ls -la", []string{"run_shell"}},
		{"send", []string{"press_key"}},
		{"ارسل", []string{"press_key"}},
		{"copy hello", []string{"set_clipboard"}},
		{"paste", []string{"get_clipboard"}},
		{"open Safari and type github.com and send", []string{"open_app", "type_text", "press_key"}},
		{"افتح VS Code ثم اكتب hello ثم ارسل", []string{"open_app", "type_text", "press_key"}},
		{"something totally random", nil},
	}

	for _, tc := range cases {
		got := Parse(tc.input)
		if tc.wantTools == nil {
			if got != nil {
				t.Errorf("Parse(%q) = %v, want nil", tc.input, toolNames(got))
			}
			continue
		}
		if len(got) != len(tc.wantTools) {
			t.Errorf("Parse(%q) length = %d, want %d (got %v)", tc.input, len(got), len(tc.wantTools), toolNames(got))
			continue
		}
		for i, want := range tc.wantTools {
			if got[i].ToolName != want {
				t.Errorf("Parse(%q)[%d] = %s, want %s", tc.input, i, got[i].ToolName, want)
			}
		}
	}
}

func TestProviderFinishAfterTool(t *testing.T) {
	p := New()
	history := []aimodel.Message{
		{Role: aimodel.RoleSystem, Parts: []aimodel.MessagePart{{Type: aimodel.ContentTypeText, Text: "sys"}}},
		{Role: aimodel.RoleUser, Parts: []aimodel.MessagePart{{Type: aimodel.ContentTypeText, Text: "open Safari"}}},
		{Role: aimodel.RoleAssistant, ToolCalls: []aimodel.ToolCall{{ID: "x", ToolName: "open_app", Arguments: map[string]string{"app_name": "Safari"}}}},
		{Role: aimodel.RoleTool, ToolCallID: "x", Parts: []aimodel.MessagePart{{Type: aimodel.ContentTypeText, Text: "opened"}}},
	}
	resp, err := p.Chat(context.Background(), history, nil)
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if !resp.Done || len(resp.ToolCalls) != 0 {
		t.Errorf("expected done=true with no tool calls, got done=%v calls=%v", resp.Done, resp.ToolCalls)
	}
}

func TestProviderFallbackOnUnknown(t *testing.T) {
	p := New()
	history := []aimodel.Message{
		{Role: aimodel.RoleUser, Parts: []aimodel.MessagePart{{Type: aimodel.ContentTypeText, Text: "fly to the moon"}}},
	}
	resp, _ := p.Chat(context.Background(), history, nil)
	if len(resp.ToolCalls) != 0 {
		t.Errorf("expected no tool calls for unknown input, got %d", len(resp.ToolCalls))
	}
	if !strings.Contains(resp.Text, "did not understand") {
		t.Errorf("expected fallback help text, got: %s", resp.Text)
	}
}

func toolNames(calls []aimodel.ToolCall) []string {
	out := make([]string, len(calls))
	for i, c := range calls {
		out[i] = c.ToolName
	}
	return out
}
