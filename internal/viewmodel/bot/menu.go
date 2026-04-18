package bot

import (
	"github.com/iamakillah/Nullhand_Linux/internal/service/telegram"
)

// SendMenu sends the persistent reply keyboard toolbar to the chat.
// Buttons that map to slash commands send the slash command text directly.
// Buttons for interactive actions send a phrase detected in handleUpdate.
func SendMenu(tg *telegram.Client, chatID int64) error {
	keyboard := &telegram.ReplyKeyboardMarkup{
		Keyboard: [][]telegram.KeyboardButton{
			{
				{Text: "/screenshot"},
				{Text: "/status"},
			},
			{
				{Text: "/ocr"},
				{Text: "/paste"},
			},
			{
				{Text: "🐚 Run Command"},
				{Text: "📤 Send File"},
			},
			{
				{Text: "/ls /home/iam404/Downloads"},
				{Text: "🔒 Lock Bot"},
			},
			{
				{Text: "/help"},
			},
		},
		ResizeKeyboard: true,
		Persistent:     true,
	}
	return tg.SendMessageWithReplyKeyboard(chatID, "🖥️ Nullhand — Quick Actions", keyboard)
}
