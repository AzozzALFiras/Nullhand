package accessibility

import (
	"fmt"
	"os/exec"
	"strings"
)

// ListElements returns a compact summary of the AT-SPI2 accessibility tree of
// the currently frontmost application's main window. Each line is:
//
//	<indent>role | name | description | value
//
// maxDepth caps recursion so very complex apps do not return 10 KB of text.
// A depth of 6–10 is usually enough to reveal whether an app exposes its
// content via AT-SPI at all — Electron apps come back with a tiny tree of only
// group containers, while native GTK/Qt apps return hundreds of elements.
func ListElements(maxDepth int) (string, error) {
	if maxDepth <= 0 {
		maxDepth = 8
	}

	// Use python3 + pyatspi to dump the AT-SPI2 tree of the active window.
	script := fmt.Sprintf(`
import pyatspi, sys

MAX_DEPTH = %d

def dump(node, depth=0):
    if depth > MAX_DEPTH:
        return ''
    indent = '  ' * depth
    role = node.getRoleName() or '?'
    name = (node.name or '').replace('\n', ' ')
    desc = (node.description or '').replace('\n', ' ')
    value = ''
    try:
        vi = node.queryValue()
        value = str(vi.currentValue)
    except Exception:
        pass
    parts = [role]
    if name:
        parts.append('name=' + name)
    if desc:
        parts.append('desc=' + desc)
    if value:
        parts.append('value=' + value)
    out = indent + ' | '.join(parts) + '\n'
    for i in range(node.childCount):
        try:
            out += dump(node[i], depth + 1)
        except Exception:
            pass
    return out

desktop = pyatspi.Registry.getDesktop(0)
front_app = None
front_name = 'unknown'
for app in desktop:
    try:
        if app is None:
            continue
        for win in app:
            try:
                state = win.getState()
                if state.contains(pyatspi.STATE_ACTIVE):
                    front_app = app
                    front_name = app.name or 'unknown'
                    break
            except Exception:
                pass
        if front_app:
            break
    except Exception:
        pass

if front_app is None:
    print('frontmost=unknown — no window')
    sys.exit(0)

header = 'frontmost=' + front_name + '\n'
tree = dump(front_app)
print(header + tree, end='')
`, maxDepth)

	out, err := exec.Command("python3", "-c", script).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inspector: AT-SPI: %w — %s", err, strings.TrimSpace(string(out)))
	}
	result := strings.TrimSpace(string(out))
	if len(result) > 4000 {
		result = result[:4000] + "\n...[truncated]"
	}
	return result, nil
}
