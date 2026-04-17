package bot

import (
	"github.com/AzozzALFiras/nullhand/internal/service/telegram"
)

// SendMenu sends the persistent shortcut toolbar with inline keyboard buttons.
// It is called after successful OTP unlock and on /menu or /start.
func SendMenu(tg *telegram.Client, chatID int64) error {
	keyboard := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "📸 Screenshot", CallbackData: "menu:screenshot"},
				{Text: "💻 System Info", CallbackData: "menu:sysinfo"},
			},
			{
				{Text: "📋 Clipboard", CallbackData: "menu:clipboard"},
				{Text: "🐚 Run Command", CallbackData: "menu:shell"},
			},
			{
				{Text: "📤 Send File", CallbackData: "menu:sendfile"},
				{Text: "📥 Downloads", CallbackData: "menu:downloads"},
			},
			{
				{Text: "🔍 Read Screen", CallbackData: "menu:ocr"},
				{Text: "🔒 Lock Bot", CallbackData: "menu:lock"},
			},
			{
				{Text: "❓ Help", CallbackData: "menu:help"},
			},
		},
	}

	text := "🖥️ Nullhand — Quick Actions"
	_, err := tg.SendMessageWithKeyboard(chatID, text, keyboard)
	return err
}
