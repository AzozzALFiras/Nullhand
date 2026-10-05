package bot

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf16"

	configmodel "github.com/AzozzALFiras/Nullhand/internal/model/config"
	msgmodel "github.com/AzozzALFiras/Nullhand/internal/model/message"
	tgfmt "github.com/AzozzALFiras/Nullhand/internal/view/telegram"
)

// These tests drive the real handleUpdate pipeline end to end. Telegram is
// replaced by fakeTelegram (via http.DefaultTransport, which the bot's HTTP
// client uses) and HOME points at a temp dir, so config, audit log and
// schedule files never touch the real ~/.nullhand.

const testUser = 42

type tgCall struct {
	method  string
	payload map[string]any
}

// fakeTelegram answers every Bot API call with ok and records it.
type fakeTelegram struct {
	mu    sync.Mutex
	calls []tgCall
}

func (f *fakeTelegram) RoundTrip(req *http.Request) (*http.Response, error) {
	var payload map[string]any
	if req.Body != nil {
		_ = json.NewDecoder(req.Body).Decode(&payload)
	}
	f.mu.Lock()
	f.calls = append(f.calls, tgCall{method: path.Base(req.URL.Path), payload: payload})
	f.mu.Unlock()
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"ok":true,"result":{"message_id":1}}`)),
		Request:    req,
	}, nil
}

func (f *fakeTelegram) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = nil
}

// texts returns the text of every message sent or edited, in order.
func (f *fakeTelegram) texts() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, c := range f.calls {
		if c.method == "sendMessage" || c.method == "editMessageText" {
			if s, ok := c.payload["text"].(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

func (f *fakeTelegram) lastText(t *testing.T) string {
	t.Helper()
	texts := f.texts()
	if len(texts) == 0 {
		t.Fatal("bot sent no message")
	}
	return texts[len(texts)-1]
}

// newUnlockedBot builds a real bot against the fake Telegram and unlocks it
// with the current OTP.
func newUnlockedBot(t *testing.T) (*ViewModel, *fakeTelegram) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	fake := &fakeTelegram{}
	orig := http.DefaultTransport
	http.DefaultTransport = fake
	t.Cleanup(func() { http.DefaultTransport = orig })

	vm, err := New(&configmodel.Config{TelegramToken: "test-token", AllowedUserID: testUser, AIProvider: "local"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(vm.Stop)

	vm.handleUpdate(textUpdate(vm.otp.CurrentCode()))
	if !vm.otp.IsUnlocked() {
		t.Fatal("setup: OTP unlock failed")
	}
	fake.reset()
	return vm, fake
}

func textUpdate(text string) msgmodel.Update {
	return msgmodel.Update{Message: &msgmodel.Message{
		From: &msgmodel.User{ID: testUser},
		Chat: msgmodel.Chat{ID: testUser},
		Text: text,
	}}
}

func tempFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "victim.txt")
	if err := os.WriteFile(p, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return p
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func TestFlowDestructiveShellWaitsForYes(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	victim := tempFile(t, "data")

	vm.handleUpdate(textUpdate("/shell rm " + victim))
	if !exists(victim) {
		t.Fatal("rm ran before /yes")
	}
	if msg := fake.lastText(t); !strings.Contains(msg, "Confirm destructive command") || !strings.Contains(msg, "/yes") {
		t.Fatalf("expected a confirmation prompt, got %q", msg)
	}

	vm.handleUpdate(textUpdate("/yes"))
	if exists(victim) {
		t.Fatal("/yes should have run the pending rm")
	}
}

func TestFlowNoCancelsPendingCommand(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	victim := tempFile(t, "data")

	vm.handleUpdate(textUpdate("/shell rm " + victim))
	vm.handleUpdate(textUpdate("/no"))
	if msg := fake.lastText(t); !strings.Contains(msg, "cancelled") {
		t.Errorf("expected cancellation reply, got %q", msg)
	}

	vm.handleUpdate(textUpdate("/yes"))
	if !exists(victim) {
		t.Fatal("a cancelled command must not run on a later /yes")
	}
	if msg := fake.lastText(t); !strings.Contains(msg, "No pending action") {
		t.Errorf("expected 'nothing pending' reply, got %q", msg)
	}
}

func TestFlowSafeShellRunsImmediately(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	vm.handleUpdate(textUpdate("/shell echo hello-nullhand"))
	if msg := fake.lastText(t); !strings.Contains(msg, "hello-nullhand") {
		t.Errorf("safe command output should be sent straight away, got %q", msg)
	}
}

func TestFlowScheduledDestructiveCommandNeedsYes(t *testing.T) {
	vm, fake := newUnlockedBot(t)

	vm.handleUpdate(textUpdate("every day at 2am run rm -rf /tmp/nullhand-flow-test"))
	if n := len(vm.scheduler.List()); n != 0 {
		t.Fatalf("task must not be created before /yes, found %d", n)
	}
	if msg := fake.lastText(t); !strings.Contains(msg, "Confirm destructive command") {
		t.Fatalf("expected a confirmation prompt, got %q", msg)
	}

	vm.handleUpdate(textUpdate("/yes"))
	if n := len(vm.scheduler.List()); n != 1 {
		t.Fatalf("/yes should create the task, found %d", n)
	}

	// The scheduler persists in the background; wait for the write so the
	// task is known to survive a restart (and so it finishes before the temp
	// HOME is removed).
	home, _ := os.UserHomeDir()
	schedule := filepath.Join(home, ".nullhand", "schedule.json")
	deadline := time.Now().Add(2 * time.Second)
	for {
		if data, err := os.ReadFile(schedule); err == nil && strings.Contains(string(data), "rm -rf /tmp/nullhand-flow-test") {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("confirmed task was never persisted to schedule.json")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestFlowIdleSessionRelocks(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	vm.otp.SetIdleTimeout(time.Millisecond)
	time.Sleep(20 * time.Millisecond)

	vm.handleUpdate(textUpdate("/shell echo should-not-run"))
	if vm.otp.IsUnlocked() {
		t.Fatal("session should have locked after the idle timeout")
	}
	for _, msg := range fake.texts() {
		if strings.Contains(msg, "should-not-run") {
			t.Fatal("the message that found the session idle must not be executed")
		}
	}
	if msg := fake.lastText(t); !strings.Contains(msg, "inactivity") {
		t.Errorf("user should be told why the bot locked, got %q", msg)
	}
}

func TestFlowLockedBotIgnoresInlineButtons(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	vm.otp.Lock()
	fake.reset()

	vm.handleUpdate(msgmodel.Update{CallbackQuery: &msgmodel.CallbackQuery{
		ID:      "cb1",
		From:    &msgmodel.User{ID: testUser},
		Message: &msgmodel.Message{Chat: msgmodel.Chat{ID: testUser}},
		Data:    "menu:sysinfo",
	}})

	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.calls) != 1 || fake.calls[0].method != "answerCallbackQuery" {
		t.Fatalf("a locked bot must only answer the button with a notice, got calls %+v", fake.calls)
	}
	if text, _ := fake.calls[0].payload["text"].(string); !strings.Contains(text, "locked") {
		t.Errorf("button notice should say the bot is locked, got %q", text)
	}
}

func TestFlowLongOutputIsSplit(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	big := tempFile(t, strings.Repeat("0123456789 abcdefghij <tag> & more\n", 400)) // ~14 KB

	vm.handleUpdate(textUpdate("/shell cat " + big))

	texts := fake.texts()[1:] // skip the "⏳ Working..." placeholder
	if len(texts) < 3 {
		t.Fatalf("~14 KB of output should arrive as several messages, got %d", len(texts))
	}
	for i, msg := range texts {
		if n := len(utf16.Encode([]rune(msg))); n > tgfmt.MaxMessageLen {
			t.Errorf("message %d is %d UTF-16 units, over Telegram's limit", i, n)
		}
		if !strings.HasPrefix(msg, "<pre>") || !strings.HasSuffix(msg, "</pre>") {
			t.Errorf("message %d must be a complete <pre> block", i)
		}
	}
}

func TestFlowAuditLogRedactsSecrets(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	vm.handleUpdate(textUpdate("/shell echo GITHUB_TOKEN=abc123def456"))

	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(filepath.Join(home, ".nullhand", "audit.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "abc123def456") {
		t.Fatalf("secret reached the audit log on disk:\n%s", data)
	}
	if !strings.Contains(string(data), "GITHUB_TOKEN=[REDACTED]") {
		t.Errorf("expected a redacted shell entry, got:\n%s", data)
	}

	fake.reset()
	vm.handleUpdate(textUpdate("/log"))
	if msg := fake.lastText(t); strings.Contains(msg, "abc123def456") || !strings.Contains(msg, "<pre>") {
		t.Errorf("/log must show redacted entries in an HTML code block, got %q", msg)
	}
}

// A slash command whose argument happens to contain "send" must reach its own
// handler instead of the natural-language file sender.
func TestFlowSlashCommandWithSendInPathIsNotHijacked(t *testing.T) {
	vm, fake := newUnlockedBot(t)
	vm.handleUpdate(textUpdate("/shell echo /var/log/sendmail.log"))

	msg := fake.lastText(t)
	if !strings.Contains(msg, "/var/log/sendmail.log") {
		t.Errorf("the shell command should have run, got %q", msg)
	}
	if strings.Contains(msg, "Failed to send") || strings.Contains(msg, "no such file") {
		t.Errorf("message was routed to the file sender: %q", msg)
	}
}
