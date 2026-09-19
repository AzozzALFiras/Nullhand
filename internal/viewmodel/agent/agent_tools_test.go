package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	aimodel "github.com/AzozzALFiras/Nullhand/internal/model/ai"
)

// The agent loop cannot wait for /yes, so a destructive run_shell call must
// be refused without touching anything and must tell the user how to run it.
func TestRunShellRefusesDestructiveCommand(t *testing.T) {
	victim := filepath.Join(t.TempDir(), "keep.txt")
	if err := os.WriteFile(victim, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}

	vm := &ViewModel{}
	parts, err := vm.executeTool(aimodel.ToolCall{
		ToolName:  "run_shell",
		Arguments: map[string]string{"command": "rm " + victim},
	}, nil)

	if err == nil {
		t.Fatal("destructive command must return an error to the agent loop")
	}
	if _, statErr := os.Stat(victim); statErr != nil {
		t.Fatalf("file was deleted even though the command was refused: %v", statErr)
	}
	text := partsText(parts)
	// The local provider only surfaces tool results starting with ❌ or ⚠️.
	if !strings.HasPrefix(text, "⚠️") {
		t.Errorf("refusal must start with ⚠️ so the local provider shows it, got %q", text)
	}
	if !strings.Contains(text, "/shell rm "+victim) || !strings.Contains(text, "/yes") {
		t.Errorf("refusal should explain how to run it with confirmation, got %q", text)
	}
}

func TestRunShellRunsSafeCommand(t *testing.T) {
	vm := &ViewModel{}
	parts, err := vm.executeTool(aimodel.ToolCall{
		ToolName:  "run_shell",
		Arguments: map[string]string{"command": "echo nullhand"},
	}, nil)
	if err != nil {
		t.Fatalf("safe command should run: %v", err)
	}
	if got := partsText(parts); got != "nullhand" {
		t.Errorf("unexpected output %q", got)
	}
}
