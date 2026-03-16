package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const validJWTSecret = "0123456789abcdef0123456789abcdef"

func TestLoad(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
    keyfile:
jwt_secret: "` + validJWTSecret + `"
`)

	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.DefaultHost != "local" {
		t.Fatalf("expected default host local, got %q", cfg.DefaultHost)
	}
	if cfg.DefaultNameTemplate != "{source}" {
		t.Fatalf("expected default name template, got %q", cfg.DefaultNameTemplate)
	}
	if cfg.DB.Path != "/var/lib/kui/kui.db" {
		t.Fatalf("expected db path /var/lib/kui/kui.db, got %q", cfg.DB.Path)
	}
	if cfg.Git.Path != "/var/lib/kui" {
		t.Fatalf("expected git path /var/lib/kui, got %q", cfg.Git.Path)
	}
	if cfg.Session.Timeout != Duration(24*time.Hour) {
		t.Fatalf("expected session timeout 24h, got %s", cfg.Session.Timeout.String())
	}
	if len(cfg.CORS.AllowedOrigins) != 1 || cfg.CORS.AllowedOrigins[0] != "http://localhost:5173" {
		t.Fatalf("unexpected cors origins: %v", cfg.CORS.AllowedOrigins)
	}
	if cfg.VMDefaults.CPU != 2 {
		t.Fatalf("expected default vm cpu 2, got %d", cfg.VMDefaults.CPU)
	}
	if cfg.VMDefaults.RAMMB != 2048 {
		t.Fatalf("expected default vm ram_mb 2048, got %d", cfg.VMDefaults.RAMMB)
	}
	if cfg.VMDefaults.Network != "default" {
		t.Fatalf("expected default vm network default, got %q", cfg.VMDefaults.Network)
	}
}

func TestLoadRejectsMissingHosts(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "missing-hosts.yaml")
	configContent := []byte(`jwt_secret: "` + validJWTSecret + `"
`)

	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(configPath); err == nil {
		t.Fatal("expected missing hosts error")
	}
}

func TestLoadRejectsMalformedKUI_SECURE_COOKIES(t *testing.T) {
	// Do not run in parallel: env var affects other tests
	t.Cleanup(func() { os.Unsetenv("KUI_SECURE_COOKIES") })
	if err := os.Setenv("KUI_SECURE_COOKIES", "invalid"); err != nil {
		t.Fatalf("setenv: %v", err)
	}

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "` + validJWTSecret + `"
`)
	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(configPath); err == nil {
		t.Fatal("expected KUI_SECURE_COOKIES validation error")
	}
}

func TestHostKeyfileEnvVarUsesKUI_HOST_Prefix(t *testing.T) {
	t.Parallel()

	if got := hostKeyfileEnvVar("local"); got != "KUI_HOST_LOCAL_KEYFILE" {
		t.Fatalf("hostKeyfileEnvVar(local) = %q, want KUI_HOST_LOCAL_KEYFILE", got)
	}
	if got := hostKeyfileEnvVar("remote1"); got != "KUI_HOST_REMOTE1_KEYFILE" {
		t.Fatalf("hostKeyfileEnvVar(remote1) = %q, want KUI_HOST_REMOTE1_KEYFILE", got)
	}
}

func TestLoadRejectsQemuSSHWithoutKeyfile(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_SECURE_COOKIES")
	_ = os.Unsetenv("KUI_HOST_REMOTE_KEYFILE")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`hosts:
  - id: remote
    uri: qemu+ssh://user@host/system
jwt_secret: "` + validJWTSecret + `"
`)

	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(configPath); err == nil {
		t.Fatal("expected qemu+ssh keyfile validation error")
	}
}

func TestLoadAcceptsQemuSSHWithKeyfileInConfig(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_SECURE_COOKIES")
	_ = os.Unsetenv("KUI_HOST_REMOTE_KEYFILE")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`hosts:
  - id: remote
    uri: qemu+ssh://user@host/system
    keyfile: /path/to/key
jwt_secret: "` + validJWTSecret + `"
`)

	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(configPath); err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}
}

func TestLoadAcceptsQemuSSHWithKeyfileInEnv(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() { os.Unsetenv("KUI_HOST_REMOTE_KEYFILE") })
	_ = os.Unsetenv("KUI_SECURE_COOKIES")
	if err := os.Setenv("KUI_HOST_REMOTE_KEYFILE", "/env/key"); err != nil {
		t.Fatalf("setenv: %v", err)
	}

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`hosts:
  - id: remote
    uri: qemu+ssh://user@host/system
jwt_secret: "` + validJWTSecret + `"
`)

	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(configPath); err != nil {
		t.Fatalf("expected load to succeed with env keyfile: %v", err)
	}
}

func TestLoadAcceptsQemuSSHWithKeyfileInURI(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_SECURE_COOKIES")
	_ = os.Unsetenv("KUI_HOST_REMOTE_KEYFILE")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`hosts:
  - id: remote
    uri: qemu+ssh://user@host/system?keyfile=/uri/key
jwt_secret: "` + validJWTSecret + `"
`)

	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := Load(configPath); err != nil {
		t.Fatalf("expected load to succeed with keyfile in URI: %v", err)
	}
}
