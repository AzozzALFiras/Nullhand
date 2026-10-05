package command

import (
	"fmt"
	"strings"

	searchsvc "github.com/AzozzALFiras/Nullhand/internal/service/search"
	tgfmt "github.com/AzozzALFiras/Nullhand/internal/view/telegram"
)

// File search and previews. Both work on every platform: macOS answers from
// Spotlight and Quick Look, elsewhere the service walks the tree and renders
// the preview from the file itself.

// findResultLimit is how many paths one /find reply shows.
const findResultLimit = 20

func (vm *ViewModel) find(args []string) Result {
	query, folder := splitFindArgs(args)
	if query == "" {
		return Result{Text: "Usage: /find `name` [in `folder`]"}
	}

	hits, err := searchsvc.Find(query, folder, findResultLimit)
	if err != nil {
		return Result{Text: tgfmt.FailWith("find", err)}
	}
	if len(hits) == 0 {
		return Result{Text: fmt.Sprintf("🔍 Nothing matching %q.", tgfmt.Escape(query))}
	}
	return Result{Text: fmt.Sprintf(
		"🔍 <b>%d match(es) for %q</b>\n%s\n\nSend <code>send me &lt;path&gt;</code> to get a file, or /preview `path` to see it.",
		len(hits), tgfmt.Escape(query), tgfmt.Code(strings.Join(hits, "\n")))}
}

// splitFindArgs separates "report.pdf in ~/Documents" into query and folder.
func splitFindArgs(args []string) (query, folder string) {
	for i, a := range args {
		if strings.EqualFold(a, "in") && i > 0 && i+1 < len(args) {
			return strings.Join(args[:i], " "), strings.Join(args[i+1:], " ")
		}
	}
	return strings.Join(args, " "), ""
}

func (vm *ViewModel) preview(args []string) Result {
	path := strings.TrimSpace(strings.Join(args, " "))
	if path == "" {
		return Result{Text: "Usage: /preview `path to a file`"}
	}

	preview, err := searchsvc.PreviewFile(path)
	if err != nil {
		return Result{Text: tgfmt.FailWith("preview", err)}
	}
	if len(preview.Image) > 0 {
		return Result{ImageData: preview.Image, Text: "🖼 " + path}
	}
	// Nothing renderable, but the file is textual — the excerpt is the preview.
	return Result{Text: fmt.Sprintf("📄 <b>%s</b>\n%s", tgfmt.Escape(path), tgfmt.Code(preview.Text))}
}
