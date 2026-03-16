package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/kui/kui/internal/config"
	"github.com/kui/kui/internal/db"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

func TestSetupStatus_ConfigMissing(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	database, err := db.Open(filepath.Join(tempDir, "kui.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	handler := NewRouter(RouterOptions{
		Logger:        nil,
		DB:            database,
		Config:        nil,
		ConfigPath:    filepath.Join(tempDir, "nonexistent.yaml"),
		ConfigPresent: false,
		DBPath:        filepath.Join(tempDir, "kui.db"),
		GitPath:       tempDir,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/setup/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body struct {
		SetupRequired bool    `json:"setup_required"`
		Reason        *string `json:"reason"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.SetupRequired || body.Reason == nil || *body.Reason != "config_missing" {
		t.Fatalf("expected setup_required=true, reason=config_missing, got %+v", body)
	}
}

func TestSetupStatus_NoAdmin(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	cfgYAML := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "` + testJWTSecret + `"
`)
	if err := os.WriteFile(configPath, cfgYAML, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	database, err := db.Open(filepath.Join(tempDir, "kui.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	handler := NewRouter(RouterOptions{
		Logger:        nil,
		DB:            database,
		Config:        loaded,
		ConfigPath:    configPath,
		ConfigPresent: true,
		DBPath:        filepath.Join(tempDir, "kui.db"),
		GitPath:       tempDir,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/setup/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body struct {
		SetupRequired bool    `json:"setup_required"`
		Reason        *string `json:"reason"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.SetupRequired || body.Reason == nil || *body.Reason != "no_admin" {
		t.Fatalf("expected setup_required=true, reason=no_admin, got %+v", body)
	}
}

func TestSetupComplete_AndLogin(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	// Config path must not exist for setup mode
	_ = configPath

	database, err := db.Open(filepath.Join(tempDir, "kui.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	handler := NewRouter(RouterOptions{
		Logger:        nil,
		DB:            database,
		Config:        nil,
		ConfigPath:    configPath,
		ConfigPresent: false,
		DBPath:        filepath.Join(tempDir, "kui.db"),
		GitPath:       tempDir,
	})

	payload := map[string]any{
		"admin": map[string]string{"username": "admin", "password": "secret123"},
		"hosts": []map[string]string{{"id": "local", "uri": "qemu:///system", "keyfile": ""}},
		"default_host": "local",
	}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/setup/complete", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Second call should return 409
	req2 := httptest.NewRequest(http.MethodPost, "/api/setup/complete", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusConflict {
		t.Fatalf("expected 409 on duplicate setup, got %d", rec2.Code)
	}
}

func TestLogin_RequiresConfig(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	database, err := db.Open(filepath.Join(tempDir, "kui.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	handler := NewRouter(RouterOptions{
		Logger:        nil,
		DB:            database,
		Config:        nil,
		ConfigPath:    filepath.Join(tempDir, "config.yaml"),
		ConfigPresent: false,
		DBPath:        filepath.Join(tempDir, "kui.db"),
		GitPath:       tempDir,
	})

	payload := map[string]string{"username": "admin", "password": "secret"}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without jwt_secret, got %d", rec.Code)
	}
}

func TestLogin_AndMe_Preferences_Hosts(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	cfgYAML := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "` + testJWTSecret + `"
`)
	if err := os.WriteFile(configPath, cfgYAML, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	database, err := db.Open(filepath.Join(tempDir, "kui.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_, err = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	handler := NewRouter(RouterOptions{
		Logger:        nil,
		DB:            database,
		Config:        loaded,
		ConfigPath:    configPath,
		ConfigPresent: true,
		DBPath:        filepath.Join(tempDir, "kui.db"),
		GitPath:       tempDir,
	})

	// Login
	loginPayload := map[string]string{"username": "admin", "password": "secret"}
	loginBody, _ := json.Marshal(loginPayload)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login expected 200, got %d: %s", loginRec.Code, loginRec.Body.String())
	}
	var loginResp struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(loginRec.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if loginResp.Token == "" {
		t.Fatal("expected token in response")
	}

	// Me
	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	meRec := httptest.NewRecorder()
	handler.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusOK {
		t.Fatalf("me expected 200, got %d", meRec.Code)
	}
	var meResp struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(meRec.Body).Decode(&meResp); err != nil {
		t.Fatalf("decode me: %v", err)
	}
	if meResp.Username != "admin" || meResp.Role != "admin" {
		t.Fatalf("unexpected me: %+v", meResp)
	}

	// Preferences GET
	prefReq := httptest.NewRequest(http.MethodGet, "/api/preferences", nil)
	prefReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	prefRec := httptest.NewRecorder()
	handler.ServeHTTP(prefRec, prefReq)
	if prefRec.Code != http.StatusOK {
		t.Fatalf("preferences get expected 200, got %d", prefRec.Code)
	}

	// Hosts GET
	hostsReq := httptest.NewRequest(http.MethodGet, "/api/hosts", nil)
	hostsReq.Header.Set("Authorization", "Bearer "+loginResp.Token)
	hostsRec := httptest.NewRecorder()
	handler.ServeHTTP(hostsRec, hostsReq)
	if hostsRec.Code != http.StatusOK {
		t.Fatalf("hosts expected 200, got %d", hostsRec.Code)
	}
	var hostsResp []struct {
		ID  string `json:"id"`
		URI string `json:"uri"`
	}
	if err := json.NewDecoder(hostsRec.Body).Decode(&hostsResp); err != nil {
		t.Fatalf("decode hosts: %v", err)
	}
	if len(hostsResp) != 1 || hostsResp[0].ID != "local" || hostsResp[0].URI != "qemu:///system" {
		t.Fatalf("unexpected hosts: %+v", hostsResp)
	}
}

func TestMe_Unauthorized(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	cfgYAML := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "` + testJWTSecret + `"
`)
	if err := os.WriteFile(configPath, cfgYAML, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	database, err := db.Open(filepath.Join(tempDir, "kui.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	handler := NewRouter(RouterOptions{
		Logger:        nil,
		DB:            database,
		Config:        loaded,
		ConfigPath:    configPath,
		ConfigPresent: true,
		DBPath:        filepath.Join(tempDir, "kui.db"),
		GitPath:       tempDir,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}

func TestValidateHost_SetupOnly(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	cfgYAML := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "` + testJWTSecret + `"
`)
	if err := os.WriteFile(configPath, cfgYAML, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	database, err := db.Open(filepath.Join(tempDir, "kui.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	handler := NewRouter(RouterOptions{
		Logger:        nil,
		DB:            database,
		Config:        loaded,
		ConfigPath:    configPath,
		ConfigPresent: true,
		DBPath:        filepath.Join(tempDir, "kui.db"),
		GitPath:       tempDir,
	})

	payload := map[string]string{"host_id": "local", "uri": "qemu:///system", "keyfile": ""}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/setup/validate-host", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when config present, got %d", rec.Code)
	}
}

func TestCORS_Headers(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	cfgYAML := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "` + testJWTSecret + `"
cors:
  allowed_origins: ["http://localhost:5173"]
`)
	if err := os.WriteFile(configPath, cfgYAML, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	database, err := db.Open(filepath.Join(tempDir, "kui.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	handler := NewRouter(RouterOptions{
		Logger:        nil,
		DB:            database,
		Config:        loaded,
		ConfigPath:    configPath,
		ConfigPresent: true,
		DBPath:        filepath.Join(tempDir, "kui.db"),
		GitPath:       tempDir,
	})

	req := httptest.NewRequest(http.MethodGet, "/api/setup/status", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("expected CORS header, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestDiscoveryEndpoints_RequireAuth(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	database, err := db.Open(filepath.Join(tempDir, "kui.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer database.Close()

	handler := NewRouter(RouterOptions{
		Logger:        nil,
		DB:            database,
		Config:        nil,
		ConfigPath:    filepath.Join(tempDir, "config.yaml"),
		ConfigPresent: false,
		DBPath:        filepath.Join(tempDir, "kui.db"),
		GitPath:       tempDir,
	})

	tests := []struct {
		path string
	}{
		{"/api/vms"},
		{"/api/hosts/local/pools"},
		{"/api/hosts/local/pools/default/volumes"},
		{"/api/hosts/local/networks"},
		{"/api/hosts/local/vms/00000000-0000-0000-0000-000000000000"},
		{"/api/templates"},
	}
	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("GET %s: expected 401 without auth, got %d", tt.path, rec.Code)
		}
	}
}
