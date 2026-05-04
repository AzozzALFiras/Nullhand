package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeLines(t *testing.T, path string, lines []string, trailingNL bool) {
	t.Helper()
	body := strings.Join(lines, "\n")
	if trailingNL {
		body += "\n"
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
}

func TestTailEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Tail(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("empty file should yield no lines, got %v", got)
	}
}

func TestTailMissingFile(t *testing.T) {
	if _, err := Tail(filepath.Join(t.TempDir(), "nope"), 10); err == nil {
		t.Error("missing file must return error so caller can show a friendly message")
	}
}

func TestTailFewerLinesThanRequested(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	writeLines(t, path, []string{"a", "b", "c"}, true)

	got, err := Tail(path, 10)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b", "c"}
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTailReturnsLastN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	var lines []string
	for i := 1; i <= 100; i++ {
		lines = append(lines, fmt.Sprintf("line-%03d", i))
	}
	writeLines(t, path, lines, true)

	got, err := Tail(path, 5)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"line-096", "line-097", "line-098", "line-099", "line-100"}
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestTailHandlesLargeFileBeyondChunk(t *testing.T) {
	// Force the multi-chunk read path: each line is ~80 bytes, 1000 lines
	// is ~80KB, well past the 16KB chunk size, so this exercises the
	// "keep reading backward" loop.
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	var lines []string
	for i := 1; i <= 1000; i++ {
		lines = append(lines, fmt.Sprintf("[2026-05-03 10:00:00] user=42 action=test idx=%05d %s",
			i, strings.Repeat("x", 30)))
	}
	writeLines(t, path, lines, true)

	got, err := Tail(path, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 50 {
		t.Fatalf("expected 50 lines, got %d", len(got))
	}
	if !strings.Contains(got[0], "idx=00951") || !strings.Contains(got[49], "idx=01000") {
		t.Errorf("multi-chunk tail returned wrong slice: first=%q last=%q", got[0], got[49])
	}

	all, err := readAllLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if !equal(got, all[len(all)-50:]) {
		t.Error("Tail must agree with full-scan reference for the same N")
	}
}

func TestTailFileWithoutTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	writeLines(t, path, []string{"a", "b", "c"}, false)

	got, err := Tail(path, 5)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b", "c"}
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSearchFiltersCaseInsensitively(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	writeLines(t, path, []string{
		"[t1] user=1 action=screenshot",
		"[t2] user=1 action=run cmd=ls",
		"[t3] user=2 action=Screenshot",
		"[t4] user=1 action=type text=hello",
	}, true)

	got, err := Search(path, "screenshot", 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 matches (case-insensitive), got %d: %v", len(got), got)
	}
}

func TestSearchEmptyQueryReturnsTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	writeLines(t, path, []string{"a", "b", "c", "d"}, true)

	got, err := Search(path, "", 100, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"c", "d"}
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestSearchClampsToLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, fmt.Sprintf("action=hit n=%d", i))
	}
	writeLines(t, path, lines, true)

	got, err := Search(path, "hit", 100, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 {
		t.Errorf("expected limit=5, got %d", len(got))
	}
	// Should be the most recent matches.
	if !strings.Contains(got[len(got)-1], "n=29") {
		t.Errorf("most recent match should be n=29, last line was %q", got[len(got)-1])
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
