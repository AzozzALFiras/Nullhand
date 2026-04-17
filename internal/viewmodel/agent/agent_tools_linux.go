//go:build linux

package agent

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	aimodel "github.com/iamakillah/Nullhand_Linux/internal/model/ai"
	a11ysvc "github.com/iamakillah/Nullhand_Linux/internal/service/linux/accessibility"
	appsvc "github.com/iamakillah/Nullhand_Linux/internal/service/linux/apps"
	filesvc "github.com/iamakillah/Nullhand_Linux/internal/service/linux/files"
	kbsvc "github.com/iamakillah/Nullhand_Linux/internal/service/linux/keyboard"
	mousesvc "github.com/iamakillah/Nullhand_Linux/internal/service/linux/mouse"
	palettesvc "github.com/iamakillah/Nullhand_Linux/internal/service/linux/palette"
	screensvc "github.com/iamakillah/Nullhand_Linux/internal/service/linux/screen"
	shellsvc "github.com/iamakillah/Nullhand_Linux/internal/service/linux/shell"
)

// executeTool calls the appropriate Linux service based on tool name.
func (vm *ViewModel) executeTool(tc aimodel.ToolCall, sendPhoto PhotoFunc) ([]aimodel.MessagePart, error) {
	args := tc.Arguments

	switch tc.ToolName {
	case "take_screenshot":
		if sendPhoto == nil {
			return textParts("error: screenshot delivery not available"), fmt.Errorf("sendPhoto is nil")
		}
		data, err := screensvc.Capture()
		if err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		if err := sendPhoto(data, "📸"); err != nil {
			data = nil
			return textParts(fmt.Sprintf("error delivering screenshot: %v", err)), err
		}
		data = nil
		return textParts("screenshot delivered to user (AI cannot see it)"), nil

	case "analyze_screenshot":
		data, err := screensvc.CaptureResized(0)
		if err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		w, h, _ := screensvc.Size()
		encoded := base64.StdEncoding.EncodeToString(data)
		data = nil
		return []aimodel.MessagePart{
			{Type: aimodel.ContentTypeText, Text: fmt.Sprintf("Screenshot captured (%dx%d). Pixel coordinates in this image map directly to click/move coordinates — no conversion needed.", w, h)},
			{Type: aimodel.ContentTypeImage, ImageBase64: encoded, MimeType: "image/png"},
		}, nil

	case "open_app":
		app := args["app_name"]
		if err := appsvc.Open(app); err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(fmt.Sprintf("opened %s", app)), nil

	case "click":
		x, y, err := parseXY(args)
		if err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		if err := mousesvc.Click(x, y); err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(fmt.Sprintf("clicked at %d,%d", x, y)), nil

	case "click_ui_element":
		app := args["app_name"]
		label := args["label"]
		if err := a11ysvc.Click(app, label); err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(fmt.Sprintf("clicked %q in %s", label, app)), nil

	case "focus_text_field":
		app := args["app_name"]
		label := args["label"]
		if err := a11ysvc.FocusField(app, label); err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(fmt.Sprintf("focused text field %q in %s", label, app)), nil

	case "focus_via_palette":
		shortcut := args["palette_shortcut"]
		command := args["command_name"]
		if err := palettesvc.Run(shortcut, command); err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(fmt.Sprintf("invoked palette command %q", command)), nil

	case "list_ui_elements":
		depth := 8
		if d := args["max_depth"]; d != "" {
			_, _ = fmt.Sscanf(d, "%d", &depth)
		}
		tree, err := a11ysvc.ListElements(depth)
		if err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(tree), nil

	case "is_native_app":
		app := args["app_name"]
		if appsvc.IsNativeAX(app) {
			return textParts("native (focus_text_field will work)"), nil
		}
		return textParts("electron or unknown (use focus_via_palette or run_recipe)"), nil

	case "list_recipes":
		if vm.recipes == nil {
			return textParts("no recipes available"), nil
		}
		var sb strings.Builder
		for _, r := range vm.recipes.List() {
			sb.WriteString(r.Name)
			if r.Description != "" {
				sb.WriteString(" — ")
				sb.WriteString(r.Description)
			}
			if len(r.Parameters) > 0 {
				sb.WriteString(" [params: ")
				sb.WriteString(strings.Join(r.Parameters, ", "))
				sb.WriteString("]")
			}
			sb.WriteString("\n")
		}
		return textParts(strings.TrimRight(sb.String(), "\n")), nil

	case "run_recipe":
		if vm.recipes == nil {
			return textParts("error: recipe service not available"), fmt.Errorf("recipes disabled")
		}
		name := args["name"]
		params := map[string]string{}
		if raw := args["params_json"]; raw != "" {
			if err := jsonUnmarshal([]byte(raw), &params); err != nil {
				return textParts(fmt.Sprintf("error: bad params_json: %v", err)), err
			}
		}
		dryRun := args["dry_run"] == "true"
		plan, err := vm.recipes.Run(name, params, dryRun)
		if err != nil {
			return textParts(fmt.Sprintf("error: %v\n%s", err, plan)), err
		}
		if dryRun {
			return textParts(fmt.Sprintf("dry-run ok:\n%s", plan)), nil
		}
		return textParts(fmt.Sprintf("recipe %q ok:\n%s", name, plan)), nil

	case "right_click":
		x, y, err := parseXY(args)
		if err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		if err := mousesvc.RightClick(x, y); err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(fmt.Sprintf("right-clicked at %d,%d", x, y)), nil

	case "double_click":
		x, y, err := parseXY(args)
		if err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		if err := mousesvc.DoubleClick(x, y); err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(fmt.Sprintf("double-clicked at %d,%d", x, y)), nil

	case "move_mouse":
		x, y, err := parseXY(args)
		if err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		if err := mousesvc.Move(x, y); err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(fmt.Sprintf("moved mouse to %d,%d", x, y)), nil

	case "scroll":
		direction := args["direction"]
		steps := 3
		if s := args["steps"]; s != "" {
			fmt.Sscanf(s, "%d", &steps)
		}
		if steps < 1 {
			steps = 1
		}
		if err := mousesvc.Scroll(direction, steps); err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(fmt.Sprintf("scrolled %s %d steps", direction, steps)), nil

	case "drag":
		var x1, y1, x2, y2 int
		fmt.Sscanf(args["x1"], "%d", &x1)
		fmt.Sscanf(args["y1"], "%d", &y1)
		fmt.Sscanf(args["x2"], "%d", &x2)
		fmt.Sscanf(args["y2"], "%d", &y2)
		if err := mousesvc.Drag(x1, y1, x2, y2); err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(fmt.Sprintf("dragged from %d,%d to %d,%d", x1, y1, x2, y2)), nil

	case "type_text":
		text := args["text"]
		prev, hadPrev, err := kbsvc.TypeAndHold(text)
		if err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		vm.savedClipboard = prev
		vm.hadSavedClipboard = hadPrev
		return textParts("typed text"), nil

	case "verify_clipboard":
		got := kbsvc.ReadClipboard()
		return textParts(got), nil

	case "restore_clipboard":
		if vm.hadSavedClipboard {
			kbsvc.RestoreClipboard(vm.savedClipboard)
			vm.savedClipboard = ""
			vm.hadSavedClipboard = false
			return textParts("clipboard restored"), nil
		}
		return textParts("nothing to restore"), nil

	case "request_manual_focus":
		if vm.manualFocus == nil {
			return textParts("error: manual focus not available in this context"), fmt.Errorf("no manualFocus callback")
		}
		reason := args["reason"]
		if reason == "" {
			reason = "I could not locate the target input automatically."
		}
		reply, err := vm.manualFocus(reason)
		if err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(fmt.Sprintf("user replied: %s", reply)), nil

	case "press_key":
		key := args["key"]
		if err := kbsvc.PressKey(key); err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(fmt.Sprintf("pressed %s", key)), nil

	case "run_shell":
		cmd := args["command"]
		out, err := shellsvc.Run(cmd)
		if err != nil {
			return textParts(fmt.Sprintf("error: %v\n%s", err, out)), err
		}
		return textParts(out), nil

	case "read_file":
		path := args["path"]
		content, err := filesvc.Read(path)
		if err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(content), nil

	case "list_directory":
		path := args["path"]
		if path == "" {
			path = "."
		}
		entries, err := filesvc.List(path)
		if err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(strings.Join(entries, "\n")), nil

	case "get_clipboard":
		text, err := filesvc.GetClipboard()
		if err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts(text), nil

	case "set_clipboard":
		text := args["text"]
		if err := filesvc.SetClipboard(text); err != nil {
			return textParts(fmt.Sprintf("error: %v", err)), err
		}
		return textParts("clipboard set"), nil

	case "browse_folder":
		path := args["path"]
		if path == "" {
			path = "~"
		}
		if vm.browse != nil {
			vm.browse(path)
		}
		return textParts(fmt.Sprintf("opened file browser at %s", path)), nil

	case "wait":
		var ms int
		fmt.Sscanf(args["ms"], "%d", &ms)
		if ms > 5000 {
			ms = 5000
		}
		if ms < 0 {
			ms = 0
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return textParts(fmt.Sprintf("waited %dms", ms)), nil

	default:
		return textParts(fmt.Sprintf("unknown tool: %s", tc.ToolName)), nil
	}
}
