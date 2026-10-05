//go:build darwin

package search

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpotlightScopes(t *testing.T) {
	// The data-volume path comes first: -onlyin silently finds nothing for the
	// short /Users form on APFS.
	got := spotlightScopes("/Users/me/Documents")
	if len(got) != 2 || got[0] != "/System/Volumes/Data/Users/me/Documents" || got[1] != "/Users/me/Documents" {
		t.Errorf("got %q", got)
	}
	if got := spotlightScopes("/System/Volumes/Data/Users/me"); len(got) != 1 {
		t.Errorf("an already-resolved path needs no alternative, got %q", got)
	}
	if got := spotlightScopes(""); got != nil {
		t.Errorf("no folder means no scope, got %q", got)
	}
}

func TestMdfindArgs(t *testing.T) {
	got := strings.Join(mdfindArgs("report.pdf", "/System/Volumes/Data/Users/me"), " ")
	if got != "-onlyin /System/Volumes/Data/Users/me -name report.pdf" {
		t.Errorf("got %q", got)
	}
	if got := strings.Join(mdfindArgs("report.pdf", ""), " "); got != "-name report.pdf" {
		t.Errorf("an unscoped search should search the whole index, got %q", got)
	}
}

// Spotlight has nothing indexed for many developer folders, so this doubles
// as a test of the fallback: the file exists and must be found either way.
func TestFindLocatesFileInRepo(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	hits, err := Find("search_darwin.go", wd, 10)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	found := false
	for _, h := range hits {
		if strings.HasSuffix(h, "search_darwin.go") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find search_darwin.go under %s, got %q", wd, hits)
	}
}

func TestFindRespectsLimit(t *testing.T) {
	wd, _ := os.Getwd()
	hits, err := Find(".go", wd, 2)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(hits) > 2 {
		t.Errorf("limit ignored: got %d hits", len(hits))
	}
}

// On a Mac even a plain text file comes back as an image, because Quick Look
// renders one. The text excerpt is the fallback for files it declines.
func TestPreviewFileUsesQuickLook(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.txt")
	if err := os.WriteFile(path, []byte("hello from nullhand\n"), 0600); err != nil {
		t.Fatal(err)
	}

	preview, err := PreviewFile(path)
	if err != nil {
		t.Fatalf("PreviewFile: %v", err)
	}
	if !bytes.HasPrefix(preview.Image, []byte("\x89PNG\r\n\x1a\n")) {
		t.Errorf("expected a Quick Look PNG, got %d image bytes and text %q", len(preview.Image), preview.Text)
	}
}

func TestPreviewFileRejectsFolder(t *testing.T) {
	if _, err := PreviewFile(t.TempDir()); err == nil {
		t.Error("a folder has no preview to send")
	}
}
