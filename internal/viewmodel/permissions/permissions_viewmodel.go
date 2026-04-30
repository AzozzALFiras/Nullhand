package permissions

import (
	permsvc "github.com/AzozzALFiras/Nullhand/internal/service/linux/permissions"
	permview "github.com/AzozzALFiras/Nullhand/internal/view/permissions"
)

// ViewModel orchestrates the desktop capability checks shown at startup.
type ViewModel struct{}

// New creates a permissions ViewModel.
func New() *ViewModel { return &ViewModel{} }

// Ensure checks the required capabilities, prints the result, and — if any
// capability is missing — opens the relevant settings panes to guide the user.
// Returns true when everything is available.
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
