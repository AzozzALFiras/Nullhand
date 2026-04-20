package bot

import (
	"github.com/AzozzALFiras/Nullhand/internal/service/telegram"
)

// SendMenu sends the persistent reply keyboard toolbar to the chat.
// All button texts are friendly labels; handleUpdate maps them to actions.
func SendMenu(tg *telegram.Client, chatID int64) error {
	keyboard := &telegram.ReplyKeyboardMarkup{
		Keyboard: [][]telegram.KeyboardButton{
			{
				{Text: "📸 Screenshot"},
				{Text: "💻 System Info"},
			},
			{
				{Text: "🔍 Read Screen"},
				{Text: "📋 Clipboard"},
			},
			{
				{Text: "🐚 Run Command"},
				{Text: "📤 Send File"},
			},
			{
				{Text: "📥 Downloads"},
				{Text: "🔒 Lock Bot"},
			},
			{
				{Text: "❓ Help"},
			},
		},
		ResizeKeyboard: true,
		Persistent:     true,
	}
	return tg.SendMessageWithReplyKeyboard(chatID, "🖥️ Nullhand — Quick Actions", keyboard)
}
