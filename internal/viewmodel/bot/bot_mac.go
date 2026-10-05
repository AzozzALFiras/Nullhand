package bot

import (
	"fmt"

	macsvc "github.com/AzozzALFiras/Nullhand/internal/service/mac"
	tgfmt "github.com/AzozzALFiras/Nullhand/internal/view/telegram"
)

// handleTrashCommand serves "/trash" with no argument. Emptying the Trash is
// irreversible, so it reports what is in there and waits for /yes — the same
// gate destructive shell commands go through.
func (vm *ViewModel) handleTrashCommand(chatID, userID int64) {
	state, err := macsvc.TrashStatus()
	if err != nil {
		vm.send(chatID, tgfmt.FailWith("trash", err))
		return
	}
	if state.Items == 0 {
		vm.send(chatID, state.Summary())
		return
	}

	vm.auditLog(userID, "trash_confirm_requested", fmt.Sprintf(`items=%d`, state.Items))
	vm.guard.SetPending(chatID, func() (string, error) {
		if err := macsvc.EmptyTrash(); err != nil {
			return "", err
		}
		vm.auditLog(userID, "trash_emptied", fmt.Sprintf(`items=%d`, state.Items))
		return tgfmt.OKWith(fmt.Sprintf("Trash emptied — %s freed.", state.Describe())), nil
	})
	vm.send(chatID, tgfmt.Confirm(fmt.Sprintf(
		"Empty the Trash: %s.\nThis cannot be undone.", state.Describe())))
}
