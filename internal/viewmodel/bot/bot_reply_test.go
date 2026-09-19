package bot

import (
	"fmt"
	"strings"
	"testing"

	tgfmt "github.com/AzozzALFiras/Nullhand/internal/view/telegram"
)

func TestReplyPartsShortText(t *testing.T) {
	parts := replyParts("✅ done")
	if len(parts) != 1 || parts[0] != "✅ done" {
		t.Errorf("short text must be sent as-is, got %q", parts)
	}
}

func TestReplyPartsSplitsLongOutput(t *testing.T) {
	text := tgfmt.Code(strings.Repeat("0123456789abcdef\n", 400)) // ~6.8k chars
	parts := replyParts(text)
	if len(parts) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(parts))
	}
	for i, p := range parts {
		if !strings.HasPrefix(p, "<pre>") || !strings.HasSuffix(p, "</pre>") {
			t.Errorf("part %d must be a complete <pre> block", i)
		}
	}
}

func TestReplyPartsCapsMessageCount(t *testing.T) {
	text := tgfmt.Code(strings.Repeat("0123456789abcdef\n", 5000)) // ~85k chars, ~21 parts
	total := len(tgfmt.Split(text, tgfmt.MaxMessageLen))
	parts := replyParts(text)

	if len(parts) != maxReplyParts {
		t.Fatalf("expected %d messages, got %d", maxReplyParts, len(parts))
	}
	last := parts[len(parts)-1]
	if !strings.Contains(last, "Output truncated") {
		t.Errorf("last message must explain the truncation, got %q", last)
	}
	omitted := total - (maxReplyParts - 1)
	if want := fmt.Sprintf("%d more message(s) not shown", omitted); !strings.Contains(last, want) {
		t.Errorf("note should say %q, got %q", want, last)
	}
}
