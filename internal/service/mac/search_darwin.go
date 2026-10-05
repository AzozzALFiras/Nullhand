//go:build darwin

package mac

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// findTimeout bounds the `find` fallback, which walks the filesystem.
const findTimeout = 45 * time.Second

// Find returns paths of files whose name contains query.
//
// Spotlight answers first because it is instant. When it returns nothing the
// search falls back to `find`: on a Mac where the folder was never indexed —
// or was excluded in System Settings → Spotlight → Privacy — mdfind reports no
// hits at all, which would otherwise look like "the file does not exist".
func Find(query, folder string, limit int) ([]string, error) {
	if query == "" {
		return nil, fmt.Errorf("search query is empty")
	}
	dir, err := expandDir(folder)
	if err != nil {
		return nil, err
	}

	scope := firstExistingPath(spotlightScopes(dir))
	if out, err := run(defaultTimeout, "mdfind", mdfindArgs(query, scope)...); err == nil {
		if hits := nonEmptyLines(out); len(hits) > 0 {
			return limitResults(hits, limit), nil
		}
	}

	// find exits non-zero when it meets an unreadable directory; whatever it
	// printed before that is still worth showing.
	out, findErr := run(findTimeout, "find", findArgs(query, dir)...)
	hits := nonEmptyLines(out)
	if len(hits) == 0 && findErr != nil {
		return nil, findErr
	}
	return limitResults(hits, limit), nil
}

// Preview renders a Quick Look thumbnail of any file macOS can preview (PDF,
// image, document, source file) as PNG bytes, ready to send as a photo.
func Preview(path string) ([]byte, error) {
	full, err := expandPath(path)
	if err != nil {
		return nil, err
	}
	if info, err := os.Stat(full); err != nil {
		return nil, fmt.Errorf("preview %q: %w", path, err)
	} else if info.IsDir() {
		return nil, fmt.Errorf("preview: %q is a folder — send \"browse %s\" to list it instead", path, path)
	}

	dir, err := os.MkdirTemp("", "nullhand-preview-")
	if err != nil {
		return nil, fmt.Errorf("preview: temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	if _, err := run(defaultTimeout, "qlmanage", "-t", "-s", "1200", "-o", dir, full); err != nil {
		return nil, err
	}

	// qlmanage names the thumbnail "<original name>.png", but silently
	// produces nothing for types it cannot render — hence the glob plus an
	// explicit error rather than a confusing "file not found".
	matches, _ := filepath.Glob(filepath.Join(dir, "*.png"))
	if len(matches) == 0 {
		return nil, fmt.Errorf("macOS has no Quick Look preview for %q", filepath.Base(full))
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		return nil, fmt.Errorf("preview: read thumbnail: %w", err)
	}
	return data, nil
}

// firstExistingPath returns the first path that exists, or "" if none do.
func firstExistingPath(paths []string) string {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
