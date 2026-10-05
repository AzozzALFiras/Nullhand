//go:build darwin

package mac

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	filesvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/files"
	"github.com/AzozzALFiras/Nullhand/internal/service/proc"
)

// Reveal opens a Finder window with the item selected — the quickest way to
// act on a file found from the phone once you are back at the Mac.
func Reveal(path string) error {
	full, err := filesvc.ExpandPath(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(full); err != nil {
		return err
	}
	_, err = proc.Run(defaultTimeout, "open", "-R", full)
	return err
}

// Info describes a file or folder: kind, size, and when it was created and
// last changed. A folder's size is measured with `du`, which is far faster
// than walking the tree from Go.
func Info(path string) (FileInfo, error) {
	full, err := filesvc.ExpandPath(path)
	if err != nil {
		return FileInfo{}, err
	}
	stat, err := os.Stat(full)
	if err != nil {
		return FileInfo{}, err
	}

	info := FileInfo{
		Path:     full,
		Name:     filepath.Base(full),
		IsDir:    stat.IsDir(),
		Size:     stat.Size(),
		Modified: stat.ModTime(),
		Created:  birthTime(stat),
		Kind:     fileKind(full),
	}

	if stat.IsDir() {
		if entries, err := os.ReadDir(full); err == nil {
			info.Items = len(entries)
		}
		if size, err := directorySize(full); err == nil {
			info.Size = size
		}
	}
	return info, nil
}

// TrashStatus reports what is in the Trash.
//
// The count comes from Finder rather than from reading ~/.Trash directly:
// macOS protects that folder, so a plain read fails with "operation not
// permitted" unless the host app was granted Full Disk Access, while Finder
// answers with the Automation permission that emptying the Trash needs
// anyway. The size is measured if the sandbox allows it and left out if not.
func TrashStatus() (TrashState, error) {
	out, err := osa(`tell application "Finder" to count items of trash`)
	if err != nil {
		return TrashState{}, err
	}
	items, convErr := strconv.Atoi(strings.TrimSpace(out))
	if convErr != nil {
		return TrashState{}, fmt.Errorf("unexpected trash count %q", out)
	}

	state := TrashState{Items: items}
	if items > 0 {
		if dir, err := trashDir(); err == nil {
			if size, err := directorySize(dir); err == nil {
				state.Size = size
			}
		}
	}
	return state, nil
}

// EmptyTrash empties the Trash. This cannot be undone, so callers gate it
// behind a confirmation.
func EmptyTrash() error {
	_, err := osa(`tell application "Finder" to empty trash`)
	return err
}

// MoveToTrash moves a file or folder to the Trash. Unlike rm this is
// recoverable, which makes it the better answer to "delete this" from a phone.
func MoveToTrash(path string) error {
	full, err := filesvc.ExpandPath(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(full); err != nil {
		return err
	}
	_, err = osa(fmt.Sprintf(`tell application "Finder" to delete POSIX file %s`, QuoteAppleScript(full)))
	return err
}

// trashDir returns the user's Trash folder.
func trashDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".Trash"), nil
}

// directorySize totals a folder with `du -sk`.
func directorySize(dir string) (int64, error) {
	out, err := proc.Run(30*time.Second, "du", "-sk", dir)
	if err != nil && out == "" {
		return 0, err
	}
	return parseDuKilobytes(out)
}

// fileKind asks Spotlight for the human-readable kind ("PDF document",
// "Folder"). It is metadata only, so an unindexed file simply has no kind.
func fileKind(path string) string {
	out, err := proc.Run(defaultTimeout, "mdls", "-name", "kMDItemKind", "-raw", path)
	if err != nil || out == "(null)" {
		return ""
	}
	return out
}

// birthTime extracts the creation time, which Go's FileInfo does not expose.
func birthTime(stat os.FileInfo) time.Time {
	sys, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Time{}
	}
	return time.Unix(sys.Birthtimespec.Sec, sys.Birthtimespec.Nsec)
}
