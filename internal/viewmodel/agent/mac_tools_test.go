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
		{"audio with unknown action", macCall("control_audio", map[string]string{"action": "eject"})},
		{"audio set without number", macCall("control_audio", map[string]string{"action": "set", "percent": "loud"})},
		{"power with unknown action", macCall("mac_power", map[string]string{"action": "explode"})},
		{"reveal without path", macCall("reveal_in_finder", map[string]string{})},
		{"file_info without path", macCall("file_info", map[string]string{})},
		{"trash move without path", macCall("manage_trash", map[string]string{"action": "move"})},
		{"trash with unknown action", macCall("manage_trash", map[string]string{"action": "burn"})},
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
		"say_text", "show_notification", "mac_power",
		"reveal_in_finder", "file_info", "manage_trash",
	} {
		if !seen[required] {
			t.Errorf("missing tool definition %q", required)
		}
	}
}

// Emptying the Trash is irreversible, so the agent may not do it: it has to
// hand the user back to the /trash + /yes flow where they see what is at
// stake first.
func TestManageTrashRefusesToEmpty(t *testing.T) {
	vm := &ViewModel{}
	parts, err, handled := vm.executeMacTool(macCall("manage_trash", map[string]string{"action": "empty"}), nil)
	if !handled || err == nil {
		t.Fatalf("emptying must be refused: handled=%v err=%v", handled, err)
	}
	text := partsText(parts)
	if !strings.Contains(text, "/trash") || !strings.Contains(text, "/yes") {
		t.Errorf("the refusal should point at the confirmed flow, got %q", text)
	}
}
