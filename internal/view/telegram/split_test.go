package telegram

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

func TestSplitShortMessageUnchanged(t *testing.T) {
	msg := "<b>hello</b> world"
	parts := Split(msg, MaxMessageLen)
	if len(parts) != 1 || parts[0] != msg {
		t.Fatalf("short message must pass through untouched, got %q", parts)
	}
}

func TestSplitRespectsLimit(t *testing.T) {
	msg := strings.Repeat("line of output\n", 1000) // 15000 chars
	parts := Split(msg, MaxMessageLen)
	if len(parts) < 4 {
		t.Fatalf("expected at least 4 parts, got %d", len(parts))
	}
	for i, p := range parts {
		if n := utf16Len(p); n > MaxMessageLen {
			t.Errorf("part %d is %d units, over the %d limit", i, n, MaxMessageLen)
		}
	}
	if got := strings.Join(parts, ""); got != msg {
		t.Error("plain text parts must concatenate back to the original")
	}
}

func TestSplitPrefersLineBreaks(t *testing.T) {
	msg := strings.Repeat("0123456789\n", 30)
	parts := Split(msg, 100)
	for i, p := range parts[:len(parts)-1] {
		if !strings.HasSuffix(p, "\n") {
			t.Errorf("part %d should end at a line break, got %q", i, p)
		}
	}
}

func TestSplitHardCutsLongLine(t *testing.T) {
	msg := strings.Repeat("x", 250) // no line breaks at all
	parts := Split(msg, 100)
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d: %q", len(parts), parts)
	}
	if strings.Join(parts, "") != msg {
		t.Error("hard-cut parts must concatenate back to the original")
	}
}

func TestSplitReopensPreAcrossParts(t *testing.T) {
	msg := Code(strings.Repeat("some log line\n", 50))
	parts := Split(msg, 200)
	if len(parts) < 2 {
		t.Fatalf("expected several parts, got %d", len(parts))
	}
	for i, p := range parts {
		if !strings.HasPrefix(p, "<pre>") || !strings.HasSuffix(p, "</pre>") {
			t.Errorf("part %d must be a complete <pre> block, got %q", i, p)
		}
		if n := utf16Len(p); n > 200 {
			t.Errorf("part %d is %d units including tags, over the limit", i, n)
		}
	}
}

func TestSplitKeepsAttributesWhenReopening(t *testing.T) {
	msg := `<a href="https://example.com">` + strings.Repeat("link text ", 30) + `</a>`
	parts := Split(msg, 120)
	for i, p := range parts {
		if !strings.HasPrefix(p, `<a href="https://example.com">`) || !strings.HasSuffix(p, "</a>") {
			t.Errorf("part %d must reopen the link with its href, got %q", i, p)
		}
	}
}

func TestSplitNeverBreaksEntitiesOrTags(t *testing.T) {
	msg := Code(strings.Repeat("a < b && c > d\n", 40))
	partial := regexp.MustCompile(`&[a-z]*$|^[a-z]*;|<[^>]*$`)
	for i, p := range Split(msg, 97) {
		body := strings.TrimSuffix(strings.TrimPrefix(p, "<pre>"), "</pre>")
		if partial.MatchString(body) {
			t.Errorf("part %d cuts through an entity or tag: %q", i, p)
		}
	}
}

func TestSplitCountsUTF16Units(t *testing.T) {
	// Each emoji is 2 UTF-16 units, so 60 of them (120 units) can't fit in 100.
	msg := strings.Repeat("😀", 60)
	parts := Split(msg, 100)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	for i, p := range parts {
		if n := utf16Len(p); n > 100 {
			t.Errorf("part %d is %d UTF-16 units", i, n)
		}
	}
}

func TestSplitArabicText(t *testing.T) {
	msg := strings.Repeat("مرحبا بالعالم\n", 100)
	parts := Split(msg, 300)
	if strings.Join(parts, "") != msg {
		t.Error("Arabic text must survive splitting intact")
	}
}

func TestSplitDropsPartsWithoutVisibleText(t *testing.T) {
	msg := "<pre>" + strings.Repeat("x", 90) + "\n" + strings.Repeat(" \n", 60) + "</pre>"
	for i, p := range Split(msg, 100) {
		if strings.TrimSpace(strings.NewReplacer("<pre>", "", "</pre>", "").Replace(p)) == "" {
			t.Errorf("part %d has no visible text and would be rejected by Telegram: %q", i, p)
		}
	}
}

func TestSplitTreatsBareAnglesAsText(t *testing.T) {
	// Not produced by our formatters, but must never panic.
	for _, msg := range []string{"<>", "< >", "a <", strings.Repeat("<> ", 100)} {
		_ = Split(msg, 20)
	}
}

func TestDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{2 * time.Minute, "2 min"},
		{30 * time.Minute, "30 min"},
		{45 * time.Second, "45s"},
		{90 * time.Second, "1m30s"},
	}
	for _, c := range cases {
		if got := Duration(c.d); got != c.want {
			t.Errorf("Duration(%s) = %q, want %q", c.d, got, c.want)
		}
	}
}
