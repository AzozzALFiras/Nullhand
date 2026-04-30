//go:build linux

package accessibility

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	appsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/apps"
)

// ElementSummary is a compact description of an AT-SPI element returned by
// FindElements. Coordinates are in screen pixels (top-left origin).
type ElementSummary struct {
	Role        string
	Name        string
	Description string
	X, Y, W, H  int
}

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

// ClickFuzzy is like Click but uses case-insensitive substring matching on the
// element's name OR description. Falls back through several strategies:
//   1. exact match (same as Click)
//   2. case-insensitive equality
//   3. substring (case-insensitive)
// Returns ErrFuzzyNotFound if no element matches.
func ClickFuzzy(appName, label string) error {
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

CLICKABLE_ROLES = (
    "push button", "toggle button", "radio button", "check box",
    "menu item", "check menu item", "radio menu item",
    "link", "list item", "tab", "tree item", "toolbar button",
    "button",
)

def is_clickable_role(role):
    if role in CLICKABLE_ROLES:
        return True
    return False

def find_candidates(node, needle, depth=0, out=None):
    if out is None:
        out = []
    if depth > 25 or len(out) > 200:
        return out
    role = ""
    name = ""
    desc = ""
    try:
        role = (node.getRoleName() or "")
    except Exception:
        pass
    try:
        name = node.name or ""
    except Exception:
        pass
    try:
        desc = node.description or ""
    except Exception:
        pass
    score = 0
    nl = name.lower().strip()
    dl = desc.lower().strip()
    if nl == needle:
        score = 100
    elif dl == needle:
        score = 95
    elif needle in nl:
        score = 80
    elif needle in dl:
        score = 70
    if score > 0:
        out.append((score, node, role, name, desc))
    for i in range(node.childCount):
        try:
            find_candidates(node[i], needle, depth + 1, out)
        except Exception:
            pass
    return out

needle = %q.lower().strip()
try:
    desktop = pyatspi.Registry.getDesktop(0)
except Exception as e:
    print("no window:at-spi unavailable: " + str(e))
    sys.exit(0)

front_app, front_name = find_active_app(desktop)
if front_app is None:
    print("no window:" + front_name)
    sys.exit(0)

candidates = find_candidates(front_app, needle)
if not candidates:
    print("not found:" + front_name)
    sys.exit(0)

# Prefer clickable roles among ties.
candidates.sort(key=lambda c: (-c[0], 0 if is_clickable_role(c[2]) else 1))
_, target, role, name, desc = candidates[0]

clicked = False
try:
    action_iface = target.queryAction()
    for i in range(action_iface.nActions):
        an = action_iface.getName(i).lower()
        if an in ("click", "press", "activate"):
            action_iface.doAction(i)
            clicked = True
            break
except Exception:
    pass

if not clicked:
    try:
        comp = target.queryComponent()
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
    print("ok:" + front_name + ":" + role + ":" + name)
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
		return fmt.Errorf("accessibility: %s", strings.TrimPrefix(result, "no window:"))
	case strings.HasPrefix(result, "not found:"):
		return fmt.Errorf("accessibility: no element matching %q in %s", label, strings.TrimPrefix(result, "not found:"))
	default:
		return fmt.Errorf("accessibility: unexpected result: %s", result)
	}
}

// WaitForElement polls every 250ms until an element matching label exists in
// the frontmost app (substring + case-insensitive), or timeoutMs expires.
func WaitForElement(appName, label string, timeoutMs int) error {
	if label == "" {
		return fmt.Errorf("accessibility: label is required")
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	deadline := time.Now().Add(time.Duration(timeoutMs) * time.Millisecond)
	for {
		summaries, err := FindElements(appName, label, 1)
		if err == nil && len(summaries) > 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("accessibility: timeout after %dms waiting for %q", timeoutMs, label)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// FindElements searches the frontmost app for elements matching label
// (case-insensitive substring on name OR description) and returns up to max
// summaries with their bounding boxes.
func FindElements(appName, label string, max int) ([]ElementSummary, error) {
	if label == "" {
		return nil, fmt.Errorf("accessibility: label is required")
	}
	if max <= 0 {
		max = 10
	}
	if appName != "" && !appsvc.IsNativeAX(appName) {
		// Still try; AT-SPI may work for some Electron apps. Don't block.
		_ = appName
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

def walk(node, needle, depth, out, limit):
    if depth > 25 or len(out) >= limit:
        return
    role = ""
    name = ""
    desc = ""
    try:
        role = (node.getRoleName() or "")
    except Exception:
        pass
    try:
        name = node.name or ""
    except Exception:
        pass
    try:
        desc = node.description or ""
    except Exception:
        pass
    nl = name.lower()
    dl = desc.lower()
    if needle in nl or needle in dl:
        x = y = w = h = 0
        try:
            comp = node.queryComponent()
            ext = comp.getExtents(pyatspi.DESKTOP_COORDS)
            x, y, w, h = ext.x, ext.y, ext.width, ext.height
        except Exception:
            pass
        out.append("|".join([role, name, desc, str(x), str(y), str(w), str(h)]))
    for i in range(node.childCount):
        try:
            walk(node[i], needle, depth + 1, out, limit)
        except Exception:
            pass

needle = %q.lower().strip()
limit = %d
try:
    desktop = pyatspi.Registry.getDesktop(0)
except Exception as e:
    print("ERR:at-spi unavailable: " + str(e))
    sys.exit(0)

front_app, front_name = find_active_app(desktop)
if front_app is None:
    print("ERR:no active window")
    sys.exit(0)

out = []
walk(front_app, needle, 0, out, limit)
if not out:
    print("EMPTY")
    sys.exit(0)
for line in out:
    print(line)
`, label, max)

	rawOut, err := exec.Command("python3", "-c", script).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("accessibility: AT-SPI failed: %w — %s", err, strings.TrimSpace(string(rawOut)))
	}
	result := strings.TrimSpace(string(rawOut))
	if strings.HasPrefix(result, "ERR:") {
		return nil, fmt.Errorf("accessibility: %s", strings.TrimPrefix(result, "ERR:"))
	}
	if result == "EMPTY" || result == "" {
		return nil, nil
	}
	var out []ElementSummary
	for _, line := range strings.Split(result, "\n") {
		fields := strings.Split(line, "|")
		if len(fields) < 7 {
			continue
		}
		var x, y, w, h int
		fmt.Sscanf(fields[3], "%d", &x)
		fmt.Sscanf(fields[4], "%d", &y)
		fmt.Sscanf(fields[5], "%d", &w)
		fmt.Sscanf(fields[6], "%d", &h)
		out = append(out, ElementSummary{
			Role:        fields[0],
			Name:        fields[1],
			Description: fields[2],
			X:           x, Y: y, W: w, H: h,
		})
	}
	return out, nil
}
