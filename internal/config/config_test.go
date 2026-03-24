package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kui/kui/internal/prefix"
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

	cfg, err := Load(configPath, "")
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

func TestLoad_PrefixNormalizesYAMLPathsUnderTempDir(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	root := t.TempDir()
	logical := "/etc/kui/config.yaml"
	physical := filepath.Join(root, "etc", "kui", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(physical), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
db:
  path: /var/lib/kui/kui.db
git:
  path: /var/lib/kui
jwt_secret: "` + validJWTSecret + `"
`)
	if err := os.WriteFile(physical, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(logical, root)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	wantDB := prefix.Resolve(root, "/var/lib/kui/kui.db")
	wantGit := prefix.Resolve(root, "/var/lib/kui")
	if cfg.DB.Path != wantDB {
		t.Errorf("DB.Path = %q, want %q", cfg.DB.Path, wantDB)
	}
	if cfg.Git.Path != wantGit {
		t.Errorf("Git.Path = %q, want %q", cfg.Git.Path, wantGit)
	}
	if rel, err := filepath.Rel(root, cfg.DB.Path); err != nil || strings.HasPrefix(rel, "..") {
		t.Errorf("DB.Path %q is not under prefix %q (rel=%q, err=%v)", cfg.DB.Path, root, rel, err)
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

	if _, err := Load(configPath, ""); err == nil {
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

	if _, err := Load(configPath, ""); err == nil {
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
	// Do not run in parallel: env var affects other tests
	t.Cleanup(func() { os.Unsetenv("KUI_HOST_REMOTE_KEYFILE") })
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

	if _, err := Load(configPath, ""); err == nil {
		t.Fatal("expected qemu+ssh keyfile validation error")
	}
}

func TestLoadAcceptsQemuSSHWithKeyfileInConfig(t *testing.T) {
	// Do not run in parallel: env var affects other tests
	t.Cleanup(func() { os.Unsetenv("KUI_HOST_REMOTE_KEYFILE") })
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

	if _, err := Load(configPath, ""); err != nil {
		t.Fatalf("expected load to succeed: %v", err)
	}
}

func TestLoadAcceptsQemuSSHWithKeyfileInEnv(t *testing.T) {
	// Do not run in parallel: env var affects other tests
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

	if _, err := Load(configPath, ""); err != nil {
		t.Fatalf("expected load to succeed with env keyfile: %v", err)
	}
}

func TestLoadAcceptsQemuSSHWithKeyfileInURI(t *testing.T) {
	// Do not run in parallel: env var affects other tests
	t.Cleanup(func() { os.Unsetenv("KUI_HOST_REMOTE_KEYFILE") })
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

	if _, err := Load(configPath, ""); err != nil {
		t.Fatalf("expected load to succeed with keyfile in URI: %v", err)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	_, err := Load("/nonexistent/path/config.yaml", "")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "read config") {
		t.Errorf("expected read config error, got %v", err)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "bad.yaml")
	if err := os.WriteFile(configPath, []byte("hosts: [invalid: yaml: here"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := Load(configPath, "")
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("expected unmarshal error, got %v", err)
	}
}

func TestLoad_InvalidDuration(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
session:
  timeout: "not-a-duration"
jwt_secret: "` + validJWTSecret + `"
`)
	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := Load(configPath, "")
	if err == nil {
		t.Fatal("expected duration parse error")
	}
	if !strings.Contains(err.Error(), "duration") && !strings.Contains(err.Error(), "invalid") {
		t.Errorf("expected duration/invalid error, got %v", err)
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{"duplicate host id", `hosts:
  - id: a
    uri: qemu:///system
  - id: a
    uri: qemu:///system
jwt_secret: "` + validJWTSecret + `"
`, "duplicate"},
		{"empty host id", `hosts:
  - id: ""
    uri: qemu:///system
jwt_secret: "` + validJWTSecret + `"
`, "host id"},
		{"missing uri", `hosts:
  - id: local
    uri: ""
jwt_secret: "` + validJWTSecret + `"
`, "missing uri"},
		{"invalid default_host", `hosts:
  - id: local
    uri: qemu:///system
default_host: "nonexistent"
jwt_secret: "` + validJWTSecret + `"
`, "invalid default_host"},
		{"session timeout zero", `hosts:
  - id: local
    uri: qemu:///system
session:
  timeout: "-1s"
jwt_secret: "` + validJWTSecret + `"
`, "positive duration"},
		{"graceful_stop negative", `hosts:
  - id: local
    uri: qemu:///system
vm_lifecycle:
  graceful_stop_timeout: "-1s"
jwt_secret: "` + validJWTSecret + `"
`, "must not be negative"},
		{"cpu zero", `hosts:
  - id: local
    uri: qemu:///system
vm_defaults:
  cpu: -1
  ram_mb: 2048
  network: default
jwt_secret: "` + validJWTSecret + `"
`, "cpu must be greater"},
		{"ram_mb zero", `hosts:
  - id: local
    uri: qemu:///system
vm_defaults:
  cpu: 2
  ram_mb: -1
  network: default
jwt_secret: "` + validJWTSecret + `"
`, "ram_mb"},
		{"network empty", `hosts:
  - id: local
    uri: qemu:///system
vm_defaults:
  cpu: 2
  ram_mb: 2048
  network: "   "
jwt_secret: "` + validJWTSecret + `"
`, "network is required"},
		{"jwt_secret empty", `hosts:
  - id: local
    uri: qemu:///system
jwt_secret: ""
`, "jwt_secret"},
		{"jwt_secret too short", `hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "short"
`, "at least 32"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "config.yaml")
			if err := os.WriteFile(configPath, []byte(tt.content), 0o600); err != nil {
				t.Fatalf("write: %v", err)
			}
			_, err := Load(configPath, "")
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestLoad_ApplyEnvOverrides(t *testing.T) {
	// Do not run in parallel: env vars affect other tests
	t.Cleanup(func() {
		os.Unsetenv("KUI_DB_PATH")
		os.Unsetenv("KUI_GIT_PATH")
		os.Unsetenv("KUI_DEFAULT_HOST")
		os.Unsetenv("KUI_DEFAULT_POOL")
		os.Unsetenv("KUI_SESSION_TIMEOUT")
		os.Unsetenv("KUI_JWT_SECRET")
		os.Unsetenv("KUI_CORS_ORIGINS")
		os.Unsetenv("KUI_SECURE_COOKIES")
	})
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	os.Setenv("KUI_DB_PATH", "/custom/db/path")
	os.Setenv("KUI_GIT_PATH", "/custom/git/path")
	os.Setenv("KUI_DEFAULT_HOST", "remote")
	os.Setenv("KUI_DEFAULT_POOL", "default")
	os.Setenv("KUI_SESSION_TIMEOUT", "1h")
	os.Setenv("KUI_JWT_SECRET", "0123456789abcdef0123456789abcdef01234567")
	os.Setenv("KUI_CORS_ORIGINS", "https://a.example.com, https://b.example.com")
	os.Setenv("KUI_SECURE_COOKIES", "false")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
  - id: remote
    uri: qemu:///system
jwt_secret: "` + validJWTSecret + `"
`)
	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := Load(configPath, "")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.DB.Path != "/custom/db/path" {
		t.Errorf("DB.Path = %q, want /custom/db/path", cfg.DB.Path)
	}
	if cfg.Git.Path != "/custom/git/path" {
		t.Errorf("Git.Path = %q, want /custom/git/path", cfg.Git.Path)
	}
	if cfg.DefaultHost != "remote" {
		t.Errorf("DefaultHost = %q, want remote", cfg.DefaultHost)
	}
	if cfg.DefaultPool == nil || *cfg.DefaultPool != "default" {
		t.Errorf("DefaultPool = %v", cfg.DefaultPool)
	}
	if cfg.Session.Timeout != Duration(time.Hour) {
		t.Errorf("Session.Timeout = %s, want 1h", cfg.Session.Timeout.String())
	}
	if cfg.JWTSecret != "0123456789abcdef0123456789abcdef01234567" {
		t.Errorf("JWTSecret not overridden")
	}
	if len(cfg.CORS.AllowedOrigins) != 2 ||
		cfg.CORS.AllowedOrigins[0] != "https://a.example.com" ||
		cfg.CORS.AllowedOrigins[1] != "https://b.example.com" {
		t.Errorf("CORS.AllowedOrigins = %v", cfg.CORS.AllowedOrigins)
	}
	if cfg.Session.SecureCookies == nil || *cfg.Session.SecureCookies {
		t.Errorf("SecureCookies = %v, want false", cfg.Session.SecureCookies)
	}
}

func TestLoad_ApplyEnvOverrides_InvalidSessionTimeout(t *testing.T) {
	t.Cleanup(func() { os.Unsetenv("KUI_SESSION_TIMEOUT") })
	_ = os.Unsetenv("KUI_SECURE_COOKIES")
	os.Setenv("KUI_SESSION_TIMEOUT", "invalid")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "` + validJWTSecret + `"
`)
	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := Load(configPath, "")
	if err == nil {
		t.Fatal("expected KUI_SESSION_TIMEOUT error")
	}
	if !strings.Contains(err.Error(), "KUI_SESSION_TIMEOUT") {
		t.Errorf("expected KUI_SESSION_TIMEOUT in error, got %v", err)
	}
}

func TestLoadWithArgs(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_SECURE_COOKIES")
	_ = os.Unsetenv("KUI_CONFIG")
	_ = os.Unsetenv("KUI_PREFIX")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "` + validJWTSecret + `"
`)
	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, path, err := LoadWithArgs([]string{"--config", configPath})
	if err != nil {
		t.Fatalf("LoadWithArgs: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config")
	}
	if path != configPath {
		t.Errorf("path = %q, want %q", path, configPath)
	}
}

func TestLoadWithArgs_EnvOverride(t *testing.T) {
	t.Cleanup(func() {
		os.Unsetenv("KUI_CONFIG")
		os.Unsetenv("KUI_PREFIX")
	})
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "` + validJWTSecret + `"
`)
	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	os.Setenv("KUI_CONFIG", configPath)

	cfg, path, err := LoadWithArgs([]string{})
	if err != nil {
		t.Fatalf("LoadWithArgs: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config")
	}
	if path != configPath {
		t.Errorf("path = %q, want %q", path, configPath)
	}
}

