package menu

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tgsvc "github.com/AzozzALFiras/nullhand/internal/service/telegram"
)

const maxButtons = 20 // max items per page to avoid Telegram limits

// BrowsePath opens a file browser menu at the given path.
func (vm *ViewModel) BrowsePath(tg TelegramSender, chatID int64, rawPath string) error {
	path := expandPath(rawPath)

	entries, err := listDir(path)
	if err != nil {
		return fmt.Errorf("cannot list %s: %w", path, err)
	}

	text, keyboard := buildBrowseView(path, entries)

	msgID, err := tg.SendMessageWithKeyboard(chatID, text, keyboard)
	if err != nil {
		return err
	}

	vm.SetState(chatID, &State{
		Type:        "file_browser",
		CurrentPath: path,
		History:     []string{},
		MessageID:   msgID,
	})
	return nil
}

// HandleBrowseCallback handles a button press in the file browser.
func (vm *ViewModel) HandleBrowseCallback(tg TelegramSender, chatID int64, callbackID string, data string) error {
	_ = tg.AnswerCallbackQuery(callbackID, "")

	state := vm.GetState(chatID)
	if state == nil {
		return nil
	}

	switch {
	case data == "menu:back":
		return vm.navigateBack(tg, chatID, state)

	case data == "menu:close":
		vm.ClearState(chatID)
		return tg.EditMessage(chatID, state.MessageID, "Menu closed.", nil)

	case strings.HasPrefix(data, "browse:"):
		name := strings.TrimPrefix(data, "browse:")
		newPath := filepath.Join(state.CurrentPath, name)
		return vm.navigateTo(tg, chatID, state, newPath)

	case strings.HasPrefix(data, "action:"):
		return vm.handleAction(tg, chatID, state, data)
	}

	return nil
}

// HandleNumberSelection handles when user types a number while menu is active.
func (vm *ViewModel) HandleNumberSelection(tg TelegramSender, chatID int64, num int) error {
	state := vm.GetState(chatID)
	if state == nil || state.Type != "file_browser" {
		return fmt.Errorf("no active menu")
	}

	entries, err := listDir(state.CurrentPath)
	if err != nil {
		return err
	}

	idx := num - 1
	if idx < 0 || idx >= len(entries) {
		return fmt.Errorf("invalid selection: %d", num)
	}

	entry := entries[idx]
	newPath := filepath.Join(state.CurrentPath, entry.name)

	if entry.isDir {
		return vm.navigateTo(tg, chatID, state, newPath)
	}

	// File selected - show action menu
	return vm.showFileActions(tg, chatID, state, newPath)
}

func (vm *ViewModel) navigateTo(tg TelegramSender, chatID int64, state *State, newPath string) error {
	info, err := os.Stat(newPath)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return vm.showFileActions(tg, chatID, state, newPath)
	}

	entries, err := listDir(newPath)
	if err != nil {
		return err
	}

	// Push current path to history
	state.History = append(state.History, state.CurrentPath)
	state.CurrentPath = newPath

	// Check if it's a git repo or project
	isGit := isGitRepo(newPath)
	text, keyboard := buildBrowseViewWithActions(newPath, entries, isGit)

	vm.SetState(chatID, state)
	return tg.EditMessage(chatID, state.MessageID, text, keyboard)
}

func (vm *ViewModel) navigateBack(tg TelegramSender, chatID int64, state *State) error {
	if len(state.History) == 0 {
		vm.ClearState(chatID)
		return tg.EditMessage(chatID, state.MessageID, "Menu closed.", nil)
	}

	// Pop from history
	prev := state.History[len(state.History)-1]
	state.History = state.History[:len(state.History)-1]
	state.CurrentPath = prev

	entries, err := listDir(prev)
	if err != nil {
		return err
	}

	isGit := isGitRepo(prev)
	text, keyboard := buildBrowseViewWithActions(prev, entries, isGit)

	vm.SetState(chatID, state)
	return tg.EditMessage(chatID, state.MessageID, text, keyboard)
}

func (vm *ViewModel) showFileActions(tg TelegramSender, chatID int64, state *State, filePath string) error {
	text := fmt.Sprintf("📄 <b>%s</b>\n\nChoose action:", filepath.Base(filePath))

	rows := [][]tgsvc.InlineKeyboardButton{
		{{Text: "📖 Read File", CallbackData: "action:read:" + filePath}},
		{{Text: "📋 Copy Path", CallbackData: "action:copy:" + filePath}},
		{{Text: "⬆️ Back", CallbackData: "menu:back"}},
	}

	keyboard := &tgsvc.InlineKeyboardMarkup{InlineKeyboard: rows}
	return tg.EditMessage(chatID, state.MessageID, text, keyboard)
}

