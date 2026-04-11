package permissions

import (
	permsvc "github.com/AzozzALFiras/nullhand/internal/service/macos/permissions"
	permview "github.com/AzozzALFiras/nullhand/internal/view/permissions"
)

// ViewModel orchestrates the macOS permission checks shown at startup.
type ViewModel struct{}

// New creates a permissions ViewModel.
func New() *ViewModel { return &ViewModel{} }

// Ensure checks the required permissions, prints the result, and — if any
// permission is missing — opens the relevant System Settings panes to guide
// the user. Returns true when everything is granted.
func (vm *ViewModel) Ensure() bool {
	permview.Header()
	status := permsvc.Check()
	permview.Report(status)

	if status.AllGranted() {
		permview.Granted()
		return true
	}

	permview.Missing(status)

	if !status.ScreenRecording {
		_ = permsvc.OpenScreenRecordingPane()
	}
	if !status.Accessibility {
		_ = permsvc.OpenAccessibilityPane()
	}
	return false
}
