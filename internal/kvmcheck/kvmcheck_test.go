package kvmcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckKVMWithPaths_KVMExists(t *testing.T) {
	t.Parallel()
	if _, err := os.Stat("/dev/kvm"); err != nil {
		t.Skip("skipping: /dev/kvm not available (e.g. in container)")
	}

	ok, suggestion, err := checkKVMWithPaths("/dev/kvm", "/etc/os-release")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Error("expected ok=true when /dev/kvm exists")
	}
	if suggestion != "" {
		t.Errorf("expected empty suggestion when ok, got %q", suggestion)
	}
}

func TestCheckKVMWithPaths_KVMMissing(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	devPath := filepath.Join(tmp, "nonexistent")

	tests := []struct {
		name          string
		osReleaseBody string
		wantContains  string
	}{
		{"ubuntu", "ID=ubuntu\n", "apt install"},
		{"debian", "ID=debian\n", "apt install"},
		{"fedora", "ID=fedora\n", "dnf install"},
		{"rhel", "ID=rhel\n", "dnf install"},
		{"centos", "ID=centos\n", "dnf install"},
		{"rocky", "ID=rocky\n", "dnf install"},
		{"alma", "ID=alma\n", "dnf install"},
		{"arch", "ID=arch\n", "pacman"},
		{"opensuse", "ID=opensuse\n", "zypper"},
		{"unknown", "ID=unknown\n", "KVM not available"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			releasePath := filepath.Join(tmp, "os-release-"+tt.name)
			if err := os.WriteFile(releasePath, []byte(tt.osReleaseBody), 0o644); err != nil {
				t.Fatalf("create os-release: %v", err)
			}

			ok, suggestion, err := checkKVMWithPaths(devPath, releasePath)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok {
				t.Error("expected ok=false when /dev/kvm missing")
			}
			if !strings.Contains(suggestion, tt.wantContains) {
				t.Errorf("suggestion %q does not contain %q", suggestion, tt.wantContains)
			}
		})
	}
}

func TestCheckKVMWithPaths_OSReleaseMissing(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	devPath := filepath.Join(tmp, "nonexistent")
	osRelease := filepath.Join(tmp, "nonexistent-release")

	ok, suggestion, err := checkKVMWithPaths(devPath, osRelease)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Error("expected ok=false")
	}
	if !strings.Contains(suggestion, "KVM not available") {
		t.Errorf("expected generic suggestion when os-release missing, got %q", suggestion)
	}
}
