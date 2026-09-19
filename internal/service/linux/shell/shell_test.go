package shell

import (
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestRunRejectsCommandsOutsideWhitelist(t *testing.T) {
	_, err := Run("sleep 1")
	if err == nil || !strings.Contains(err.Error(), "not in the allowed list") {
		t.Fatalf("sleep is not whitelisted and must be rejected, got err=%v", err)
	}
}

func TestRunRejectsEmptyCommand(t *testing.T) {
	if _, err := Run("   "); err == nil {
		t.Fatal("blank command must return an error")
	}
}

func TestRunWhitelistedCommand(t *testing.T) {
	out, err := Run("echo hello nullhand")
	if err != nil {
		t.Fatalf("echo should succeed: %v", err)
	}
	if out != "hello nullhand" {
		t.Errorf("unexpected output %q", out)
	}
}

func TestExecuteStopsAtTimeoutAndKeepsPartialOutput(t *testing.T) {
	start := time.Now()
	out, err := execute([]string{"sh", "-c", "echo started; sleep 10"}, 300*time.Millisecond, MaxOutputBytes)
	elapsed := time.Since(start)

	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("expected ErrTimeout, got %v", err)
	}
	if elapsed > 3*time.Second {
		t.Errorf("command should be stopped near the 300ms deadline, took %s", elapsed)
	}
	if out != "started" {
		t.Errorf("output produced before the kill must be kept, got %q", out)
	}
}

func TestExecuteReportsNonZeroExitWithoutTimeout(t *testing.T) {
	out, err := execute([]string{"sh", "-c", "echo oops; exit 3"}, 5*time.Second, MaxOutputBytes)
	if err == nil {
		t.Fatal("exit status 3 must be reported as an error")
	}
	if errors.Is(err, ErrTimeout) {
		t.Errorf("a normal failure must not be reported as a timeout: %v", err)
	}
	if out != "oops" {
		t.Errorf("stdout of the failing command must be returned, got %q", out)
	}
}

func TestExecuteCapsOutput(t *testing.T) {
	out, err := execute([]string{"sh", "-c", "yes nullhand | head -c 100000"}, 5*time.Second, 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "output truncated after 1000 bytes") {
		t.Errorf("truncated output must carry a note, got tail %q", out[len(out)-60:])
	}
	if body := strings.SplitN(out, "\n…", 2)[0]; len(body) > 1000 {
		t.Errorf("kept %d bytes, want at most 1000", len(body))
	}
}

func TestCappedBufferDropsPartialRunes(t *testing.T) {
	// "مرحبا" is 10 bytes (2 per letter); a 5-byte cap splits the third letter.
	b := &cappedBuffer{max: 5}
	_, _ = b.Write([]byte("مرحبا"))
	got := b.String()
	if !utf8.ValidString(got) {
		t.Fatalf("output must stay valid UTF-8, got %q", got)
	}
	if !strings.HasPrefix(got, "مر") {
		t.Errorf("the two complete letters must be kept, got %q", got)
	}
}

func TestCappedBufferAlwaysReportsFullWrite(t *testing.T) {
	b := &cappedBuffer{max: 3}
	n, err := b.Write([]byte("abcdef"))
	if n != 6 || err != nil {
		t.Errorf("Write must claim the whole slice so the child never gets EPIPE, got n=%d err=%v", n, err)
	}
	if n, _ := b.Write([]byte("more")); n != 4 {
		t.Errorf("writes after the cap must also report success, got n=%d", n)
	}
}

func TestSetTimeout(t *testing.T) {
	t.Cleanup(func() { SetTimeout(0) })

	if Timeout() != DefaultTimeout {
		t.Fatalf("default should be %s, got %s", DefaultTimeout, Timeout())
	}
	SetTimeout(5 * time.Second)
	if Timeout() != 5*time.Second {
		t.Errorf("SetTimeout(5s) not applied, got %s", Timeout())
	}
	SetTimeout(-time.Second)
	if Timeout() != DefaultTimeout {
		t.Errorf("negative timeout must restore the default, got %s", Timeout())
	}
}
