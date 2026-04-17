package accessibility

import (
	"fmt"
	"os/exec"
	"strings"

	appsvc "github.com/AzozzALFiras/nullhand/internal/service/macos/apps"
)

// FocusField searches the FRONTMOST application's frontmost window for a
// text input field (text field, text area, or combo box) whose placeholder,
// help, description, or title contains the labelHint substring, and clicks
// it to give it keyboard focus. Case-insensitive substring match.
//
// appName is only used for error messages — the actual search always targets
// whatever process is currently frontmost. This is intentional: Linux process
// names often differ from the user-facing app name (e.g. "Visual Studio Code"
// runs as process "code"), and callers are expected to have already called
// open_app to bring the target app to the front.
//
// Requires AT-SPI2 accessibility support (python3-pyatspi).
func FocusField(appName, labelHint string) error {
	if labelHint == "" {
		return fmt.Errorf("accessibility: label is required")
	}

	// Guard: focus_text_field only works for native AT-SPI apps. Electron apps
	// expose an empty AX tree and will always return "not found", wasting
	// 10+ seconds per attempt. Self-correct immediately.
	if appName != "" && !appsvc.IsNativeAX(appName) {
		return fmt.Errorf("accessibility: %q is not a native AX app — use focus_via_palette or run_recipe instead", appName)
	}

	// Use python3 + pyatspi to find and click the text field by label hint.
	script := fmt.Sprintf(`
import pyatspi, sys

needle = %q.lower()

def find_text_field(node, depth=0):
    if depth > 20:
        return None
    role = node.getRoleName()
    if role in ('text', 'entry', 'text area', 'combo box', 'editbar'):
        for attr in ('description', 'name', 'label'):
            try:
                val = getattr(node, attr, '') or ''
                if needle in val.lower():
                    return node
            except Exception:
                pass
        # Check relations (labelled-by)
        try:
            for rel in node.getRelationSet():
                if rel.getRelationType() == pyatspi.RELATION_LABELLED_BY:
                    for i in range(rel.getNTargets()):
                        label_node = rel.getTarget(i)
                        if needle in (label_node.name or '').lower():
                            return node
        except Exception:
            pass
    for i in range(node.childCount):
        try:
            result = find_text_field(node[i], depth + 1)
            if result:
                return result
        except Exception:
            pass
    return None

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
    print('no window:unknown')
    sys.exit(0)

field = find_text_field(front_app)
if field is None:
    print('not found:' + front_name)
    sys.exit(0)

try:
    action_iface = field.queryAction()
    for i in range(action_iface.nActions):
        name = action_iface.getName(i)
        if name.lower() in ('click', 'activate', 'set focus'):
            action_iface.doAction(i)
            print('ok:' + front_name)
            sys.exit(0)
except Exception:
    pass

# Fallback: use component interface to get coordinates and xdotool click.
try:
    comp = field.queryComponent()
    ext = comp.getExtents(pyatspi.DESKTOP_COORDS)
    cx = ext.x + ext.width // 2
    cy = ext.y + ext.height // 2
    import subprocess
    subprocess.run(['xdotool', 'mousemove', str(cx), str(cy), 'click', '1'])
    print('ok:' + front_name)
    sys.exit(0)
except Exception as e:
    print('not found:' + front_name)
`, labelHint)

	out, err := exec.Command("python3", "-c", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("accessibility: focus AT-SPI failed: %w — %s", err, strings.TrimSpace(string(out)))
	}

	result := strings.TrimSpace(string(out))
	switch {
	case strings.HasPrefix(result, "ok:"):
		return nil
	case strings.HasPrefix(result, "no window:"):
		frontName := strings.TrimPrefix(result, "no window:")
		return fmt.Errorf("accessibility: frontmost app %q has no open window (expected %q)", frontName, appName)
	case strings.HasPrefix(result, "not found:"):
		frontName := strings.TrimPrefix(result, "not found:")
		return fmt.Errorf("accessibility: no text field matching %q in %q", labelHint, frontName)
	default:
		return fmt.Errorf("accessibility: unexpected focus result: %s", result)
	}
}

// Click searches the FRONTMOST application's frontmost window for a UI
// element whose name, title, or description matches label, and clicks it.
//
// appName is only used for error messages — the actual search targets the
// currently frontmost process. Callers must have already focused the target
// app via open_app before calling this.
//
// Requires AT-SPI2 accessibility support (python3-pyatspi).
func Click(appName, label string) error {
	if label == "" {
		return fmt.Errorf("accessibility: label is required")
	}

	script := fmt.Sprintf(`
import pyatspi, sys

needle = %q

def find_element(node, depth=0):
    if depth > 20:
        return None
    for attr in ('name', 'description'):
        try:
            val = getattr(node, attr, '') or ''
            if val == needle:
                return node
        except Exception:
            pass
    for i in range(node.childCount):
        try:
            result = find_element(node[i], depth + 1)
            if result:
                return result
        except Exception:
            pass
    return None

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
    print('no window:unknown')
    sys.exit(0)

el = find_element(front_app)
if el is None:
    print('not found:' + front_name)
    sys.exit(0)

try:
    action_iface = el.queryAction()
    for i in range(action_iface.nActions):
        name = action_iface.getName(i)
        if name.lower() in ('click', 'press', 'activate'):
            action_iface.doAction(i)
            print('ok:' + front_name)
            sys.exit(0)
except Exception:
    pass

# Fallback: get screen coordinates and use xdotool.
try:
    comp = el.queryComponent()
    ext = comp.getExtents(pyatspi.DESKTOP_COORDS)
    cx = ext.x + ext.width // 2
    cy = ext.y + ext.height // 2
    import subprocess
    subprocess.run(['xdotool', 'mousemove', str(cx), str(cy), 'click', '1'])
    print('ok:' + front_name)
    sys.exit(0)
except Exception:
    print('not found:' + front_name)
`, label)

	out, err := exec.Command("python3", "-c", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("accessibility: AT-SPI failed: %w — %s", err, strings.TrimSpace(string(out)))
	}

	result := strings.TrimSpace(string(out))
	switch {
	case strings.HasPrefix(result, "ok:"):
		return nil
	case strings.HasPrefix(result, "no window:"):
		frontName := strings.TrimPrefix(result, "no window:")
		return fmt.Errorf("accessibility: frontmost app %q has no open window (expected %q)", frontName, appName)
	case strings.HasPrefix(result, "not found:"):
		frontName := strings.TrimPrefix(result, "not found:")
		return fmt.Errorf("accessibility: no element named %q found in %q", label, frontName)
	default:
		return fmt.Errorf("accessibility: unexpected result: %s", result)
	}
}
