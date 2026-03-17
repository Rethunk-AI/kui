package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCreatesLayout(t *testing.T) {
	t.Parallel()

	base := filepath.Join(t.TempDir(), "storage")

	if err := Init(base); err != nil {
		t.Fatalf("init git layout: %v", err)
	}

	directories := []string{
		base,
		filepath.Join(base, "templates"),
		filepath.Join(base, "audit"),
		filepath.Join(base, "audit", "vm"),
		filepath.Join(base, "audit", "template"),
		filepath.Join(base, "audit", "wizard"),
	}

	for _, dir := range directories {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("expected directory %q to exist: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("expected %q to be a directory", dir)
		}
	}
}

func TestInitIdempotent(t *testing.T) {
	t.Parallel()

	base := filepath.Join(t.TempDir(), "storage")

	if err := Init(base); err != nil {
		t.Fatalf("first init: %v", err)
	}
	if err := Init(base); err != nil {
		t.Fatalf("second init (idempotent): %v", err)
	}
}

func TestInit_EmptyPath(t *testing.T) {
	t.Parallel()

	err := Init("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if !contains(err.Error(), "git base path is required") {
		t.Errorf("expected git base path error, got %v", err)
	}
}

func TestInit_WhitespacePath(t *testing.T) {
	t.Parallel()

	err := Init("   ")
	if err == nil {
		t.Fatal("expected error for whitespace path")
	}
	if !contains(err.Error(), "git base path is required") {
		t.Errorf("expected git base path error, got %v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return len(sub) == 0
}
