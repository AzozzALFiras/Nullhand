package message

// Update is a single Telegram update object.
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

// Message is a Telegram message object.
type Message struct {
	MessageID int    `json:"message_id"`
	From      *User  `json:"from"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	Date      int64  `json:"date"`
}

// User represents a Telegram user.
type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}
