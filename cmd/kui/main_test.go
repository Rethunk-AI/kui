package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMainStartup(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "startup.db")
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "0123456789abcdef0123456789abcdef"
db:
  path: ` + dbPath + `
git:
  path: ` + tempDir + `
`)
	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	opts, err := parseFlags([]string{"--config", configPath, "--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	app, err := buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("build application: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.shutdown(ctx)
	}()

	listener, err := app.startServer(opts.listen, "", "")
	if err != nil {
		t.Fatalf("start server: %v", err)
	}

	resp, err := http.Get(fmt.Sprintf("http://%s/", listener.Addr().String()))
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}
}

func TestSetupMode(t *testing.T) {
	// Do not run in parallel: sets KUI_DB_PATH which would pollute other tests.
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "setup.db")
	configPath := filepath.Join(tempDir, "config.yaml")
	t.Cleanup(func() {
		_ = os.Unsetenv("KUI_DB_PATH")
	})
	if err := os.Setenv("KUI_DB_PATH", dbPath); err != nil {
		t.Fatalf("set KUI_DB_PATH: %v", err)
	}

	opts, err := parseFlags([]string{"--config", configPath, "--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	app, err := buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("build application: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.shutdown(ctx)
	}()

	listener, err := app.startServer(opts.listen, "", "")
	if err != nil {
		t.Fatalf("start server: %v", err)
	}
	if app.configExists {
		t.Fatalf("expected setup mode configuration")
	}

	resp, err := http.Get(fmt.Sprintf("http://%s/api/setup/status", listener.Addr().String()))
	if err != nil {
		t.Fatalf("GET /api/setup/status failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	var payload struct {
		SetupRequired bool    `json:"setup_required"`
		Reason        *string `json:"reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.SetupRequired {
		t.Fatalf("expected setup_required=true")
	}
	if payload.Reason == nil || *payload.Reason != "config_missing" {
		t.Fatalf("expected reason config_missing, got %#v", payload.Reason)
	}
}

