package menu

import (
	"fmt"
	"os/exec"
	"strings"

	filesvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/files"
)

// openInApp opens a path in the specified application.
func (vm *ViewModel) openInApp(tg TelegramSender, chatID int64, state *State, app string, path string) error {
	_ = tg.EditMessage(chatID, state.MessageID, fmt.Sprintf("Opening <b>%s</b> in %s...", shortenPath(path), app), nil)

	var err error
	switch app {
	case "Finder":
		err = exec.Command("open", path).Run()
	default:
		err = exec.Command("open", "-a", app, path).Run()
	}

	if err != nil {
		return tg.SendMessage(chatID, fmt.Sprintf("❌ Failed to open: %v", err))
	}
	return tg.SendMessage(chatID, fmt.Sprintf("✅ Opened <b>%s</b> in %s", shortenPath(path), app))
}

// readFile reads and sends file contents.
func (vm *ViewModel) readFile(tg TelegramSender, chatID int64, state *State, path string) error {
	_ = tg.EditMessage(chatID, state.MessageID, fmt.Sprintf("Reading %s...", shortenPath(path)), nil)

	content, err := filesvc.Read(path)
	if err != nil {
		return tg.SendMessage(chatID, fmt.Sprintf("❌ Cannot read: %v", err))
	}

	// Truncate if too long for Telegram
	if len(content) > 4000 {
		content = content[:4000] + "\n\n... (truncated)"
	}

	return tg.SendMessage(chatID, fmt.Sprintf("📄 <b>%s</b>\n<pre>%s</pre>", shortenPath(path), content))
}

// copyPath copies the path to clipboard.
func (vm *ViewModel) copyPath(tg TelegramSender, chatID int64, state *State, path string) error {
	_ = tg.EditMessage(chatID, state.MessageID, "Copied!", nil)
	_ = filesvc.SetClipboard(path)
	return tg.SendMessage(chatID, fmt.Sprintf("📋 Copied to clipboard:\n<code>%s</code>", path))
}

// gitAction runs a git command in the given repo path.
func (vm *ViewModel) gitAction(tg TelegramSender, chatID int64, state *State, repoPath string, action string) error {
	_ = tg.EditMessage(chatID, state.MessageID, fmt.Sprintf("Running git %s...", action), nil)

	var out string
	var err error

	switch action {
	case "status":
		out, err = runGit(repoPath, "status", "--short")
		if out == "" {
			out = "Working tree clean ✨"
		}

	case "push":
		// Full push flow: add → commit → push
		_, _ = runGit(repoPath, "add", ".")

		// Get status to check if there's something to commit
		status, _ := runGit(repoPath, "status", "--porcelain")
		if status != "" {
			// Generate a simple commit message from changed files
			msg := generateCommitMessage(status)
			_, _ = runGit(repoPath, "commit", "-m", msg)
		}

		out, err = runGit(repoPath, "push")
		if err != nil {
			out = fmt.Sprintf("Push failed: %v\n%s", err, out)
		} else {
			if out == "" {
				out = "Everything up-to-date"
			}
			out = "✅ Push complete!\n" + out
		}

	case "pull":
		out, err = runGit(repoPath, "pull")
		if err != nil {
			out = fmt.Sprintf("Pull failed: %v\n%s", err, out)
		}
	}

	if err != nil && out == "" {
		out = fmt.Sprintf("❌ git %s failed: %v", action, err)
	}

	return tg.SendMessage(chatID, fmt.Sprintf("🔧 <b>git %s</b>\n<pre>%s</pre>", action, out))
}

// runGit executes a git command in the given directory.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// generateCommitMessage creates a simple commit message from git status.
func generateCommitMessage(status string) string {
	lines := strings.Split(strings.TrimSpace(status), "\n")
	if len(lines) == 1 {
		// Single file: use its name
		parts := strings.Fields(lines[0])
		if len(parts) >= 2 {
			return "update " + parts[len(parts)-1]
		}
	}
	return fmt.Sprintf("update %d files", len(lines))
}
