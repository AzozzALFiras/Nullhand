package agent

import (
	"fmt"
	"strconv"
	"strings"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
	searchsvc "github.com/AzozzALFiras/Nullhand/internal/service/search"
)

// File search and preview tools. Unlike the macOS tools these are offered on
// every platform — the service picks Spotlight and Quick Look on a Mac and a
// filesystem walk elsewhere.

// executeSearchTool handles the file search tools. handled is false when the
// tool name belongs to someone else.
func (vm *ViewModel) executeSearchTool(tc aimodel.ToolCall, sendPhoto PhotoFunc) (parts []aimodel.MessagePart, err error, handled bool) {
	args := tc.Arguments

	switch tc.ToolName {
	case "find_files":
		query := strings.TrimSpace(args["query"])
		if query == "" {
			e := fmt.Errorf("find_files needs a query")
			return failParts(e), e, true
		}
		limit, _ := strconv.Atoi(args["limit"])
		hits, err := searchsvc.Find(query, args["folder"], limit)
		if err != nil {
			return failParts(err), err, true
		}
		if len(hits) == 0 {
			return infoParts(fmt.Sprintf("No files matching %q.", query)), nil, true
		}
		return infoParts(fmt.Sprintf("%d match(es) for %q:\n%s", len(hits), query, strings.Join(hits, "\n"))), nil, true

	case "preview_file":
		path := strings.TrimSpace(args["path"])
		preview, err := searchsvc.PreviewFile(path)
		if err != nil {
			return failParts(err), err, true
		}
		// A textual file has no image to deliver; its excerpt is the answer.
		if len(preview.Image) == 0 {
			return infoParts(fmt.Sprintf("%s:\n%s", path, preview.Text)), nil, true
		}
		if sendPhoto == nil {
			e := fmt.Errorf("preview delivery is not available in this context")
			return failParts(e), e, true
		}
		if err := sendPhoto(preview.Image, "🖼 "+path); err != nil {
			return failParts(err), err, true
		}
		return textParts("preview delivered to the user (the AI cannot see it)"), nil, true
	}

	return nil, nil, false
}

// searchToolDefinitions describes the file search tools for the AI.
func searchToolDefinitions() []aimodel.ToolDefinition {
	return []aimodel.ToolDefinition{
		{
			Name: "find_files",
			Description: "Find files by name. Uses Spotlight on macOS and a bounded filesystem " +
				"walk elsewhere (or when the folder is not indexed). Returns matching paths.",
			Parameters: []aimodel.ToolParameter{
				{Name: "query", Type: "string", Description: "Part of the file name", Required: true},
				{Name: "folder", Type: "string", Description: "Folder to search (default: the home folder)"},
				{Name: "limit", Type: "string", Description: "Maximum number of results (default 20)"},
			},
		},
		{
			Name: "preview_file",
			Description: "Preview a file: an image, PDF page or document is delivered to the user's " +
				"Telegram chat as a photo (you will NOT see it), and a text file comes back as an excerpt.",
			Parameters: []aimodel.ToolParameter{
				{Name: "path", Type: "string", Description: "Path to the file", Required: true},
			},
		},
	}
}