func TestInvalidConfigFallsBackToSetupMode(t *testing.T) {
	// Do not run in parallel: sets KUI_DB_PATH which would pollute other tests.
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "invalid_config.db")
	configPath := filepath.Join(tempDir, "config.yaml")
	t.Cleanup(func() {
		_ = os.Unsetenv("KUI_DB_PATH")
	})
	if err := os.Setenv("KUI_DB_PATH", dbPath); err != nil {
		t.Fatalf("set KUI_DB_PATH: %v", err)
	}

	// Invalid config: empty hosts (validation error)
	invalidConfig := []byte(`hosts: []
jwt_secret: "0123456789abcdef0123456789abcdef"
`)
	if err := os.WriteFile(configPath, invalidConfig, 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	opts, err := parseFlags([]string{"--config", configPath, "--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	app, err := buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("build application: %v (invalid config should fall back to setup mode, not crash)", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.shutdown(ctx)
	}()

	if app.config != nil {
		t.Fatalf("expected app.config=nil when config invalid")
	}

	listener, err := app.startServer(opts.listen, "", "")
	if err != nil {
		t.Fatalf("start server: %v", err)
	}

	resp, err := http.Get(fmt.Sprintf("http://%s/api/setup/status", listener.Addr().String()))
	if err != nil {
		t.Fatalf("GET /api/setup/status failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	var payload struct {
		SetupRequired bool    `json:"setup_required"`
		Reason        *string `json:"reason"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.SetupRequired {
		t.Fatalf("expected setup_required=true when config invalid")
	}
	if payload.Reason == nil || *payload.Reason != "config_missing" {
		t.Fatalf("expected reason config_missing, got %#v", payload.Reason)
	}
}

func TestSetupCompleteIdempotent(t *testing.T) {
	// Do not run in parallel: sets KUI_DB_PATH/KUI_GIT_PATH which would pollute other tests.
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "idempotent.db")
	configPath := filepath.Join(tempDir, "config.yaml")
	t.Cleanup(func() {
		_ = os.Unsetenv("KUI_DB_PATH")
		_ = os.Unsetenv("KUI_GIT_PATH")
	})
	_ = os.Setenv("KUI_DB_PATH", dbPath)
	_ = os.Setenv("KUI_GIT_PATH", tempDir)

	opts, err := parseFlags([]string{"--config", configPath, "--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	app, err := buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("build application: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.shutdown(ctx)
	}()

	listener, err := app.startServer(opts.listen, "", "")
	if err != nil {
		t.Fatalf("start server: %v", err)
	}

	body := []byte(`{"admin":{"username":"admin","password":"secret"},"hosts":[{"id":"local","uri":"qemu:///system","keyfile":null}],"default_host":"local"}`)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/api/setup/complete", listener.Addr().String()), nil)
	req.Body = io.NopCloser(strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")

	resp1, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("first setup/complete: %v", err)
	}
	resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("first setup/complete: expected 201, got %d", resp1.StatusCode)
	}

	req2, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/api/setup/complete", listener.Addr().String()), nil)
	req2.Body = io.NopCloser(strings.NewReader(string(body)))
	req2.Header.Set("Content-Type", "application/json")

	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		t.Fatalf("second setup/complete: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusConflict {
		t.Fatalf("second setup/complete: expected 409 Conflict, got %d", resp2.StatusCode)
	}
}

func TestShutdown(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "shutdown.db")
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "0123456789abcdef0123456789abcdef"
db:
  path: ` + dbPath + `
git:
  path: ` + tempDir + `
`)
	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	opts, err := parseFlags([]string{"--config", configPath, "--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	app, err := buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("build application: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.shutdown(ctx)
	}()

	listener, err := app.startServer(opts.listen, "", "")
	if err != nil {
		t.Fatalf("start server: %v", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	if err := app.shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if listener == nil {
		t.Fatalf("expected listener to exist")
	}
}

func TestParseFlags_InvalidFlag(t *testing.T) {
	t.Parallel()
	_, err := parseFlags([]string{"--listen", "127.0.0.1:0", "--unknown-flag"})
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
}

func TestParseFlags_EmptyListen(t *testing.T) {
	t.Parallel()
	_ = os.Unsetenv("KUI_LISTEN")

	_, err := parseFlags([]string{"--listen", ""})
	if err == nil {
		t.Fatal("expected error for empty listen")
	}
	if !strings.Contains(err.Error(), "listen address") {
		t.Errorf("expected listen address error, got %v", err)
	}
}

func TestParseFlags_TLSPairRequired(t *testing.T) {
	t.Parallel()

	_, err := parseFlags([]string{"--listen", "127.0.0.1:0", "--tls-cert", "/cert.pem"})
	if err == nil {
		t.Fatal("expected error for tls-cert without tls-key")
	}
	if !strings.Contains(err.Error(), "tls-cert") {
		t.Errorf("expected tls pair error, got %v", err)
	}

	_, err = parseFlags([]string{"--listen", "127.0.0.1:0", "--tls-key", "/key.pem"})
	if err == nil {
		t.Fatal("expected error for tls-key without tls-cert")
	}
}

func TestParseFlags_EnvOverrides(t *testing.T) {
	t.Cleanup(func() {
		os.Unsetenv("KUI_CONFIG")
		os.Unsetenv("KUI_LISTEN")
	})

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "0123456789abcdef0123456789abcdef"
`)
	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	os.Setenv("KUI_CONFIG", configPath)
	os.Setenv("KUI_LISTEN", "127.0.0.1:0")

	opts, err := parseFlags([]string{})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if opts.configPath != configPath {
		t.Errorf("configPath = %q, want %q", opts.configPath, configPath)
	}
	if opts.listen != "127.0.0.1:0" {
		t.Errorf("listen = %q, want 127.0.0.1:0", opts.listen)
	}
	if opts.configSource != "env" {
		t.Errorf("configSource = %q, want env", opts.configSource)
	}
}

func TestParseFlags_ConfigFlagOverridesEnv(t *testing.T) {
	t.Cleanup(func() {
		os.Unsetenv("KUI_CONFIG")
	})

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "0123456789abcdef0123456789abcdef"
`)
	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	os.Setenv("KUI_CONFIG", "/other/path")

	opts, err := parseFlags([]string{"--config", configPath, "--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if opts.configPath != configPath {
		t.Errorf("configPath = %q, want flag value %q", opts.configPath, configPath)
	}
	if opts.configSource != "--config" {
		t.Errorf("configSource = %q, want --config", opts.configSource)
	}
}

func TestBuildApplication_DBOpenFailureInSetupMode(t *testing.T) {
	t.Cleanup(func() { os.Unsetenv("KUI_DB_PATH") })

	// Use a path that is a directory (not a file) - sqlite Open will fail
	tempDir := t.TempDir()
	os.Setenv("KUI_DB_PATH", tempDir)

	opts, err := parseFlags([]string{"--config", filepath.Join(tempDir, "nonexistent.yaml"), "--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}

	_, err = buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("expected error when db open fails in setup mode")
	}
	if !strings.Contains(err.Error(), "open db") && !strings.Contains(err.Error(), "database") {
		t.Errorf("expected db open error, got %v", err)
	}
}

func TestCloseDatabase_Nil(t *testing.T) {
	t.Parallel()
	if err := closeDatabase(nil); err != nil {
		t.Errorf("closeDatabase(nil) should return nil, got %v", err)
	}
}

func TestGetFileStatError(t *testing.T) {
	t.Parallel()
	if err := getFileStatError("/nonexistent/path/xyz"); err == nil {
		t.Error("getFileStatError(nonexistent) should return error")
	}
	tempDir := t.TempDir()
	if err := getFileStatError(tempDir); err != nil {
		t.Errorf("getFileStatError(existing) = %v", err)
	}
}
