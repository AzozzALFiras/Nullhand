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