func (vm *ViewModel) handleAction(tg TelegramSender, chatID int64, state *State, data string) error {
	parts := strings.SplitN(data, ":", 3)
	if len(parts) < 3 {
		return nil
	}
	action := parts[1]
	target := parts[2]

	vm.ClearState(chatID)

	switch action {
	case "vscode":
		return vm.openInApp(tg, chatID, state, "Visual Studio Code", target)
	case "terminal":
		return vm.openInApp(tg, chatID, state, "Terminal", target)
	case "finder":
		return vm.openInApp(tg, chatID, state, "Finder", target)
	case "cursor":
		return vm.openInApp(tg, chatID, state, "Cursor", target)
	case "read":
		return vm.readFile(tg, chatID, state, target)
	case "copy":
		return vm.copyPath(tg, chatID, state, target)
	case "gitstatus":
		return vm.gitAction(tg, chatID, state, target, "status")
	case "gitpush":
		return vm.gitAction(tg, chatID, state, target, "push")
	case "gitpull":
		return vm.gitAction(tg, chatID, state, target, "pull")
	}
	return nil
}

// ── View builders ──────────────────────────────────────────────────────

func buildBrowseView(path string, entries []dirEntry) (string, *tgsvc.InlineKeyboardMarkup) {
	return buildBrowseViewWithActions(path, entries, isGitRepo(path))
}

func buildBrowseViewWithActions(path string, entries []dirEntry, isGit bool) (string, *tgsvc.InlineKeyboardMarkup) {
	shortPath := shortenPath(path)
	text := fmt.Sprintf("📂 <b>%s</b>", shortPath)
	if isGit {
		text += " (git)"
	}

	var rows [][]tgsvc.InlineKeyboardButton

	// Entry buttons (folders and files)
	for i, e := range entries {
		if i >= maxButtons {
			text += fmt.Sprintf("\n\n<i>... and %d more items</i>", len(entries)-maxButtons)
			break
		}
		icon := "📄"
		if e.isDir {
			icon = "📁"
		}
		rows = append(rows, []tgsvc.InlineKeyboardButton{
			{Text: fmt.Sprintf("%s %s", icon, e.name), CallbackData: "browse:" + e.name},
		})
	}

	// Action buttons for current folder
	actionRow := []tgsvc.InlineKeyboardButton{
		{Text: "💻 VS Code", CallbackData: "action:vscode:" + path},
		{Text: "🔧 Terminal", CallbackData: "action:terminal:" + path},
	}
	rows = append(rows, actionRow)

	actionRow2 := []tgsvc.InlineKeyboardButton{
		{Text: "📂 Finder", CallbackData: "action:finder:" + path},
	}
	rows = append(rows, actionRow2)

	// Git actions if applicable
	if isGit {
		gitRow := []tgsvc.InlineKeyboardButton{
			{Text: "🔄 Git Status", CallbackData: "action:gitstatus:" + path},
			{Text: "⬆️ Git Push", CallbackData: "action:gitpush:" + path},
			{Text: "⬇️ Git Pull", CallbackData: "action:gitpull:" + path},
		}
		rows = append(rows, gitRow)
	}

	// Navigation
	navRow := []tgsvc.InlineKeyboardButton{
		{Text: "⬆️ Back", CallbackData: "menu:back"},
		{Text: "❌ Close", CallbackData: "menu:close"},
	}
	rows = append(rows, navRow)

	return text, &tgsvc.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// ── Helpers ────────────────────────────────────────────────────────────

type dirEntry struct {
	name  string
	isDir bool
}

func listDir(path string) ([]dirEntry, error) {
	items, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var entries []dirEntry
	for _, item := range items {
		name := item.Name()
		if strings.HasPrefix(name, ".") {
			continue // skip hidden files
		}
		entries = append(entries, dirEntry{name: name, isDir: item.IsDir()})
	}

	// Sort: directories first, then files
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].isDir != entries[j].isDir {
			return entries[i].isDir
		}
		return entries[i].name < entries[j].name
	})

	return entries, nil
}

func isGitRepo(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

func expandPath(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "~/") || raw == "~" {
		home, _ := os.UserHomeDir()
		if raw == "~" {
			return home
		}
		return filepath.Join(home, raw[2:])
	}
	if !filepath.IsAbs(raw) {
		home, _ := os.UserHomeDir()
		// Common shorthand: "documents" → ~/Documents
		lower := strings.ToLower(raw)
		switch lower {
		case "documents", "docs", "المستندات":
			return filepath.Join(home, "Documents")
		case "desktop", "سطح المكتب":
			return filepath.Join(home, "Desktop")
		case "downloads", "التنزيلات":
			return filepath.Join(home, "Downloads")
		default:
			return filepath.Join(home, raw)
		}
	}
	return raw
}

func shortenPath(path string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
