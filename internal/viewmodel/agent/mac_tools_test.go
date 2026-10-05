package agent

import (
	"strings"
	"testing"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
)

func macCall(tool string, args map[string]string) aimodel.ToolCall {
	return aimodel.ToolCall{ToolName: tool, Arguments: args}
}

// Bad arguments must be caught before the tool touches the system, so these
// run the same on macOS and Linux.
func TestMacToolsRejectBadArguments(t *testing.T) {
	vm := &ViewModel{}
	cases := []struct {
		name string
		call aimodel.ToolCall
	}{
		{"shortcut without name", macCall("run_shortcut", map[string]string{})},
		{"find without query", macCall("find_files", map[string]string{"folder": "~"})},
		{"audio with unknown action", macCall("control_audio", map[string]string{"action": "eject"})},
		{"audio set without number", macCall("control_audio", map[string]string{"action": "set", "percent": "loud"})},
		{"power with unknown action", macCall("mac_power", map[string]string{"action": "explode"})},
	}
	for _, c := range cases {
		parts, err, handled := vm.executeMacTool(c.call, nil)
		if !handled {
			t.Errorf("%s: tool should be handled by the mac handler", c.name)
			continue
		}
		if err == nil {
			t.Errorf("%s: expected an error", c.name)
		}
		if text := partsText(parts); !strings.HasPrefix(text, "⚠️") {
			t.Errorf("%s: the offline parser only surfaces ⚠️/❌/ℹ️ results, got %q", c.name, text)
		}
	}
}

func TestPreviewFileNeedsPhotoDelivery(t *testing.T) {
	vm := &ViewModel{}
	_, err, handled := vm.executeMacTool(macCall("preview_file", map[string]string{"path": "/etc/hosts"}), nil)
	if !handled || err == nil {
		t.Errorf("without a photo callback the preview cannot be delivered: handled=%v err=%v", handled, err)
	}
}

func TestExecuteMacToolIgnoresOtherTools(t *testing.T) {
	vm := &ViewModel{}
	if _, _, handled := vm.executeMacTool(macCall("run_shell", map[string]string{"command": "ls"}), nil); handled {
		t.Error("non-macOS tools must fall through to the main handler")
	}
}

// The offline parser decides what to show the user by prefix, so the marker
// the informational results carry must be one it knows.
func TestInfoPartsUseTheSurfacedMarker(t *testing.T) {
	if got := partsText(infoParts("battery 95%")); !strings.HasPrefix(got, "ℹ️") {
		t.Errorf("got %q", got)
	}
}

func TestMacToolDefinitionsAreUsable(t *testing.T) {
	defs := macToolDefinitions()
	if len(defs) == 0 {
		t.Fatal("no macOS tools defined")
	}
	seen := map[string]bool{}
	for _, d := range defs {
		if d.Name == "" || d.Description == "" {
			t.Errorf("tool %+v needs a name and a description", d)
		}
		if seen[d.Name] {
			t.Errorf("duplicate tool name %q", d.Name)
		}
		seen[d.Name] = true
		for _, p := range d.Parameters {
			if p.Name == "" || p.Description == "" {
				t.Errorf("tool %q has an undocumented parameter %+v", d.Name, p)
			}
		}
	}
	// Every tool the intent layer can emit must exist here, or an offline
	// phrase would resolve to a tool the agent cannot execute.
	for _, required := range []string{
		"run_shortcut", "list_shortcuts", "control_audio", "control_media",
		"say_text", "show_notification", "find_files", "preview_file", "mac_power",
	} {
		if !seen[required] {
			t.Errorf("missing tool definition %q", required)
		}
	}
}
