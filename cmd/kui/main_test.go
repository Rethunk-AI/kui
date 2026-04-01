package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"math/big"
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
	defer func() { _ = resp.Body.Close() }()
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
	defer func() { _ = resp.Body.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

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
	// Do not run in parallel: sets KUI_DB_PATH/KUI_GIT_PATH/KUI_TEST_SETUP_MOCK which would pollute other tests.
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "idempotent.db")
	configPath := filepath.Join(tempDir, "config.yaml")
	t.Cleanup(func() {
		_ = os.Unsetenv("KUI_DB_PATH")
		_ = os.Unsetenv("KUI_GIT_PATH")
		_ = os.Unsetenv("KUI_TEST_SETUP_MOCK")
	})
	_ = os.Setenv("KUI_DB_PATH", dbPath)
	_ = os.Setenv("KUI_GIT_PATH", tempDir)
	_ = os.Setenv("KUI_TEST_SETUP_MOCK", "1")

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
	_ = resp1.Body.Close()
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
	_ = resp2.Body.Close()
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
		_ = os.Unsetenv("KUI_CONFIG")
		_ = os.Unsetenv("KUI_LISTEN")
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

	_ = os.Setenv("KUI_CONFIG", configPath)
	_ = os.Setenv("KUI_LISTEN", "127.0.0.1:0")

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
		_ = os.Unsetenv("KUI_CONFIG")
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
	_ = os.Setenv("KUI_CONFIG", "/other/path")

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
	t.Cleanup(func() { _ = os.Unsetenv("KUI_DB_PATH") })

	// Use a path that is a directory (not a file) - sqlite Open will fail
	tempDir := t.TempDir()
	_ = os.Setenv("KUI_DB_PATH", tempDir)

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

