package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/AzozzALFiras/nullhand/internal/model/message"
)

const apiBase = "https://api.telegram.org/bot"

// Client sends requests to the Telegram Bot API.
type Client struct {
	token      string
	httpClient *http.Client
}

// NewClient creates a Client for the given bot token.
func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// GetUpdates fetches pending updates starting from offset with a long-poll timeout.
func (c *Client) GetUpdates(offset int, timeout int) ([]message.Update, error) {
	url := fmt.Sprintf("%s%s/getUpdates?offset=%d&timeout=%d", apiBase, c.token, offset, timeout)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("getUpdates request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool             `json:"ok"`
		Result []message.Update `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("getUpdates decode failed: %w", err)
	}
	if !result.OK {
		return nil, fmt.Errorf("getUpdates: Telegram returned ok=false")
	}
	return result.Result, nil
}

// BotCommand represents a single bot command entry in the Telegram UI menu.
type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// SetMyCommands registers the command menu shown in Telegram clients.
// Pass commands in the order you want them displayed.
func (c *Client) SetMyCommands(commands []BotCommand) error {
	payload := map[string]any{"commands": commands}
	return c.post("setMyCommands", payload)
}

// InlineKeyboardButton is a single button in an inline keyboard.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

// InlineKeyboardMarkup is a grid of inline buttons attached to a message.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// SendMessage sends a plain text message to chatID.
func (c *Client) SendMessage(chatID int64, text string) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	return c.post("sendMessage", payload)
}

// SendMessageWithKeyboard sends a text message with inline keyboard buttons.
// Returns the message ID of the sent message for later editing.
func (c *Client) SendMessageWithKeyboard(chatID int64, text string, keyboard *InlineKeyboardMarkup) (int, error) {
	payload := map[string]any{
		"chat_id":      chatID,
		"text":         text,
		"parse_mode":   "HTML",
		"reply_markup": keyboard,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("sendMessage: marshal: %w", err)
	}

	url := fmt.Sprintf("%s%s/sendMessage", apiBase, c.token)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return 0, fmt.Errorf("sendMessage: request: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("sendMessage: decode: %w", err)
	}
	if !result.OK {
		return 0, fmt.Errorf("sendMessage: telegram returned ok=false")
	}
	return result.Result.MessageID, nil
}

// EditMessage edits an existing message's text and keyboard.
func (c *Client) EditMessage(chatID int64, messageID int, text string, keyboard *InlineKeyboardMarkup) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
		"parse_mode": "HTML",
	}
	if keyboard != nil {
		payload["reply_markup"] = keyboard
	}
	return c.post("editMessageText", payload)
}

// AnswerCallbackQuery acknowledges a callback query to remove the loading
// indicator on the button. Optional text shows a brief notification.
func (c *Client) AnswerCallbackQuery(callbackID string, text string) error {
	payload := map[string]any{
		"callback_query_id": callbackID,
	}
	if text != "" {
		payload["text"] = text
	}
	return c.post("answerCallbackQuery", payload)
}

// SendPhoto sends a PNG image with an optional caption to chatID.
func (c *Client) SendPhoto(chatID int64, imageData []byte, caption string) error {
	url := fmt.Sprintf("%s%s/sendPhoto", apiBase, c.token)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	_ = w.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	if caption != "" {
		_ = w.WriteField("caption", caption)
	}

	part, err := w.CreateFormFile("photo", "screenshot.png")
	if err != nil {
		return fmt.Errorf("sendPhoto: create form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(imageData)); err != nil {
		return fmt.Errorf("sendPhoto: copy image: %w", err)
	}
	w.Close()

	resp, err := c.httpClient.Post(url, w.FormDataContentType(), &body)
	if err != nil {
		return fmt.Errorf("sendPhoto request failed: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// SendDocument sends a file as a Telegram document using multipart/form-data.
// Signature matches the filetransfer.TelegramBot interface.
func (c *Client) SendDocument(chatID int64, data []byte, filename string) error {
	url := fmt.Sprintf("%s%s/sendDocument", apiBase, c.token)

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	_ = w.WriteField("chat_id", strconv.FormatInt(chatID, 10))
	part, err := w.CreateFormFile("document", filename)
	if err != nil {
		return fmt.Errorf("sendDocument: create form file: %w", err)
	}
	if _, err := io.Copy(part, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("sendDocument: copy file: %w", err)
	}
	w.Close()

	resp, err := c.httpClient.Post(url, w.FormDataContentType(), &body)
	if err != nil {
		return fmt.Errorf("sendDocument request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("sendDocument decode failed: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("sendDocument: telegram returned ok=false")
	}
	return nil
}

// DownloadFile fetches a file from Telegram by file_id.
// Returns raw bytes, the filename (last segment of file_path), and any error.
func (c *Client) DownloadFile(fileID string) ([]byte, string, error) {
	type tgFile struct {
		FilePath string `json:"file_path"`
	}
	var resp struct {
		OK     bool   `json:"ok"`
		Result tgFile `json:"result"`
	}

	getURL := fmt.Sprintf("%s%s/getFile?file_id=%s", apiBase, c.token, fileID)
	r, err := c.httpClient.Get(getURL)
	if err != nil {
		return nil, "", fmt.Errorf("getFile request: %w", err)
	}
	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		return nil, "", fmt.Errorf("getFile decode: %w", err)
	}
	if !resp.OK {
		return nil, "", fmt.Errorf("getFile: Telegram returned ok=false")
	}

	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", c.token, resp.Result.FilePath)
	fr, err := c.httpClient.Get(fileURL)
	if err != nil {
		return nil, "", fmt.Errorf("file download: %w", err)
	}
	defer fr.Body.Close()

	data, err := io.ReadAll(fr.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read file body: %w", err)
	}

	// Extract filename from the last path segment
	fp := resp.Result.FilePath
	filename := "downloaded_file"
	for i := len(fp) - 1; i >= 0; i-- {
		if fp[i] == '/' {
			filename = fp[i+1:]
			break
		}
	}

	return data, filename, nil
}

// SendTyping sends the "typing..." chat action.
func (c *Client) SendTyping(chatID int64) error {
	payload := map[string]any{
		"chat_id": chatID,
		"action":  "typing",
	}
	return c.post("sendChatAction", payload)
}

// DeleteMessage deletes a message by ID.
func (c *Client) DeleteMessage(chatID int64, messageID int) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
	}
	return c.post("deleteMessage", payload)
}

// post encodes payload as JSON and calls the given Telegram method.
func (c *Client) post(method string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("post %s: marshal: %w", method, err)
	}

	url := fmt.Sprintf("%s%s/%s", apiBase, c.token, method)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("post %s: request: %w", method, err)
	}
	defer resp.Body.Close()

	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("post %s: decode: %w", method, err)
	}
	if !result.OK {
		return fmt.Errorf("post %s: %s", method, result.Description)
	}
	return nil
}
