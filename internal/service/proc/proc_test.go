package proc

import (
	"strings"
	"testing"
	"time"
)

func TestRunReturnsTrimmedStdout(t *testing.T) {
	out, err := Run(5*time.Second, "echo", "hello", "proc")
	if err != nil {
		t.Fatal(err)
	}
	if out != "hello proc" {
		t.Errorf("got %q", out)
	}
}

func TestRunTimesOut(t *testing.T) {
	start := time.Now()
	_, err := Run(200*time.Millisecond, "sleep", "10")
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("expected a timeout error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Errorf("waited %s, the deadline should have cut it short", elapsed)
	}
}

func TestRunPutsStderrInTheError(t *testing.T) {
	_, err := Run(5*time.Second, "sh", "-c", "echo first line >&2; echo second >&2; exit 1")
	if err == nil {
		t.Fatal("a failing command must return an error")
	}
	if !strings.Contains(err.Error(), "first line") {
		t.Errorf("stderr should explain the failure, got %v", err)
	}
	if strings.Contains(err.Error(), "second") {
		t.Errorf("only the first stderr line belongs in a chat reply, got %v", err)
	}
}

func TestRunKeepsPartialOutputOnFailure(t *testing.T) {
	out, err := Run(5*time.Second, "sh", "-c", "echo partial; exit 2")
	if err == nil {
		t.Fatal("expected an error")
	}
	if out != "partial" {
		t.Errorf("output produced before the failure must be returned, got %q", out)
	}
}

func TestRunMissingBinary(t *testing.T) {
	if _, err := Run(time.Second, "nullhand-does-not-exist"); err == nil {
		t.Error("a missing binary must return an error")
	}
}

func TestFirstLine(t *testing.T) {
	for in, want := range map[string]string{
		"one":            "one",
		"one\ntwo":       "one",
		"  one  \ntwo  ": "one",
		"":               "",
	} {
		if got := FirstLine(in); got != want {
			t.Errorf("FirstLine(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLines(t *testing.T) {
	got := Lines("/a/b\n\n  /c/d  \n")
	if len(got) != 2 || got[0] != "/a/b" || got[1] != "/c/d" {
		t.Errorf("got %q", got)
	}
}
