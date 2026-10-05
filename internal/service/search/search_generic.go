//go:build !darwin

package search

// Off macOS there is no Spotlight index and no Quick Look, so both operations
// use the portable paths: a bounded `find` walk, and a preview rendered from
// the file itself. Nothing here needs a package installed — pdftoppm and
// ImageMagick only widen what can be previewed.

// Find returns paths of files whose name contains query, searching folder
// (the home directory by default) with a bounded filesystem walk.
func Find(query, folder string, limit int) ([]string, error) {
	if query == "" {
		return nil, errEmptyQuery
	}
	dir, err := expandDir(folder)
	if err != nil {
		return nil, err
	}
	return walkFind(query, dir, limit)
}

// PreviewFile renders a preview of a file: images as they are, PDFs through
// pdftoppm, and anything textual as an excerpt.
func PreviewFile(path string) (Preview, error) {
	full, err := expandPath(path)
	if err != nil {
		return Preview{}, err
	}
	return previewFallback(full)
}
