// Package search finds files and renders previews of them.
//
// macOS has Spotlight and Quick Look, so it uses those; everywhere else — and
// on macOS whenever Spotlight has nothing indexed for the folder — the search
// walks the tree with `find` and the preview is rendered from the file itself
// (images as they are, PDFs through pdftoppm, anything textual as an excerpt).
package search

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	filesvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/files"
	"github.com/AzozzALFiras/Nullhand/internal/service/proc"
)

// Preview is a renderable view of a file: an image when one can be produced,
// otherwise a text excerpt.
type Preview struct {
	Image []byte // PNG/JPEG bytes, ready to send as a photo
	Text  string // excerpt, when the file is textual and no image was rendered
}

// ErrNoPreview means the file can be shown neither as an image nor as text.
var ErrNoPreview = errors.New("no preview available for this file type")

// errEmptyQuery guards against a bare /find listing the whole disk.
var errEmptyQuery = errors.New("search query is empty")

const (
	// findTimeout bounds the filesystem walk.
	findTimeout = 45 * time.Second
	// renderTimeout bounds one preview renderer (qlmanage, pdftoppm, magick).
	renderTimeout = 20 * time.Second
	// maxImageBytes is about Telegram's limit for a photo.
	maxImageBytes = 9 << 20
	// maxTextBytes and maxTextLines bound a text excerpt.
	maxTextBytes = 3000
	maxTextLines = 40
	// defaultLimit caps results when the caller asks for no limit.
	defaultLimit = 20
)

// sendableImages can go straight to Telegram as photo bytes.
var sendableImages = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".bmp": true,
}

// convertibleImages need a converter before they can be sent.
var convertibleImages = map[string]bool{
	".heic": true, ".heif": true, ".tiff": true, ".tif": true, ".svg": true, ".ico": true, ".avif": true,
}

// findArgs builds the `find` invocation. It stays shallow and skips
// dot-directories so it returns in reasonable time on a large home folder.
func findArgs(query, dir string) []string {
	return []string{
		dir, "-maxdepth", "6",
		"-not", "-path", "*/.*",
		"-iname", "*" + query + "*",
	}
}

// limitResults trims a result list (limit <= 0 means the default, not
// unbounded output).
func limitResults(paths []string, limit int) []string {
	if limit <= 0 {
		limit = defaultLimit
	}
	if len(paths) > limit {
		return paths[:limit]
	}
	return paths
}

// walkFind is the search that works everywhere: a bounded `find` walk.
func walkFind(query, dir string, limit int) ([]string, error) {
	// find exits non-zero when it meets an unreadable directory; whatever it
	// printed before that is still worth showing.
	out, err := proc.Run(findTimeout, "find", findArgs(query, dir)...)
	hits := proc.Lines(out)
	if len(hits) == 0 && err != nil {
		return nil, err
	}
	return limitResults(hits, limit), nil
}

// previewFallback renders a preview without Quick Look. It is the only
// renderer off macOS, and the safety net on macOS when Quick Look declines.
func previewFallback(path string) (Preview, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Preview{}, err
	}
	if info.IsDir() {
		return Preview{}, fmt.Errorf("%q is a folder — send \"browse %s\" to list it instead",
			filepath.Base(path), path)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch {
	case sendableImages[ext]:
		if info.Size() > maxImageBytes {
			return Preview{}, fmt.Errorf("%s is too big to send as a photo (%d MB)",
				filepath.Base(path), info.Size()>>20)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return Preview{}, err
		}
		return Preview{Image: data}, nil

	case ext == ".pdf":
		return renderPDF(path)

	case convertibleImages[ext]:
		return convertImage(path)
	}

	if excerpt, ok := textExcerpt(path); ok {
		return Preview{Text: excerpt}, nil
	}
	return Preview{}, fmt.Errorf("%w: %s", ErrNoPreview, filepath.Base(path))
}

// renderPDF turns the first page into a PNG.
func renderPDF(path string) (Preview, error) {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return Preview{}, fmt.Errorf("previewing a PDF needs pdftoppm (install poppler-utils)")
	}
	dir, err := os.MkdirTemp("", "nullhand-preview-")
	if err != nil {
		return Preview{}, fmt.Errorf("preview: temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	// pdftoppm appends its own page suffix to the prefix we give it.
	if _, err := proc.Run(renderTimeout, "pdftoppm",
		"-png", "-r", "100", "-f", "1", "-l", "1", path, filepath.Join(dir, "page")); err != nil {
		return Preview{}, err
	}
	return firstPNG(dir, filepath.Base(path))
}

// convertImage renders formats Telegram will not take (HEIC from an iPhone,
// TIFF, SVG) through ImageMagick, if it is installed.
func convertImage(path string) (Preview, error) {
	tool := ""
	for _, candidate := range []string{"magick", "convert"} {
		if _, err := exec.LookPath(candidate); err == nil {
			tool = candidate
			break
		}
	}
	if tool == "" {
		return Preview{}, fmt.Errorf("previewing %s needs ImageMagick (install imagemagick)",
			strings.TrimPrefix(filepath.Ext(path), "."))
	}

	dir, err := os.MkdirTemp("", "nullhand-preview-")
	if err != nil {
		return Preview{}, fmt.Errorf("preview: temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	out := filepath.Join(dir, "page.png")
	// "1600x1600>" only shrinks, so a small image is not blown up.
	if _, err := proc.Run(renderTimeout, tool, path+"[0]", "-resize", "1600x1600>", out); err != nil {
		return Preview{}, err
	}
	return firstPNG(dir, filepath.Base(path))
}

// firstPNG reads the single PNG a renderer left in dir. Renderers can exit
// successfully having produced nothing, so the miss gets its own message.
func firstPNG(dir, name string) (Preview, error) {
	matches, _ := filepath.Glob(filepath.Join(dir, "*.png"))
	if len(matches) == 0 {
		return Preview{}, fmt.Errorf("%w: %s", ErrNoPreview, name)
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		return Preview{}, fmt.Errorf("preview: read render: %w", err)
	}
	return Preview{Image: data}, nil
}

// textExcerpt returns the first lines of a textual file. ok is false for
// binaries, which are detected by a NUL byte or invalid UTF-8 rather than by
// extension — that way .conf, .log and unfamiliar source files still work.
func textExcerpt(path string) (string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer f.Close()

	buf := make([]byte, maxTextBytes)
	n, err := io.ReadFull(f, buf)
	if n == 0 || (err != nil && err != io.ErrUnexpectedEOF && err != io.EOF) {
		return "", false
	}
	data := buf[:n]
	if bytes.IndexByte(data, 0) >= 0 {
		return "", false
	}
	// The read may stop mid-character; drop that fragment before judging.
	data = bytes.ToValidUTF8(data, nil)
	if !utf8.Valid(data) || len(data) == 0 {
		return "", false
	}

	lines := strings.Split(string(data), "\n")
	truncated := len(lines) > maxTextLines
	if truncated {
		lines = lines[:maxTextLines]
	}
	excerpt := strings.TrimRight(strings.Join(lines, "\n"), "\n")
	if excerpt == "" {
		return "", false
	}
	if truncated || n == maxTextBytes {
		excerpt += "\n… (truncated)"
	}
	return excerpt, true
}

// expandDir resolves the folder a search starts from.
func expandDir(dir string) (string, error) { return filesvc.ExpandDir(dir) }

// expandPath resolves the file a preview is asked for.
func expandPath(path string) (string, error) { return filesvc.ExpandPath(path) }
