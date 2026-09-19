package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	cases := []struct {
		name, in, secret string
	}{
		{"telegram bot token", `cmd="echo 1234567890:AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsawQ"`, "AAHdqTcvCH1vGWJxfSeofSAs0K5PALDsawQ"},
		{"anthropic key", `input="use key sk-ant-api03-abcdefghijklmnopqrstuvwxyz"`, "abcdefghijklmnop"},
		{"openai project key", `cmd="export OPENAI=sk-proj-abcdefghijklmnopqrst"`, "sk-proj-abcdefghijklmnopqrst"},
		{"github token", `cmd="git clone https://ghp_abcdefghij1234567890@github.com/x/y"`, "ghp_abcdefghij1234567890"},
		{"github fine-grained token", `input="github_pat_11ABCDEFG0123456789_abcdef"`, "11ABCDEFG0123456789"},
		{"slack token", `input="xoxb-123456789012-abcdefghij"`, "xoxb-123456789012-abcdefghij"},
		{"google key", `input="AIzaSyA1234567890abcdefghijklmnopqrstu"`, "SyA1234567890abcdefghij"},
		{"aws access key", `cmd="aws configure set aws_access_key_id AKIAIOSFODNN7EXAMPLE"`, "AKIAIOSFODNN7EXAMPLE"},
		{"bearer header", `cmd="curl -H Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig https://api"`, "eyJhbGciOiJIUzI1NiJ9"},
		{"url credentials", `cmd="curl https://admin:hunter22@example.com/api"`, "hunter22"},
		{"password assignment", `cmd="mysql --password=hunter22 db"`, "hunter22"},
		{"env token", `cmd="export GITHUB_TOKEN=abc123def456"`, "abc123def456"},
		{"colon form", `input="my api_key: sup3rs3cret"`, "sup3rs3cret"},
		{"flag with value", `cmd="tool --token abc123def456 run"`, "abc123def456"},
		{"arabic context", `input="كلمة السر password=سرّي123"`, "سرّي123"},
		{"json key in %q", `params=` + fmt.Sprintf("%q", `{"api_key":"xyz12345"}`), "xyz12345"},
		{"authorization token header", `cmd="curl -H Authorization: token abcdef123456"`, "abcdef123456"},
	}
	for _, c := range cases {
		got := Redact(c.in)
		if strings.Contains(got, c.secret) {
			t.Errorf("%s: secret survived redaction: %q", c.name, got)
		}
		if !strings.Contains(got, Redacted) {
			t.Errorf("%s: expected %s marker in %q", c.name, Redacted, got)
		}
	}
}

func TestRedactKeepsLabels(t *testing.T) {
	cases := map[string]string{
		`GITHUB_TOKEN=abc123def456`:              `GITHUB_TOKEN=[REDACTED]`,
		`Authorization: Bearer abcdefgh12345678`: `Authorization: Bearer [REDACTED]`,
		`https://admin:hunter22@example.com`:     `https://admin:[REDACTED]@example.com`,
		`--password hunter22`:                    `--password [REDACTED]`,
	}
	for in, want := range cases {
		if got := Redact(in); got != want {
			t.Errorf("Redact(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRedactLeavesOrdinaryLinesAlone(t *testing.T) {
	lines := []string{
		`[2026-04-17 09:31:05] user=123456789 action=screenshot`,
		`[2026-04-17 09:32:11] user=123456789 action=shell cmd="git status"`,
		`[2026-04-17 09:33:00] user=123456789 action=file_send path="/home/user/report.pdf"`,
		`[2026-04-17 09:35:00] user=123456789 action=schedule_create id="task_001"`,
		`[2026-04-17 09:41:10] user=123456789 action=natural_language input="open Firefox and go to github.com"`,
		`[2026-04-17 09:41:10] user=123456789 action=natural_language input="افتح فايرفوكس وروح إلى github.com"`,
		`[2026-04-17 09:42:00] user=123456789 action=log_search query="token" matches=0`,
		`[2026-04-17 09:43:00] user=123456789 action=shell cmd="ls --sort=time"`,
		`[2026-04-17 09:44:00] user=123456789 action=shell cmd="curl https://example.com/path"`,
		`[2026-04-17 09:45:00] user=123456789 action=natural_language input="open basic settings for firefox"`,
		`[2026-04-17 09:46:00] user=123456789 action=natural_language input="count the token usage today"`,
		`[2026-04-17 09:47:00] user=123456789 action=file_send path="/home/user/tokens.txt"`,
	}
	for _, l := range lines {
		if got := Redact(l); got != l {
			t.Errorf("ordinary line was altered:\n in: %q\nout: %q", l, got)
		}
	}
}

// Log must redact before anything reaches disk.
func TestLogWritesRedactedLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	l := &Logger{file: f}
	if err := l.Log(42, "shell", `cmd="export GITHUB_TOKEN=abc123def456"`); err != nil {
		t.Fatal(err)
	}
	l.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "abc123def456") {
		t.Fatalf("secret was written to disk: %q", data)
	}
	if !strings.Contains(string(data), `action=shell cmd="export GITHUB_TOKEN=[REDACTED]"`) {
		t.Errorf("unexpected log line %q", data)
	}
}
