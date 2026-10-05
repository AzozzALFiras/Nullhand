package files

import (
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cases := map[string]string{
		"~/Documents/x.pdf": filepath.Join(home, "Documents/x.pdf"),
		"~":                 home,
	}
	for in, want := range cases {
		got, err := ExpandPath(in)
		if err != nil {
			t.Fatalf("ExpandPath(%q): %v", in, err)
		}
		if got != want {
			t.Errorf("ExpandPath(%q) = %q, want %q", in, got, want)
		}
	}

	// Relative paths come back absolute so helper processes resolve them the
	// same way the bot does.
	got, err := ExpandPath("notes.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("ExpandPath should return an absolute path, got %q", got)
	}

	if _, err := ExpandPath("  "); err == nil {
		t.Error("an empty path must return an error")
	}
}

func TestExpandDirDefaultsToHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := ExpandDir("")
	if err != nil {
		t.Fatal(err)
	}
	if got != home {
		t.Errorf("ExpandDir(\"\") = %q, want the home directory %q", got, home)
	}
}
