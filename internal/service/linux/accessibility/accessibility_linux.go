//go:build linux

package accessibility

import (
	"fmt"
	"os/exec"
	"strings"

	appsvc "github.com/AzozzALFiras/nullhand/internal/service/linux/apps"
)

// FocusField searches the FRONTMOST application's frontmost window for a
// text input field whose placeholder, help, description, or label contains
// labelHint (case-insensitive) and clicks it to give it focus.
//
// Requires AT-SPI2: sudo apt install python3-pyatspi at-spi2-core
func FocusField(appName, labelHint string) error {
	if labelHint == "" {
		return fmt.Errorf("accessibility: label is required")
	}
	if appName != "" && !appsvc.IsNativeAX(appName) {
		return fmt.Errorf("accessibility: %q is not a native AT-SPI app — use focus_via_palette or run_recipe instead", appName)
	}

	script := fmt.Sprintf(`
import pyatspi, sys, os

def find_active_app(desktop):
    for app in desktop:
        if app is None:
            continue
        try:
            for win in app:
                try:
                    if win.getState().contains(pyatspi.STATE_ACTIVE):
                        return app, app.name or "unknown"
                except Exception:
                    pass
        except Exception:
            pass
    return None, "unknown"

def find_text_field(node, needle, depth=0):
    if depth > 25:
        return None
    role = node.getRoleName()
    if role in ("text", "entry", "text area", "combo box", "editbar", "spin button"):
        # Check name, description and labelled-by relation.
        for attr in ("name", "description"):
            try:
                val = getattr(node, attr, "") or ""
                if needle in val.lower():
                    return node
            except Exception:
                pass
        try:
            for rel in node.getRelationSet():
                if rel.getRelationType() == pyatspi.RELATION_LABELLED_BY:
                    for i in range(rel.getNTargets()):
                        lbl = rel.getTarget(i)
                        if needle in (lbl.name or "").lower():
                            return node
        except Exception:
            pass
    for i in range(node.childCount):
        try:
            result = find_text_field(node[i], needle, depth + 1)
            if result is not None:
                return result
        except Exception:
            pass
    return None

needle = %q.lower()
try:
    desktop = pyatspi.Registry.getDesktop(0)
except Exception as e:
    print("no window:at-spi unavailable: " + str(e))
    sys.exit(0)

front_app, front_name = find_active_app(desktop)
if front_app is None:
    print("no window:" + front_name)
    sys.exit(0)

field = find_text_field(front_app, needle)
if field is None:
    print("not found:" + front_name)
    sys.exit(0)

# Try AT-SPI action first, then coordinate fallback.
activated = False
try:
    action_iface = field.queryAction()
    for i in range(action_iface.nActions):
        name = action_iface.getName(i).lower()
        if name in ("click", "activate", "set focus", "focus"):
            action_iface.doAction(i)
            activated = True
            break
except Exception:
    pass

if not activated:
    try:
        comp = field.queryComponent()
        ext = comp.getExtents(pyatspi.DESKTOP_COORDS)
        cx = ext.x + ext.width // 2
        cy = ext.y + ext.height // 2
        import subprocess
        subprocess.run(["xdotool", "mousemove", "--sync", str(cx), str(cy), "click", "1"],
                       check=True, capture_output=True)
        activated = True
    except Exception as e:
        print("not found:" + front_name + " (coordinate fallback failed: " + str(e) + ")")
        sys.exit(0)

if activated:
    print("ok:" + front_name)
else:
    print("not found:" + front_name)
`, labelHint)

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
		return fmt.Errorf("accessibility: no text field matching %q in %q", labelHint, frontName)
	default:
		return fmt.Errorf("accessibility: unexpected focus result: %s", result)
	}
}

// Click searches the FRONTMOST application's frontmost window for a UI
// element whose name or description exactly matches label and clicks it.
//
// Requires AT-SPI2: sudo apt install python3-pyatspi at-spi2-core
func Click(appName, label string) error {
	if label == "" {
		return fmt.Errorf("accessibility: label is required")
	}

	script := fmt.Sprintf(`
import pyatspi, sys

def find_active_app(desktop):
    for app in desktop:
        if app is None:
            continue
        try:
            for win in app:
                try:
                    if win.getState().contains(pyatspi.STATE_ACTIVE):
                        return app, app.name or "unknown"
                except Exception:
                    pass
        except Exception:
            pass
    return None, "unknown"

def find_element(node, needle, depth=0):
    if depth > 25:
        return None
    for attr in ("name", "description"):
        try:
            val = getattr(node, attr, "") or ""
            if val == needle:
                return node
        except Exception:
            pass
    for i in range(node.childCount):
        try:
            result = find_element(node[i], needle, depth + 1)
            if result is not None:
                return result
        except Exception:
            pass
    return None

needle = %q
try:
    desktop = pyatspi.Registry.getDesktop(0)
except Exception as e:
    print("no window:at-spi unavailable: " + str(e))
    sys.exit(0)

front_app, front_name = find_active_app(desktop)
if front_app is None:
    print("no window:" + front_name)
    sys.exit(0)

el = find_element(front_app, needle)
if el is None:
    print("not found:" + front_name)
    sys.exit(0)

clicked = False
try:
    action_iface = el.queryAction()
    for i in range(action_iface.nActions):
        name = action_iface.getName(i).lower()
        if name in ("click", "press", "activate"):
            action_iface.doAction(i)
            clicked = True
            break
except Exception:
    pass

if not clicked:
    try:
        comp = el.queryComponent()
        ext = comp.getExtents(pyatspi.DESKTOP_COORDS)
        cx = ext.x + ext.width // 2
        cy = ext.y + ext.height // 2
        import subprocess
        subprocess.run(["xdotool", "mousemove", "--sync", str(cx), str(cy), "click", "1"],
                       check=True, capture_output=True)
        clicked = True
    except Exception as e:
        print("not found:" + front_name + " (fallback failed: " + str(e) + ")")
        sys.exit(0)

if clicked:
    print("ok:" + front_name)
else:
    print("not found:" + front_name)
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