func TestBuildApplication_ConfigPathStatError(t *testing.T) {
	t.Cleanup(func() { _ = os.Unsetenv("KUI_DB_PATH") })

	// Use a path that causes stat to fail with non-ErrNotExist (e.g. permission denied)
	tempDir := t.TempDir()
	restrictedDir := filepath.Join(tempDir, "restricted")
	if err := os.Mkdir(restrictedDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Chmod(restrictedDir, 0o000); err != nil {
		t.Skip("chmod 000 not supported or not effective")
	}
	defer func() { _ = os.Chmod(restrictedDir, 0o700) }()

	cfgPath := filepath.Join(restrictedDir, "config.yaml")
	opts, err := parseFlags([]string{"--config", cfgPath, "--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}

	_, err = buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("expected error when config path stat fails with non-ErrNotExist")
	}
	if !strings.Contains(err.Error(), "read config path") {
		t.Errorf("expected config path error, got %v", err)
	}
}

func TestBuildApplication_GitInitFailure(t *testing.T) {
	t.Cleanup(func() { _ = os.Unsetenv("KUI_DB_PATH") })

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "kui.db")
	configPath := filepath.Join(tempDir, "config.yaml")
	gitPath := filepath.Join(tempDir, "git")
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "0123456789abcdef0123456789abcdef"
db:
  path: ` + dbPath + `
git:
  path: ` + gitPath + `
`)
	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	// Create git path as a file so git.Init fails (cannot create .git in a file)
	if err := os.WriteFile(gitPath, []byte("x"), 0o600); err != nil {
		t.Fatalf("write git path: %v", err)
	}

	opts, err := parseFlags([]string{"--config", configPath, "--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}

	_, err = buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("expected error when git init fails")
	}
	if !strings.Contains(err.Error(), "init git") {
		t.Errorf("expected git init error, got %v", err)
	}
}

func TestShutdown_NilApp(t *testing.T) {
	t.Parallel()
	var app *application
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := app.shutdown(ctx); err != nil {
		t.Errorf("shutdown(nil) should return nil, got %v", err)
	}
}

func TestParseFlags_ValidTLSPair(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	certPath := filepath.Join(tempDir, "cert.pem")
	keyPath := filepath.Join(tempDir, "key.pem")
	if err := os.WriteFile(certPath, []byte("cert"), 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	if err := os.WriteFile(keyPath, []byte("key"), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	opts, err := parseFlags([]string{"--listen", "127.0.0.1:0", "--tls-cert", certPath, "--tls-key", keyPath})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if opts.tlsCert != certPath || opts.tlsKey != keyPath {
		t.Errorf("unexpected tls opts: cert=%q key=%q", opts.tlsCert, opts.tlsKey)
	}
}

func TestStartServer_TLS(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	certPath, keyPath := filepath.Join(tempDir, "cert.pem"), filepath.Join(tempDir, "key.pem")
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	opts, err := parseFlags([]string{"--config", filepath.Join(tempDir, "nonexistent.yaml"), "--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	_ = os.Setenv("KUI_DB_PATH", filepath.Join(tempDir, "kui.db"))
	t.Cleanup(func() { _ = os.Unsetenv("KUI_DB_PATH") })

	app, err := buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("buildApplication: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.shutdown(ctx)
	}()

	listener, err := app.startServer(opts.listen, certPath, keyPath)
	if err != nil {
		t.Fatalf("startServer: %v", err)
	}
	// Verify TLS server is listening (skip cert verification for self-signed)
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	resp, err := client.Get(fmt.Sprintf("https://%s/", listener.Addr().String()))
	if err != nil {
		t.Fatalf("GET https: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestStartServer_TLSUnderPrefix(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	certPath := filepath.Join(root, "etc", "ssl", "cert.pem")
	keyPath := filepath.Join(root, "etc", "ssl", "key.pem")
	if err := os.MkdirAll(filepath.Dir(certPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	if err := os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600); err != nil {
		t.Fatalf("write cert: %v", err)
	}
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshal key: %v", err)
	}
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	opts, err := parseFlags([]string{
		"--prefix", root,
		"--config", filepath.Join(root, "nonexistent.yaml"),
		"--listen", "127.0.0.1:0",
		"--tls-cert", "/etc/ssl/cert.pem",
		"--tls-key", "/etc/ssl/key.pem",
	})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	_ = os.Setenv("KUI_DB_PATH", filepath.Join(root, "var", "lib", "kui", "tls.db"))
	t.Cleanup(func() { _ = os.Unsetenv("KUI_DB_PATH") })
	if err := os.MkdirAll(filepath.Dir(os.Getenv("KUI_DB_PATH")), 0o755); err != nil {
		t.Fatalf("mkdir db parent: %v", err)
	}

	app, err := buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("buildApplication: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.shutdown(ctx)
	}()

	tlsCert, tlsKey := resolveTLSCertKey(opts.bootstrapPrefix, opts.tlsCert, opts.tlsKey)
	if tlsCert != certPath || tlsKey != keyPath {
		t.Fatalf("resolved tls paths cert=%q key=%q, want %q %q", tlsCert, tlsKey, certPath, keyPath)
	}

	listener, err := app.startServer(opts.listen, tlsCert, tlsKey)
	if err != nil {
		t.Fatalf("startServer: %v", err)
	}
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	resp, err := client.Get(fmt.Sprintf("https://%s/", listener.Addr().String()))
	if err != nil {
		t.Fatalf("GET https: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestParseFlags_PrefixFromEnvWhenFlagUnset(t *testing.T) {
	t.Cleanup(func() { _ = os.Unsetenv("KUI_PREFIX") })
	if err := os.Setenv("KUI_PREFIX", "/from/env"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	opts, err := parseFlags([]string{"--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if opts.bootstrapPrefix != "/from/env" {
		t.Fatalf("bootstrapPrefix = %q, want /from/env", opts.bootstrapPrefix)
	}
}

func TestParseFlags_PrefixFlagOverridesEnv(t *testing.T) {
	t.Cleanup(func() { _ = os.Unsetenv("KUI_PREFIX") })
	if err := os.Setenv("KUI_PREFIX", "/from/env"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	opts, err := parseFlags([]string{"--prefix", "/from/flag", "--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if opts.bootstrapPrefix != "/from/flag" {
		t.Fatalf("bootstrapPrefix = %q, want /from/flag", opts.bootstrapPrefix)
	}
}

func TestResolveTLSCertKey_NoPrefix(t *testing.T) {
	t.Parallel()
	c, k := resolveTLSCertKey("", "/etc/ssl/a.pem", "/etc/ssl/b.pem")
	if c != "/etc/ssl/a.pem" || k != "/etc/ssl/b.pem" {
		t.Fatalf("got cert=%q key=%q", c, k)
	}
}

func TestResolveTLSCertKey_UnderPrefix(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	c, k := resolveTLSCertKey(root, "/etc/ssl/a.pem", "/etc/ssl/b.pem")
	wantC := filepath.Join(root, "etc", "ssl", "a.pem")
	wantK := filepath.Join(root, "etc", "ssl", "b.pem")
	if c != wantC || k != wantK {
		t.Fatalf("got cert=%q key=%q, want %q %q", c, k, wantC, wantK)
	}
}

func TestResolveTLSCertKey_UnderPrefixEmptyCertUnchanged(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	c, k := resolveTLSCertKey(root, "", "")
	if c != "" || k != "" {
		t.Fatalf("got cert=%q key=%q, want empty", c, k)
	}
}

func TestBuildApplication_InvalidPrefixNotDirectory(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	notDir := filepath.Join(tempDir, "file")
	if err := os.WriteFile(notDir, []byte("x"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	opts, err := parseFlags([]string{"--prefix", notDir, "--config", filepath.Join(tempDir, "n.yaml"), "--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	_, err = buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil {
		t.Fatal("expected error for prefix that is not a directory")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected not-a-directory error, got %v", err)
	}
}

func TestBuildApplication_BootstrapPrefixResolvesSetupDBPath(t *testing.T) {
	t.Cleanup(func() { _ = os.Unsetenv("KUI_DB_PATH") })
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "var", "lib", "kui"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Setenv("KUI_DB_PATH", "/var/lib/kui/chroot.db"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	opts, err := parseFlags([]string{
		"--prefix", root,
		"--config", filepath.Join(root, "no-such-config.yaml"),
		"--listen", "127.0.0.1:0",
	})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	app, err := buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("buildApplication: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.shutdown(ctx)
	}()
	wantDB := filepath.Join(root, "var", "lib", "kui", "chroot.db")
	if app.dbPath != wantDB {
		t.Fatalf("dbPath = %q, want %q", app.dbPath, wantDB)
	}
	if _, err := os.Stat(wantDB); err != nil {
		t.Fatalf("expected db file at resolved path: %v", err)
	}
}

func TestBuildApplication_BootstrapPrefixResolvesLogicalConfigPath(t *testing.T) {
	root := t.TempDir()
	resolvedCfg := filepath.Join(root, "etc", "kui", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(resolvedCfg), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dbResolved := filepath.Join(root, "var", "lib", "kui", "pfx.db")
	gitResolved := filepath.Join(root, "var", "lib", "kui", "git")
	if err := os.MkdirAll(filepath.Dir(dbResolved), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(gitResolved, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "0123456789abcdef0123456789abcdef"
db:
  path: /var/lib/kui/pfx.db
git:
  path: /var/lib/kui/git
`)
	if err := os.WriteFile(resolvedCfg, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	opts, err := parseFlags([]string{"--prefix", root, "--config", "/etc/kui/config.yaml", "--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	app, err := buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("buildApplication: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.shutdown(ctx)
	}()
	if app.configPath != resolvedCfg {
		t.Fatalf("configPath = %q, want resolved %q", app.configPath, resolvedCfg)
	}
	if app.config == nil {
		t.Fatal("expected config loaded")
	}
	if app.config.DB.Path != dbResolved {
		t.Fatalf("config db path = %q, want %q", app.config.DB.Path, dbResolved)
	}
}

func TestBuildApplication_KUI_PREFIXEnvResolvesLogicalConfigPath(t *testing.T) {
	t.Cleanup(func() {
		_ = os.Unsetenv("KUI_PREFIX")
		_ = os.Unsetenv("KUI_CONFIG")
	})
	root := t.TempDir()
	resolvedCfg := filepath.Join(root, "etc", "kui", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(resolvedCfg), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dbResolved := filepath.Join(root, "var", "lib", "kui", "envpfx.db")
	gitResolved := filepath.Join(root, "var", "lib", "kui", "git")
	if err := os.MkdirAll(filepath.Dir(dbResolved), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(gitResolved, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configContent := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "0123456789abcdef0123456789abcdef"
db:
  path: /var/lib/kui/envpfx.db
git:
  path: /var/lib/kui/git
`)
	if err := os.WriteFile(resolvedCfg, configContent, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.Setenv("KUI_PREFIX", root); err != nil {
		t.Fatalf("setenv KUI_PREFIX: %v", err)
	}
	if err := os.Setenv("KUI_CONFIG", "/etc/kui/config.yaml"); err != nil {
		t.Fatalf("setenv KUI_CONFIG: %v", err)
	}
	opts, err := parseFlags([]string{"--listen", "127.0.0.1:0"})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if opts.configSource != "env" {
		t.Fatalf("configSource = %q, want env", opts.configSource)
	}
	app, err := buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("buildApplication: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.shutdown(ctx)
	}()
	if app.configPath != resolvedCfg {
		t.Fatalf("configPath = %q, want resolved %q", app.configPath, resolvedCfg)
	}
	if app.config == nil {
		t.Fatal("expected config loaded")
	}
	if app.config.DB.Path != dbResolved {
		t.Fatalf("config db path = %q, want %q", app.config.DB.Path, dbResolved)
	}
}

func TestEmbeddedWeb_KUI_WEB_DIRUnset(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() { _ = os.Unsetenv("KUI_WEB_DIR") })
	_ = os.Unsetenv("KUI_WEB_DIR")

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "emb.db")
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
		t.Fatalf("write config: %v", err)
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
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestBuildApplication_KUI_WEB_DIRResolvedUnderPrefix(t *testing.T) {
	t.Cleanup(func() { _ = os.Unsetenv("KUI_WEB_DIR"); _ = os.Unsetenv("KUI_DB_PATH") })
	root := t.TempDir()
	webResolved := filepath.Join(root, "opt", "kui", "static")
	if err := os.MkdirAll(webResolved, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(webResolved, "index.html"), []byte("<!doctype html><html></html>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.Setenv("KUI_WEB_DIR", "/opt/kui/static"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "var", "lib", "kui"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.Setenv("KUI_DB_PATH", "/var/lib/kui/webtest.db"); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	opts, err := parseFlags([]string{
		"--prefix", root,
		"--config", filepath.Join(root, "missing.yaml"),
		"--listen", "127.0.0.1:0",
	})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	app, err := buildApplication(opts, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("buildApplication: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.shutdown(ctx)
	}()
	listener, err := app.startServer(opts.listen, "", "")
	if err != nil {
		t.Fatalf("startServer: %v", err)
	}
	resp, err := http.Get(fmt.Sprintf("http://%s/", listener.Addr().String()))
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}
