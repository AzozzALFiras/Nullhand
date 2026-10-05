package search

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// minimalPNG is a 1x1 PNG, enough to check that image bytes pass through
// untouched.
var minimalPNG = []byte{
	0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0a, 'I', 'D', 'A', 'T',
	0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05,
	0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00,
	0x00, 0x00, 'I', 'E', 'N', 'D', 0xae, 0x42, 0x60, 0x82,
}

// minimalPDF is a one-page PDF. poppler warns about the missing xref table
// but still renders it, which is all this needs.
const minimalPDF = "%PDF-1.4\n" +
	"1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n" +
	"2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n" +
	"3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]>>endobj\n" +
	"trailer<</Root 1 0 R>>\n"

func writeFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFindArgs(t *testing.T) {
	got := findArgs("report", "/Users/me")
	if got[0] != "/Users/me" {
		t.Errorf("find must start at the folder, got %q", got[0])
	}
	joined := strings.Join(got, " ")
	for _, want := range []string{"-maxdepth 6", "-iname *report*", "-not -path */.*"} {
		if !strings.Contains(joined, want) {
			t.Errorf("args %q missing %q", joined, want)
		}
	}
}

func TestLimitResults(t *testing.T) {
	ten := make([]string, 10)
	if got := limitResults(ten, 3); len(got) != 3 {
		t.Errorf("got %d results, want 3", len(got))
	}
	if got := limitResults(ten, 0); len(got) != 10 {
		t.Errorf("a non-positive limit falls back to the default, got %d", len(got))
	}
}

// The `find` walk is the portable search path, and the only one off macOS.
func TestWalkFindLocatesFile(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	hits, err := walkFind("search_generic.go", wd, 10)
	if err != nil {
		t.Fatalf("walkFind: %v", err)
	}
	found := false
	for _, h := range hits {
		if strings.HasSuffix(h, "search_generic.go") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected to find search_generic.go under %s, got %q", wd, hits)
	}
}

func TestWalkFindSkipsDotDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".hidden"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".hidden", "secret.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "visible.txt"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}

	hits, err := walkFind(".txt", dir, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range hits {
		if strings.Contains(h, ".hidden") {
			t.Errorf("dot-directories should be skipped, got %q", h)
		}
	}
	if len(hits) != 1 {
		t.Errorf("expected only the visible file, got %q", hits)
	}
}

func TestFindRejectsEmptyQuery(t *testing.T) {
	if _, err := Find("", "", 10); err == nil {
		t.Error("an empty query must return an error instead of listing everything")
	}
}

func TestPreviewImagePassesThrough(t *testing.T) {
	path := writeFile(t, "dot.png", minimalPNG)
	preview, err := previewFallback(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(preview.Image, minimalPNG) {
		t.Error("an image Telegram accepts should be sent as it is")
	}
	if preview.Text != "" {
		t.Errorf("no text excerpt expected, got %q", preview.Text)
	}
}

func TestPreviewRejectsOversizedImage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "huge.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	// Sparse file: no bytes written, but the size check sees it.
	if err := f.Truncate(maxImageBytes + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()

	if _, err := previewFallback(path); err == nil {
		t.Error("an image over Telegram's photo limit must be refused with an explanation")
	}
}

func TestPreviewRendersPDF(t *testing.T) {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm not installed")
	}
	path := writeFile(t, "doc.pdf", []byte(minimalPDF))

	preview, err := previewFallback(path)
	if err != nil {
		t.Fatalf("previewFallback: %v", err)
	}
	if !bytes.HasPrefix(preview.Image, []byte("\x89PNG\r\n\x1a\n")) {
		t.Errorf("expected a PNG of the first page, got %d bytes", len(preview.Image))
	}
}

func TestPreviewTextExcerpt(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 200; i++ {
		sb.WriteString("line of a long text file\n")
	}
	path := writeFile(t, "notes.txt", []byte(sb.String()))

	preview, err := previewFallback(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Image) != 0 {
		t.Error("a text file has no image to send")
	}
	if !strings.Contains(preview.Text, "truncated") {
		t.Error("a long excerpt should say it was cut")
	}
	if lines := strings.Count(preview.Text, "\n") + 1; lines > maxTextLines+2 {
		t.Errorf("excerpt has %d lines, want at most %d", lines, maxTextLines)
	}
}

// Files with no familiar extension are judged by content, so source files,
// .conf and .log all preview as text.
func TestPreviewUnknownExtensionWithTextContent(t *testing.T) {
	path := writeFile(t, "sshd_config.bak", []byte("Port 22\nPermitRootLogin no\n"))
	preview, err := previewFallback(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview.Text, "PermitRootLogin") {
		t.Errorf("got %q", preview.Text)
	}
}

func TestPreviewRejectsBinary(t *testing.T) {
	path := writeFile(t, "blob.bin", []byte{0x00, 0x01, 0x02, 0xff, 0xfe})
	_, err := previewFallback(path)
	if !errors.Is(err, ErrNoPreview) {
		t.Errorf("expected ErrNoPreview for binary content, got %v", err)
	}
}

func TestPreviewRejectsFolderAndMissingFile(t *testing.T) {
	err := func() error { _, err := previewFallback(t.TempDir()); return err }()
	if err == nil || !strings.Contains(err.Error(), "folder") {
		t.Errorf("a folder should be reported as such, got %v", err)
	}
	if _, err := previewFallback(filepath.Join(t.TempDir(), "nope.pdf")); err == nil {
		t.Error("a missing file must be reported before any renderer runs")
	}
}

func TestPreviewFileExpandsHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, "note.txt"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}

	preview, err := PreviewFile("~/note.txt")
	if err != nil {
		t.Fatalf("PreviewFile: %v", err)
	}
	if preview.Text == "" && len(preview.Image) == 0 {
		t.Error("expected either an excerpt or an image")
	}
}

func TestConvertibleImageWithoutToolExplainsItself(t *testing.T) {
	for _, tool := range []string{"magick", "convert"} {
		if _, err := exec.LookPath(tool); err == nil {
			t.Skip("ImageMagick is installed; the missing-tool path cannot be exercised")
		}
	}
	path := writeFile(t, "photo.heic", []byte("not really heic"))
	_, err := previewFallback(path)
	if err == nil || !strings.Contains(err.Error(), "ImageMagick") {
		t.Errorf("the error should name the missing tool, got %v", err)
	}
}