func TestLoadWithArgs_ConfigWinsOverKUI_CONFIG(t *testing.T) {
	t.Cleanup(func() {
		os.Unsetenv("KUI_CONFIG")
		os.Unsetenv("KUI_PREFIX")
	})
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	tempDir := t.TempDir()
	pathA := filepath.Join(tempDir, "a.yaml")
	pathB := filepath.Join(tempDir, "b.yaml")
	contentA := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "` + validJWTSecret + `"
`)
	contentB := []byte(`hosts:
  - id: remote
    uri: qemu:///system
default_host: remote
jwt_secret: "` + validJWTSecret + `"
`)
	if err := os.WriteFile(pathA, contentA, 0o600); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.WriteFile(pathB, contentB, 0o600); err != nil {
		t.Fatalf("write b: %v", err)
	}
	if err := os.Setenv("KUI_CONFIG", pathA); err != nil {
		t.Fatalf("setenv KUI_CONFIG: %v", err)
	}

	cfg, resolved, err := LoadWithArgs([]string{"--config", pathB})
	if err != nil {
		t.Fatalf("LoadWithArgs: %v", err)
	}
	if resolved != pathB {
		t.Errorf("resolved path = %q, want %q", resolved, pathB)
	}
	if cfg.DefaultHost != "remote" {
		t.Errorf("DefaultHost = %q, want remote (config B)", cfg.DefaultHost)
	}
}

func TestLoadWithOptions_BootstrapPrefixNormalizesYAMLPaths(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	root := t.TempDir()
	logical := "/etc/kui/config.yaml"
	physical := filepath.Join(root, "etc", "kui", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(physical), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
db:
  path: /var/lib/kui/kui.db
git:
  path: /var/lib/kui
jwt_secret: "` + validJWTSecret + `"
`)
	if err := os.WriteFile(physical, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadWithOptions(logical, LoadOptions{Prefix: root})
	if err != nil {
		t.Fatalf("LoadWithOptions: %v", err)
	}
	wantDB := prefix.Resolve(root, "/var/lib/kui/kui.db")
	wantGit := prefix.Resolve(root, "/var/lib/kui")
	if cfg.DB.Path != wantDB {
		t.Errorf("DB.Path = %q, want %q", cfg.DB.Path, wantDB)
	}
	if cfg.Git.Path != wantGit {
		t.Errorf("Git.Path = %q, want %q", cfg.Git.Path, wantGit)
	}
}

