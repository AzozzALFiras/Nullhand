package command

// Command represents a parsed manual command from a Telegram message.
// Name is the command without the leading slash (e.g. "click", "screenshot").
// Args contains the remaining tokens split by whitespace.
// Raw is the original unmodified text.
type Command struct {
	Name string
	Args []string
	Raw  string
}
