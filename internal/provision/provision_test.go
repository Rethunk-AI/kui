package provision

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kui/kui/internal/prefix"
)

// requireUnderRoot fails if p is not rooted under root (after Clean).
func requireUnderRoot(t *testing.T, root, p string) {
	t.Helper()
	root = filepath.Clean(root)
	p = filepath.Clean(p)
	rel, err := filepath.Rel(root, p)
	if err != nil {
		t.Fatalf("Rel(%q, %q): %v", root, p, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("path %q is not under prefix root %q (rel=%q)", p, root, rel)
	}
}

func TestSelectPoolPath_emptyPrefix_envOverrideRaw(t *testing.T) {
	raw := "  /tmp/kui-test-pool  "
	t.Setenv("KUI_TEST_PROVISION_POOL_PATH", raw)
	got, err := SelectPoolPath("")
	if err != nil {
		t.Fatalf("SelectPoolPath: %v", err)
	}
	if got.Path != raw {
		t.Fatalf("Path: got %q want %q (legacy: untrimmed getenv)", got.Path, raw)
	}
}

func TestSelectPoolPath_emptyPrefix_noEnv_oneOfDefaults(t *testing.T) {
	t.Setenv("KUI_TEST_PROVISION_POOL_PATH", "")
	got, err := SelectPoolPath("")
	if err != nil {
		t.Fatalf("SelectPoolPath: %v", err)
	}
	if got.Path != DefaultPoolPath && got.Path != DefaultKuiPoolPath {
		t.Fatalf("Path %q not one of defaults %q / %q", got.Path, DefaultPoolPath, DefaultKuiPoolPath)
	}
}

func TestSelectPoolPath_nonEmptyPrefix_envAbsResolved(t *testing.T) {
	root := t.TempDir()
	t.Setenv("KUI_TEST_PROVISION_POOL_PATH", "  /opt/my-pool  ")
	want := prefix.Resolve(root, "/opt/my-pool")
	got, err := SelectPoolPath(root)
	if err != nil {
		t.Fatalf("SelectPoolPath: %v", err)
	}
	if got.Path != want {
		t.Fatalf("Path: got %q want %q", got.Path, want)
	}
	requireUnderRoot(t, root, got.Path)
}

func TestSelectPoolPath_nonEmptyPrefix_defaults_libvirtWhenNonEmpty(t *testing.T) {
	root := t.TempDir()
	t.Setenv("KUI_TEST_PROVISION_POOL_PATH", "")
	libvirt := prefix.Resolve(root, DefaultPoolPath)
	if err := os.MkdirAll(libvirt, 0o755); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(libvirt, "dummy.qcow2")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := SelectPoolPath(root)
	if err != nil {
		t.Fatalf("SelectPoolPath: %v", err)
	}
	if got.Path != libvirt {
		t.Fatalf("Path: got %q want %q", got.Path, libvirt)
	}
	requireUnderRoot(t, root, got.Path)
}

func TestSelectPoolPath_nonEmptyPrefix_defaults_kuiWhenLibvirtEmpty(t *testing.T) {
	root := t.TempDir()
	t.Setenv("KUI_TEST_PROVISION_POOL_PATH", "")
	libvirt := prefix.Resolve(root, DefaultPoolPath)
	if err := os.MkdirAll(libvirt, 0o755); err != nil {
		t.Fatal(err)
	}
	kui := prefix.Resolve(root, DefaultKuiPoolPath)
	got, err := SelectPoolPath(root)
	if err != nil {
		t.Fatalf("SelectPoolPath: %v", err)
	}
	if got.Path != kui {
		t.Fatalf("Path: got %q want %q", got.Path, kui)
	}
	requireUnderRoot(t, root, got.Path)
}

func TestSelectPoolPath_nonEmptyPrefix_defaults_kuiWhenLibvirtMissing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("KUI_TEST_PROVISION_POOL_PATH", "")
	kui := prefix.Resolve(root, DefaultKuiPoolPath)
	got, err := SelectPoolPath(root)
	if err != nil {
		t.Fatalf("SelectPoolPath: %v", err)
	}
	if got.Path != kui {
		t.Fatalf("Path: got %q want %q", got.Path, kui)
	}
	requireUnderRoot(t, root, got.Path)
}

func TestSelectPoolPath_whitespaceOnlyPrefix_likeLegacy(t *testing.T) {
	raw := "/strict/raw"
	t.Setenv("KUI_TEST_PROVISION_POOL_PATH", raw)
	got, err := SelectPoolPath("  \t  ")
	if err != nil {
		t.Fatalf("SelectPoolPath: %v", err)
	}
	if got.Path != raw {
		t.Fatalf("whitespace-only prefix should act as empty: got %q want %q", got.Path, raw)
	}
}
