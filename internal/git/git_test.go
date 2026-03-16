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
