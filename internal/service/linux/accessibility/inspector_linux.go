//go:build linux

package accessibility

import (
	"fmt"
	"os/exec"
	"strings"
)

// ListElements returns a compact summary of the AT-SPI2 accessibility tree of
// the currently frontmost application. Each line is:
//
//	<indent>role | name | description | value
//
// maxDepth caps recursion to avoid 10 KB walls of text. A depth of 6–10 is
// normally enough to detect whether an app exposes real AT-SPI content.
//
// Requires AT-SPI2: sudo apt install python3-pyatspi at-spi2-core
func ListElements(maxDepth int) (string, error) {
	if maxDepth <= 0 {
		maxDepth = 8
	}

	script := fmt.Sprintf(`
import pyatspi, sys

MAX_DEPTH = %d

def dump(node, depth=0):
    if depth > MAX_DEPTH:
        return ""
    indent = "  " * depth
    role = node.getRoleName() or "?"
    name = (node.name or "").replace("\n", " ")
    desc = (node.description or "").replace("\n", " ")
    value = ""
    try:
        vi = node.queryValue()
        value = str(vi.currentValue)
    except Exception:
        pass
    parts = [role]
    if name:
        parts.append("name=" + name)
    if desc:
        parts.append("desc=" + desc)
    if value:
        parts.append("value=" + value)
    out = indent + " | ".join(parts) + "\n"
    for i in range(node.childCount):
        try:
            out += dump(node[i], depth + 1)
        except Exception:
            pass
    return out

try:
    desktop = pyatspi.Registry.getDesktop(0)
except Exception as e:
    print("frontmost=unknown — AT-SPI unavailable: " + str(e))
    sys.exit(0)

front_app = None
front_name = "unknown"
for app in desktop:
    if app is None:
        continue
    try:
        for win in app:
            try:
                if win.getState().contains(pyatspi.STATE_ACTIVE):
                    front_app = app
                    front_name = app.name or "unknown"
                    break
            except Exception:
                pass
        if front_app is not None:
            break
    except Exception:
        pass

if front_app is None:
    print("frontmost=" + front_name + " — no active window found")
    sys.exit(0)

header = "frontmost=" + front_name + "\n"
tree = dump(front_app)
print(header + tree, end="")
`, maxDepth)

	out, err := exec.Command("python3", "-c", script).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inspector: AT-SPI: %w — %s (sudo apt install python3-pyatspi at-spi2-core)", err, strings.TrimSpace(string(out)))
	}
	result := strings.TrimSpace(string(out))
	if len(result) > 4000 {
		result = result[:4000] + "\n...[truncated]"
	}
	return result, nil
}
