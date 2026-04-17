//go:build !linux && !darwin

package permissions

// Stub for unsupported platforms.
type ViewModel struct{}

func New() *ViewModel { return &ViewModel{} }

func (vm *ViewModel) Ensure() bool {
	return false
}
