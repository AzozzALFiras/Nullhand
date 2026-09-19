package bot

import "testing"

// The schedule-creation confirmation recognises shell tasks by the
// "shell: <cmd>" label resolveScheduledAction produces; pin that contract.
func TestResolveScheduledActionLabelsShellCommands(t *testing.T) {
	vm := &ViewModel{}
	action, label := vm.resolveScheduledAction("run rm -rf /tmp/cache", 1, 1, "")
	if action == nil {
		t.Fatal("run <cmd> should resolve to a shell action")
	}
	if label != "shell: rm -rf /tmp/cache" {
		t.Errorf("label = %q, want %q", label, "shell: rm -rf /tmp/cache")
	}
}
