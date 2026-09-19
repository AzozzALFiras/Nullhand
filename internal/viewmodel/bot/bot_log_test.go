package bot

import (
	"strings"
	"testing"
)

func TestJoinWithBudgetKeepsAllWhenFits(t *testing.T) {
	lines := []string{"aaa", "bbb", "ccc"}
	got, dropped := joinWithBudget(lines, 1000)
	if dropped != 0 {
		t.Errorf("nothing should be dropped, got dropped=%d", dropped)
	}
	if got != "aaa\nbbb\nccc" {
		t.Errorf("unexpected join: %q", got)
	}
}

func TestJoinWithBudgetDropsOldestFirst(t *testing.T) {
	// Each line is 4 bytes including the implicit trailing newline.
	// budget=8 should keep exactly the two newest lines.
	lines := []string{"aaa", "bbb", "ccc"}
	got, dropped := joinWithBudget(lines, 8)
	if dropped != 1 {
		t.Errorf("oldest line should be dropped (dropped=1), got %d", dropped)
	}
	if got != "bbb\nccc" {
		t.Errorf("expected last two lines, got %q", got)
	}
}

func TestJoinWithBudgetZeroBudgetDropsAll(t *testing.T) {
	got, dropped := joinWithBudget([]string{"a", "b"}, 0)
	if got != "" || dropped != 2 {
		t.Errorf("zero budget must drop everything: got=%q dropped=%d", got, dropped)
	}
}

func TestJoinWithBudgetSingleLineLargerThanBudget(t *testing.T) {
	// A line bigger than the budget cannot fit at all — we must not panic
	// and should report the line as dropped rather than emitting it.
	huge := strings.Repeat("x", 100)
	got, dropped := joinWithBudget([]string{huge}, 50)
	if got != "" {
		t.Errorf("oversized single line should not be emitted, got %q", got)
	}
	if dropped != 1 {
		t.Errorf("expected dropped=1, got %d", dropped)
	}
}

func TestJoinWithBudgetExactFit(t *testing.T) {
	// "ab\ncd" is 5 bytes (2 + newline + 2). Budget 5 should keep both.
	got, dropped := joinWithBudget([]string{"ab", "cd"}, 6)
	if dropped != 0 || got != "ab\ncd" {
		t.Errorf("budget=6 should keep both: got=%q dropped=%d", got, dropped)
	}
}

func TestFormatLogReplyRedactsAndEscapes(t *testing.T) {
	lines := []string{
		`[2026-09-19 09:00:00] user=1 action=shell cmd="export GITHUB_TOKEN=abc123def456"`,
		`[2026-09-19 09:01:00] user=1 action=shell cmd="grep <div> index.html && echo ok"`,
	}
	got := formatLogReply(`🔍 2 match(es) for "<div>"`, lines)

	if strings.Contains(got, "abc123def456") {
		t.Errorf("secrets written before redaction existed must still be hidden:\n%s", got)
	}
	if strings.Contains(got, "<div>") {
		t.Errorf("header and body must be HTML-escaped for parse_mode=HTML:\n%s", got)
	}
	if !strings.Contains(got, "&lt;div&gt;") || !strings.Contains(got, "<pre>") {
		t.Errorf("expected an escaped <pre> block:\n%s", got)
	}
	if strings.Contains(got, "```") {
		t.Errorf("Markdown fences render literally in HTML mode:\n%s", got)
	}
}
