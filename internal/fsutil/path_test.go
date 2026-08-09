package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveSafe(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := Resolve(root, "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(sub) {
		t.Fatalf("got %s want %s", got, sub)
	}

	if _, err := Resolve(root, "../etc"); err == nil {
		t.Fatal("expected escape error")
	}
	if _, err := Resolve(root, "/etc/passwd"); err == nil {
		t.Fatal("expected abs error")
	}
}

func TestRing(t *testing.T) {
	// ring is in session package — keep path tests only here
}
