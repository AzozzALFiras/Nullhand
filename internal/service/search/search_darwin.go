//go:build darwin

package search

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AzozzALFiras/Nullhand/internal/service/proc"
)

// spotlightTimeout is short: mdfind answers from its index or not at all.
const spotlightTimeout = 15 * time.Second

// Find returns paths of files whose name contains query.
//
// Spotlight answers first because it is instant. When it returns nothing the
// search falls back to the `find` walk: on a Mac where the folder was never
// indexed — or was excluded in System Settings → Spotlight → Privacy — mdfind
// reports no hits at all, which would otherwise read as "the file does not
// exist".
func Find(query, folder string, limit int) ([]string, error) {
	if query == "" {
		return nil, errEmptyQuery
	}
	dir, err := expandDir(folder)
	if err != nil {
		return nil, err
	}

	scope := firstExistingPath(spotlightScopes(dir))
	if out, err := proc.Run(spotlightTimeout, "mdfind", mdfindArgs(query, scope)...); err == nil {
		if hits := proc.Lines(out); len(hits) > 0 {
			return limitResults(hits, limit), nil
		}
	}
	return walkFind(query, dir, limit)
}

// PreviewFile renders a preview of any file macOS can show (PDF, image,
// document, source file) as PNG bytes, ready to send as a photo. Quick Look
// covers almost everything; the shared fallback catches the rest.
func PreviewFile(path string) (Preview, error) {
	full, err := expandPath(path)
	if err != nil {
		return Preview{}, err
	}
	if info, statErr := os.Stat(full); statErr != nil {
		return Preview{}, statErr
	} else if info.IsDir() {
		return previewFallback(full) // reports the folder consistently
	}

	dir, err := os.MkdirTemp("", "nullhand-preview-")
	if err != nil {
		return Preview{}, err
	}
	defer os.RemoveAll(dir)

	if _, err := proc.Run(renderTimeout, "qlmanage", "-t", "-s", "1200", "-o", dir, full); err == nil {
		// qlmanage exits successfully for types it cannot render, leaving the
		// output folder empty — so a miss falls through rather than erroring.
		if preview, err := firstPNG(dir, filepath.Base(full)); err == nil {
			return preview, nil
		}
	}
	return previewFallback(full)
}

// spotlightScopes lists the paths to try for mdfind's -onlyin, best first.
// On APFS the home directory is a firmlink: everyone types /Users/me, but the
// index stores /System/Volumes/Data/Users/me and -onlyin silently returns
// nothing for the short form — the caller uses the first path that exists.
func spotlightScopes(dir string) []string {
	if dir == "" {
		return nil
	}
	const dataVolume = "/System/Volumes/Data"
	if strings.HasPrefix(dir, dataVolume) {
		return []string{dir}
	}
	return []string{filepath.Join(dataVolume, dir), dir}
}

// mdfindArgs builds a filename search, scoped to one folder when scope is set.
func mdfindArgs(query, scope string) []string {
	args := []string{}
	if scope != "" {
		args = append(args, "-onlyin", scope)
	}
	return append(args, "-name", query)
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