func TestLoadWithOptions_BootstrapPrefixWinsOverYAMLRuntimePrefix(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	yamlClaimedRoot := filepath.Join(t.TempDir(), "yaml-prefix-should-lose")
	bootstrapRoot := t.TempDir()
	logical := "/etc/kui/config.yaml"
	physical := filepath.Join(bootstrapRoot, "etc", "kui", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(physical), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configContent := []byte(`runtime:
  prefix: "` + yamlClaimedRoot + `"
hosts:
  - id: local
    uri: qemu:///system
db:
  path: /var/lib/kui/kui.db
jwt_secret: "` + validJWTSecret + `"
`)
	if err := os.WriteFile(physical, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadWithOptions(logical, LoadOptions{Prefix: bootstrapRoot})
	if err != nil {
		t.Fatalf("LoadWithOptions: %v", err)
	}
	wantDB := prefix.Resolve(bootstrapRoot, "/var/lib/kui/kui.db")
	if cfg.DB.Path != wantDB {
		t.Errorf("DB.Path = %q, want bootstrap-normalized %q (not YAML runtime.prefix)", cfg.DB.Path, wantDB)
	}
}

func TestLoadWithOptions_YAMLRuntimePrefixDoesNotAffectPaths(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	runtimeRoot := filepath.Join(t.TempDir(), "yaml-prefix-ignored")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	configContent := []byte(`runtime:
  prefix: "` + runtimeRoot + `"
hosts:
  - id: local
    uri: qemu:///system
db:
  path: /var/lib/kui/kui.db
git:
  path: /var/lib/kui
jwt_secret: "` + validJWTSecret + `"
`)
	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadWithOptions(configPath, LoadOptions{})
	if err != nil {
		t.Fatalf("LoadWithOptions: %v", err)
	}
	// Prefix is flag/env/load-options only; a runtime.prefix key in YAML is not applied.
	if cfg.DB.Path != "/var/lib/kui/kui.db" {
		t.Errorf("DB.Path = %q, want /var/lib/kui/kui.db", cfg.DB.Path)
	}
	if cfg.Git.Path != "/var/lib/kui" {
		t.Errorf("Git.Path = %q, want /var/lib/kui", cfg.Git.Path)
	}
}

func TestLoadWithArgs_BootstrapResolvesConfigPathAndPaths(t *testing.T) {
	t.Cleanup(func() {
		os.Unsetenv("KUI_CONFIG")
		os.Unsetenv("KUI_PREFIX")
	})
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	root := t.TempDir()
	logical := "/etc/kui/config.yaml"
	physical := filepath.Join(root, "etc", "kui", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(physical), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
db:
  path: /var/lib/kui/kui.db
git:
  path: /var/lib/kui
jwt_secret: "` + validJWTSecret + `"
`)
	if err := os.WriteFile(physical, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, resolved, err := LoadWithArgs([]string{"--prefix", root, "--config", logical})
	if err != nil {
		t.Fatalf("LoadWithArgs: %v", err)
	}
	if resolved != physical {
		t.Errorf("resolved = %q, want %q", resolved, physical)
	}
	wantDB := prefix.Resolve(root, "/var/lib/kui/kui.db")
	wantGit := prefix.Resolve(root, "/var/lib/kui")
	if cfg.DB.Path != wantDB {
		t.Errorf("DB.Path = %q, want %q", cfg.DB.Path, wantDB)
	}
	if cfg.Git.Path != wantGit {
		t.Errorf("Git.Path = %q, want %q", cfg.Git.Path, wantGit)
	}
}

func TestLoadWithArgs_KUI_PREFIXNormalizesPaths(t *testing.T) {
	t.Cleanup(func() {
		os.Unsetenv("KUI_CONFIG")
		os.Unsetenv("KUI_PREFIX")
	})
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	root := t.TempDir()
	if err := os.Setenv("KUI_PREFIX", root); err != nil {
		t.Fatalf("setenv KUI_PREFIX: %v", err)
	}

	logical := "/etc/kui/config.yaml"
	physical := filepath.Join(root, "etc", "kui", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(physical), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configContent := []byte(`hosts:
  - id: remote
    uri: qemu+ssh://user@host/system
    keyfile: /root/.ssh/kui_ed25519
db:
  path: /var/lib/kui/kui.db
git:
  path: /var/lib/kui
jwt_secret: "` + validJWTSecret + `"
`)
	if err := os.WriteFile(physical, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, resolved, err := LoadWithArgs([]string{"--config", logical})
	if err != nil {
		t.Fatalf("LoadWithArgs: %v", err)
	}
	if resolved != physical {
		t.Errorf("resolved = %q, want %q", resolved, physical)
	}
	wantDB := prefix.Resolve(root, "/var/lib/kui/kui.db")
	wantGit := prefix.Resolve(root, "/var/lib/kui")
	wantKey := prefix.Resolve(root, "/root/.ssh/kui_ed25519")
	if cfg.DB.Path != wantDB {
		t.Errorf("DB.Path = %q, want %q", cfg.DB.Path, wantDB)
	}
	if cfg.Git.Path != wantGit {
		t.Errorf("Git.Path = %q, want %q", cfg.Git.Path, wantGit)
	}
	if cfg.Hosts[0].Keyfile == nil || *cfg.Hosts[0].Keyfile != wantKey {
		t.Errorf("Keyfile = %v, want %q", cfg.Hosts[0].Keyfile, wantKey)
	}
}

func TestLoadWithOptions_EnvOverridesNormalizedUnderBootstrapPrefix(t *testing.T) {
	t.Cleanup(func() {
		os.Unsetenv("KUI_DB_PATH")
		os.Unsetenv("KUI_GIT_PATH")
	})
	_ = os.Unsetenv("KUI_SECURE_COOKIES")

	root := t.TempDir()
	if err := os.Setenv("KUI_DB_PATH", "/override/db.sqlite"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	if err := os.Setenv("KUI_GIT_PATH", "/override/git"); err != nil {
		t.Fatalf("setenv: %v", err)
	}

	logical := "/etc/kui/config.yaml"
	physical := filepath.Join(root, "etc", "kui", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(physical), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "` + validJWTSecret + `"
`)
	if err := os.WriteFile(physical, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadWithOptions(logical, LoadOptions{Prefix: root})
	if err != nil {
		t.Fatalf("LoadWithOptions: %v", err)
	}
	wantDB := prefix.Resolve(root, "/override/db.sqlite")
	wantGit := prefix.Resolve(root, "/override/git")
	if cfg.DB.Path != wantDB {
		t.Errorf("DB.Path = %q, want %q", cfg.DB.Path, wantDB)
	}
	if cfg.Git.Path != wantGit {
		t.Errorf("Git.Path = %q, want %q", cfg.Git.Path, wantGit)
	}
}

func TestLoadWithArgs_InvalidFlag(t *testing.T) {
	_, _, err := LoadWithArgs([]string{"--unknown-flag"})
	if err == nil {
		t.Fatal("expected flag parse error")
	}
}

func TestHostKeyfileEnvVar_SpecialChars(t *testing.T) {
	t.Parallel()
	// host-id-with-dashes -> KUI_HOST_ID_WITH_DASHES_KEYFILE
	if got := hostKeyfileEnvVar("host-id"); got != "KUI_HOST_HOST_ID_KEYFILE" {
		t.Errorf("hostKeyfileEnvVar(host-id) = %q", got)
	}
}

func TestDuration_String(t *testing.T) {
	t.Parallel()
	d := Duration(time.Hour)
	if d.String() != "1h0m0s" {
		t.Errorf("Duration.String() = %q, want 1h0m0s", d.String())
	}
}
