package mac

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FileInfo describes a file or folder for /info.
type FileInfo struct {
	Path     string
	Name     string
	IsDir    bool
	Size     int64 // for a folder: the total size of its contents
	Items    int   // for a folder: how many entries it holds directly
	Kind     string
	Modified time.Time
	Created  time.Time
}

// Summary renders the file for a chat reply. It is plain text: the view layer
// escapes it, and a file name is exactly the kind of string that would
// otherwise arrive carrying markup of its own.
func (f FileInfo) Summary() string {
	icon, kind := "📄", f.Kind
	if f.IsDir {
		icon = "📁"
		if kind == "" {
			kind = "Folder"
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s %s\n", icon, f.Name)
	if kind != "" {
		sb.WriteString(kind + " · ")
	}
	sb.WriteString(humanBytes(f.Size))
	if f.IsDir {
		fmt.Fprintf(&sb, " · %d item(s)", f.Items)
	}
	sb.WriteString("\n")
	fmt.Fprintf(&sb, "Modified %s\n", f.Modified.Format("2006-01-02 15:04"))
	if !f.Created.IsZero() {
		fmt.Fprintf(&sb, "Created %s\n", f.Created.Format("2006-01-02 15:04"))
	}
	sb.WriteString(f.Path)
	return sb.String()
}

// TrashState is how much is sitting in the Trash.
type TrashState struct {
	Items int
	Size  int64
}

// Summary renders the Trash state for a chat reply.
func (t TrashState) Summary() string {
	if t.Items == 0 {
		return "🗑 The Trash is empty."
	}
	return "🗑 " + t.Describe()
}

// Describe is the bare "12 item(s) · 340 MB", for callers writing their own
// sentence around it. The size is dropped when it could not be measured —
// reading the Trash folder needs Full Disk Access, which the count does not.
func (t TrashState) Describe() string {
	if t.Size <= 0 {
		return fmt.Sprintf("%d item(s)", t.Items)
	}
	return fmt.Sprintf("%d item(s) · %s", t.Items, humanBytes(t.Size))
}

// humanBytes renders a byte count the way a file manager does.
func humanBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	value := float64(n) / 1024
	for _, unit := range []string{"KB", "MB", "GB", "TB"} {
		if value < 1024 {
			if value < 10 {
				return fmt.Sprintf("%.1f %s", value, unit)
			}
			return fmt.Sprintf("%.0f %s", value, unit)
		}
		value /= 1024
	}
	return fmt.Sprintf("%.0f PB", value)
}

// parseDuKilobytes reads the size out of `du -sk <path>` output, which is
// kilobytes and a tab before the path.
func parseDuKilobytes(out string) (int64, error) {
	field := strings.TrimSpace(out)
	if idx := strings.IndexAny(field, " \t"); idx > 0 {
		field = field[:idx]
	}
	kb, err := strconv.ParseInt(field, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("unexpected du output %q", strings.TrimSpace(out))
	}
	return kb * 1024, nil
}
