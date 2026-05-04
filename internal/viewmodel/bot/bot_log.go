package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/AzozzALFiras/Nullhand/internal/audit"
)

// Telegram caps a single text message at ~4096 characters. We round down to
// leave headroom for the header lines, code-block fences, etc.
const logMaxBytes = 3500

// Default tail size when /log is invoked with no numeric argument.
const logDefaultLines = 20

// Hard cap on requested tail size — past this users should download the
// raw file via /recipes-style export rather than trying to chat-paginate it.
const logMaxLines = 200

// Default scan window for /log search before filtering.
const logSearchScan = 500

// handleLogCommand serves three forms:
//
//	/log                     — last logDefaultLines lines
//	/log <N>                 — last N lines, capped at logMaxLines
//	/log search <query…>     — substring filter over the last logSearchScan lines
//
// Output is always wrapped in a Telegram-friendly preformatted block so the
// bracketed timestamps don't get mangled into Markdown emphasis.
func (vm *ViewModel) handleLogCommand(chatID, userID int64, args []string) {
	path, err := audit.Path()
	if err != nil {
		vm.send(chatID, fmt.Sprintf("❌ Could not resolve audit log path: %v", err))
		return
	}

	if len(args) > 0 && strings.EqualFold(args[0], "search") {
		query := strings.TrimSpace(strings.Join(args[1:], " "))
		if query == "" {
			vm.send(chatID, "Usage: /log search <query>")
			return
		}
		vm.replyLogSearch(chatID, userID, path, query)
		return
	}

	n := logDefaultLines
	if len(args) > 0 {
		if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
			n = v
		} else {
			vm.send(chatID, "Usage: /log [N] | /log search <query>")
			return
		}
	}
	if n > logMaxLines {
		n = logMaxLines
	}

	lines, err := audit.Tail(path, n)
	if err != nil {
		vm.send(chatID, fmt.Sprintf("❌ Could not read audit log: %v", err))
		return
	}
	vm.auditLog(userID, "log_view", fmt.Sprintf(`n=%d`, n))
	vm.sendLogReply(chatID, fmt.Sprintf("📜 Last %d audit entries:", len(lines)), lines)
}

func (vm *ViewModel) replyLogSearch(chatID, userID int64, path, query string) {
	lines, err := audit.Search(path, query, logSearchScan, logMaxLines)
	if err != nil {
		vm.send(chatID, fmt.Sprintf("❌ Could not search audit log: %v", err))
		return
	}
	vm.auditLog(userID, "log_search", fmt.Sprintf(`query=%q matches=%d`, query, len(lines)))
	if len(lines) == 0 {
		vm.send(chatID, fmt.Sprintf("🔍 No matches for %q in the last %d entries.", query, logSearchScan))
		return
	}
	header := fmt.Sprintf("🔍 %d match(es) for %q in the last %d entries:", len(lines), query, logSearchScan)
	vm.sendLogReply(chatID, header, lines)
}

// sendLogReply formats lines into a code-block message, trimming OLDEST
// entries first if the body would exceed logMaxBytes — the user almost
// always cares more about the most recent activity than the start of the
// window.
func (vm *ViewModel) sendLogReply(chatID int64, header string, lines []string) {
	if len(lines) == 0 {
		vm.send(chatID, header+"\n(empty)")
		return
	}

	body, dropped := joinWithBudget(lines, logMaxBytes)
	suffix := ""
	if dropped > 0 {
		suffix = fmt.Sprintf("\n…(%d earlier line(s) trimmed to fit Telegram's message limit)", dropped)
	}
	// Code block keeps brackets/equals signs intact and gives a monospace
	// view that matches what users see when they cat the file directly.
	vm.send(chatID, fmt.Sprintf("%s\n```\n%s\n```%s", header, body, suffix))
}

// joinWithBudget joins lines with newlines, dropping from the start until
// the total byte length fits within budget. Returns the joined string plus
// the number of dropped lines.
func joinWithBudget(lines []string, budget int) (string, int) {
	if budget <= 0 {
		return "", len(lines)
	}
	total := 0
	keepFrom := 0
	for i := len(lines) - 1; i >= 0; i-- {
		add := len(lines[i]) + 1 // +1 for the newline
		if total+add > budget {
			keepFrom = i + 1
			break
		}
		total += add
	}
	return strings.Join(lines[keepFrom:], "\n"), keepFrom
}
