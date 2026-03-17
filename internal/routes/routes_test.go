package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/kui/kui/internal/config"
	"github.com/kui/kui/internal/db"
	"github.com/kui/kui/internal/libvirtconn"
)

const testJWTSecret = "0123456789abcdef0123456789abcdef"

// mockConnector implements libvirtconn.Connector for handler tests.
type mockConnector struct {
	listDomainsErr   error
	domains          []libvirtconn.DomainInfo
	lookupErr        error
	domainInfo       libvirtconn.DomainInfo
	listPoolsErr     error
	pools            []libvirtconn.StoragePoolInfo
	listVolumesErr   error
	volumes          []libvirtconn.StorageVolumeInfo
	listNetworksErr  error
	networks         []libvirtconn.NetworkInfo
	getStateErr      error
	state            libvirtconn.DomainLifecycleState
	getDomainXMLErr  error
	domainXML        string
	validatePoolErr  error
	validateVolErr   error
	createVolErr     error
	volInfo          libvirtconn.StorageVolumeInfo
	defineXMLErr     error
	createErr        error
	shutdownErr      error
	destroyErr       error
	undefineErr      error
	suspendErr       error
	resumeErr        error
	cloneVolErr      error
	copyVolErr       error
	createVolBytesErr error
}

func (m *mockConnector) Close() error { return nil }

func (m *mockConnector) ListDomains(ctx context.Context) ([]libvirtconn.DomainInfo, error) {
	if m.listDomainsErr != nil {
		return nil, m.listDomainsErr
	}
	return m.domains, nil
}

func (m *mockConnector) LookupByUUID(ctx context.Context, uuid string) (libvirtconn.DomainInfo, error) {
	if m.lookupErr != nil {
		return libvirtconn.DomainInfo{}, m.lookupErr
	}
	return m.domainInfo, nil
}

func (m *mockConnector) GetDomainXML(ctx context.Context, uuid string) (string, error) {
	if m.getDomainXMLErr != nil {
		return "", m.getDomainXMLErr
	}
	return m.domainXML, nil
}

func (m *mockConnector) DefineXML(ctx context.Context, xmlConfig string) (libvirtconn.DomainInfo, error) {
	if m.defineXMLErr != nil {
		return libvirtconn.DomainInfo{}, m.defineXMLErr
	}
	return m.domainInfo, nil
}

func (m *mockConnector) Create(ctx context.Context, uuid string) error   { return m.createErr }
func (m *mockConnector) Shutdown(ctx context.Context, uuid string) error { return m.shutdownErr }
func (m *mockConnector) Destroy(ctx context.Context, uuid string) error { return m.destroyErr }
func (m *mockConnector) Undefine(ctx context.Context, uuid string) error { return m.undefineErr }
func (m *mockConnector) Suspend(ctx context.Context, uuid string) error { return m.suspendErr }
func (m *mockConnector) Resume(ctx context.Context, uuid string) error  { return m.resumeErr }

func (m *mockConnector) GetState(ctx context.Context, uuid string) (libvirtconn.DomainLifecycleState, error) {
	if m.getStateErr != nil {
		return "", m.getStateErr
	}
	return m.state, nil
}

func (m *mockConnector) ListNetworks(ctx context.Context) ([]libvirtconn.NetworkInfo, error) {
	if m.listNetworksErr != nil {
		return nil, m.listNetworksErr
	}
	return m.networks, nil
}

func (m *mockConnector) ListPools(ctx context.Context) ([]libvirtconn.StoragePoolInfo, error) {
	if m.listPoolsErr != nil {
		return nil, m.listPoolsErr
	}
	return m.pools, nil
}

func (m *mockConnector) ListVolumes(ctx context.Context, pool string) ([]libvirtconn.StorageVolumeInfo, error) {
	if m.listVolumesErr != nil {
		return nil, m.listVolumesErr
	}
	return m.volumes, nil
}

func (m *mockConnector) ValidatePool(ctx context.Context, pool string) error   { return m.validatePoolErr }
func (m *mockConnector) ValidatePath(ctx context.Context, pool, path string) error { return nil }
func (m *mockConnector) ValidateVolume(ctx context.Context, pool, name string) error {
	return m.validateVolErr
}

func (m *mockConnector) CreateVolumeFromXML(ctx context.Context, pool, xml string) (libvirtconn.StorageVolumeInfo, error) {
	if m.createVolErr != nil {
		return libvirtconn.StorageVolumeInfo{}, m.createVolErr
	}
	return m.volInfo, nil
}

func (m *mockConnector) CloneVolume(ctx context.Context, pool, sourceName, targetName string) error {
	return m.cloneVolErr
}

func (m *mockConnector) CopyVolume(ctx context.Context, pool, volumeName string) ([]byte, error) {
	if m.copyVolErr != nil {
		return nil, m.copyVolErr
	}
	return []byte{}, nil
}

func (m *mockConnector) CreateVolumeFromBytes(ctx context.Context, pool, name string, data []byte, format string) (libvirtconn.StorageVolumeInfo, error) {
	if m.createVolBytesErr != nil {
		return libvirtconn.StorageVolumeInfo{}, m.createVolBytesErr
	}
	return m.volInfo, nil
}

func (m *mockConnector) OpenSerialConsole(ctx context.Context, uuid string) (io.ReadWriteCloser, error) {
	return nil, errors.New("not implemented")
}

// flushRecorder wraps ResponseRecorder to implement http.Flusher for SSE tests.
type flushRecorder struct {
	*httptest.ResponseRecorder
}

func (f *flushRecorder) Flush() {}

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

func TestSetupStatus_DBMissing(t *testing.T) {
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

	handler := NewRouter(RouterOptions{
		Logger:        nil,
		DB:            nil,
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
	if !body.SetupRequired || body.Reason == nil || *body.Reason != "db_missing" {
		t.Fatalf("expected setup_required=true, reason=db_missing, got %+v", body)
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

func TestSanitizeValidationError(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"passes through", "connection refused", "connection refused"},
		{"redacts path", "Failed to open /home/user/.ssh/id_rsa: permission denied", "Failed to open [path redacted]: permission denied"},
		{"redacts path", "error: /var/run/libvirt/libvirt-sock: No such file", "error: [path redacted]: No such file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeValidationError(tt.in)
			if got != tt.want {
				t.Errorf("sanitizeValidationError(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
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
		{"/api/hosts/local/vms/00000000-0000-0000-0000-000000000000/vnc"},
		{"/api/hosts/local/vms/00000000-0000-0000-0000-000000000000/serial"},
		{"/api/templates"},
		{"/api/events"},
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

func TestEvents_SSEStream(t *testing.T) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	// ResponseRecorder does not implement http.Flusher; wrap it for SSE tests.
	flusher := &flushRecorder{ResponseRecorder: rec}
	handler.ServeHTTP(flusher, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type: text/event-stream, got %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: host.online") {
		t.Errorf("expected SSE event host.online in body, got: %q", body)
	}
	if !strings.Contains(body, `data: {"host_id":"kui"}`) {
		t.Errorf("expected SSE data with host_id kui in body, got: %q", body)
	}
}

func TestVNC_HostNotFound(t *testing.T) {
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
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	handler := NewRouter(RouterOptions{
		Logger: nil, DB: database, Config: loaded, ConfigPath: configPath,
		ConfigPresent: true, DBPath: filepath.Join(tempDir, "kui.db"), GitPath: tempDir,
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", loginRec.Code, loginRec.Body.String())
	}
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)

	req := httptest.NewRequest(http.MethodGet, "/api/hosts/nonexistent/vms/00000000-0000-0000-0000-000000000000/vnc", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 host not found, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestVNC_VMNotFound(t *testing.T) {
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
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	handler := NewRouter(RouterOptions{
		Logger: nil, DB: database, Config: loaded, ConfigPath: configPath,
		ConfigPresent: true, DBPath: filepath.Join(tempDir, "kui.db"), GitPath: tempDir,
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", loginRec.Code, loginRec.Body.String())
	}
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)

	req := httptest.NewRequest(http.MethodGet, "/api/hosts/local/vms/00000000-0000-0000-0000-000000000000/vnc", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 VM not found, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSerial_HostNotFound(t *testing.T) {
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
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	handler := NewRouter(RouterOptions{
		Logger: nil, DB: database, Config: loaded, ConfigPath: configPath,
		ConfigPresent: true, DBPath: filepath.Join(tempDir, "kui.db"), GitPath: tempDir,
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", loginRec.Code, loginRec.Body.String())
	}
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)

	req := httptest.NewRequest(http.MethodGet, "/api/hosts/nonexistent/vms/00000000-0000-0000-0000-000000000000/serial", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 host not found, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSerial_VMNotFound(t *testing.T) {
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
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	handler := NewRouter(RouterOptions{
		Logger: nil, DB: database, Config: loaded, ConfigPath: configPath,
		ConfigPresent: true, DBPath: filepath.Join(tempDir, "kui.db"), GitPath: tempDir,
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", loginRec.Code, loginRec.Body.String())
	}
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)

	req := httptest.NewRequest(http.MethodGet, "/api/hosts/local/vms/00000000-0000-0000-0000-000000000000/serial", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 VM not found, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestVNCPortFromDomainXML(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		xml     string
		wantPort int
	}{
		{"no devices", `<domain><name>vm</name></domain>`, 0},
		{"no graphics", `<domain><devices></devices></domain>`, 0},
		{"vnc with port", `<domain><devices><graphics type="vnc" port="5900"/></devices></domain>`, 5900},
		{"vnc autoport no port", `<domain><devices><graphics type="vnc" autoport="yes"/></devices></domain>`, 0},
		{"spice ignored", `<domain><devices><graphics type="spice" port="5901"/></devices></domain>`, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			port, err := vncPortFromDomainXML(tt.xml)
			if err != nil {
				t.Fatalf("vncPortFromDomainXML: %v", err)
			}
			if port != tt.wantPort {
				t.Errorf("got port %d, want %d", port, tt.wantPort)
			}
		})
	}
}

func TestStaticHandler_NilFS(t *testing.T) {
	t.Parallel()
	handler := staticHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "web assets not available") {
		t.Errorf("expected 'web assets not available' in body, got %q", body)
	}
}

func TestExtractFirstDiskPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		xml  string
		want string
	}{
		{"no disk", `<domain><name>vm</name></domain>`, ""},
		{"disk with file", `<domain><devices><disk><source file="/var/lib/libvirt/images/disk.qcow2"/></disk></devices></domain>`, "/var/lib/libvirt/images/disk.qcow2"},
		{"disk with single quotes", `<domain><devices><disk><source file='/path/to/disk.qcow2'/></disk></devices></domain>`, "/path/to/disk.qcow2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFirstDiskPath(tt.xml)
			if got != tt.want {
				t.Errorf("extractFirstDiskPath = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeHosts(t *testing.T) {
	t.Parallel()
	valid := []struct {
		ID      string `json:"id"`
		URI     string `json:"uri"`
		Keyfile string `json:"keyfile"`
	}{
		{ID: "a", URI: "qemu:///system", Keyfile: ""},
		{ID: "b", URI: "qemu+ssh://host/system", Keyfile: "/key"},
	}
	hosts, ok := normalizeHosts(valid)
	if !ok || len(hosts) != 2 {
		t.Fatalf("normalizeHosts(valid) = %v, %v; want ok=true, len=2", hosts, ok)
	}
	if hosts[0].ID != "a" || hosts[1].ID != "b" {
		t.Errorf("unexpected hosts: %+v", hosts)
	}

	emptyID := []struct {
		ID      string `json:"id"`
		URI     string `json:"uri"`
		Keyfile string `json:"keyfile"`
	}{{ID: "", URI: "qemu:///system", Keyfile: ""}}
	if _, ok := normalizeHosts(emptyID); ok {
		t.Error("normalizeHosts(empty id) should fail")
	}

	emptyURI := []struct {
		ID      string `json:"id"`
		URI     string `json:"uri"`
		Keyfile string `json:"keyfile"`
	}{{ID: "a", URI: "", Keyfile: ""}}
	if _, ok := normalizeHosts(emptyURI); ok {
		t.Error("normalizeHosts(empty uri) should fail")
	}

	dupID := []struct {
		ID      string `json:"id"`
		URI     string `json:"uri"`
		Keyfile string `json:"keyfile"`
	}{
		{ID: "a", URI: "qemu:///system", Keyfile: ""},
		{ID: "a", URI: "qemu:///session", Keyfile: ""},
	}
	if _, ok := normalizeHosts(dupID); ok {
		t.Error("normalizeHosts(duplicate id) should fail")
	}

	qemuSSHNoKeyfile := []struct {
		ID      string `json:"id"`
		URI     string `json:"uri"`
		Keyfile string `json:"keyfile"`
	}{{ID: "a", URI: "qemu+ssh://host/system", Keyfile: ""}}
	if _, ok := normalizeHosts(qemuSSHNoKeyfile); ok {
		t.Error("normalizeHosts(qemu+ssh without keyfile) should fail")
	}
}

func TestContainsHost(t *testing.T) {
	t.Parallel()
	hosts := []config.Host{
		{ID: "a", URI: "qemu:///system"},
		{ID: "b", URI: "qemu:///session"},
	}
	if !containsHost(hosts, "a") {
		t.Error("containsHost(hosts, 'a') = false, want true")
	}
	if containsHost(hosts, "c") {
		t.Error("containsHost(hosts, 'c') = true, want false")
	}
}

func TestClientIPFromRequest(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		req  *http.Request
		want string
	}{
		{"X-Real-IP", httptest.NewRequest(http.MethodGet, "/", nil), ""},
		{"RemoteAddr", httptest.NewRequest(http.MethodGet, "/", nil), ""},
	}
	// Set headers/RemoteAddr after creation
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.Header.Set("X-Real-IP", "192.168.1.1")
	if got := clientIPFromRequest(req1); got != "192.168.1.1" {
		t.Errorf("X-Real-IP: got %q, want 192.168.1.1", got)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	if got := clientIPFromRequest(req2); got != "10.0.0.1" {
		t.Errorf("X-Forwarded-For: got %q, want 10.0.0.1", got)
	}

	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.RemoteAddr = "172.16.0.1:12345"
	if got := clientIPFromRequest(req3); got != "172.16.0.1" {
		t.Errorf("RemoteAddr: got %q, want 172.16.0.1", got)
	}
	_ = tests
}

func TestWriteConfigFile(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "subdir", "config.yaml")
	payload := map[string]string{"key": "value"}
	if err := writeConfigFile(path, payload); err != nil {
		t.Fatalf("writeConfigFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(data), "key") || !strings.Contains(string(data), "value") {
		t.Errorf("unexpected content: %s", data)
	}
}

func TestSanitizeValidationError_Truncates(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 300)
	got := sanitizeValidationError(long)
	if len(got) > 256 {
		t.Errorf("sanitizeValidationError should truncate to 256, got len %d", len(got))
	}
}

// authHandler returns a router with mock ConnectorProvider and a helper to get a valid token.
func authHandler(t *testing.T, connectorProvider ConnectorProvider) (http.Handler, string) {
	t.Helper()
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
	t.Cleanup(func() { _ = database.Close() })
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_, err = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	opts := RouterOptions{
		Logger:            nil,
		DB:                database,
		Config:            loaded,
		ConfigPath:        configPath,
		ConfigPresent:     true,
		DBPath:            filepath.Join(tempDir, "kui.db"),
		GitPath:           tempDir,
		ConnectorProvider: connectorProvider,
	}
	handler := NewRouter(opts)
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", loginRec.Code, loginRec.Body.String())
	}
	var loginResp struct{ Token string }
	if err := json.NewDecoder(loginRec.Body).Decode(&loginResp); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	return handler, loginResp.Token
}

func TestGetVMs_WithMockConnector(t *testing.T) {
	t.Parallel()
	mock := &mockConnector{
		domains: []libvirtconn.DomainInfo{
			{Name: "vm1", UUID: "uuid-1", State: libvirtconn.DomainStateRunning},
		},
	}
	handler, token := authHandler(t, func(ctx context.Context, hostID string) (libvirtconn.Connector, error) {
		if hostID != "local" {
			return nil, errors.New("host not found")
		}
		return mock, nil
	})
	req := httptest.NewRequest(http.MethodGet, "/api/vms", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body vmListResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.VMs) == 0 && len(body.Orphans) == 0 {
		t.Log("VMs/Orphans empty (no claimed metadata); hosts should show local")
	}
	if body.Hosts["local"] != "online" {
		t.Errorf("expected local=online, got %q", body.Hosts["local"])
	}
}

func TestGetVMs_ConnectorError(t *testing.T) {
	t.Parallel()
	handler, token := authHandler(t, func(ctx context.Context, hostID string) (libvirtconn.Connector, error) {
		return nil, errors.New("connection refused")
	})
	req := httptest.NewRequest(http.MethodGet, "/api/vms", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (hosts offline), got %d", rec.Code)
	}
	var body vmListResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Hosts["local"] != "offline" {
		t.Errorf("expected local=offline when connector fails, got %q", body.Hosts["local"])
	}
}

func TestGetHostPools_WithMockConnector(t *testing.T) {
	t.Parallel()
	mock := &mockConnector{
		pools: []libvirtconn.StoragePoolInfo{
			{Name: "default", UUID: "u1", Active: true},
		},
	}
	handler, token := authHandler(t, func(ctx context.Context, hostID string) (libvirtconn.Connector, error) {
		return mock, nil
	})
	req := httptest.NewRequest(http.MethodGet, "/api/hosts/local/pools", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body []poolResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 1 || body[0].Name != "default" {
		t.Errorf("unexpected pools: %+v", body)
	}
}

func TestGetHostPools_HostNotFound(t *testing.T) {
	t.Parallel()
	handler, token := authHandler(t, func(ctx context.Context, hostID string) (libvirtconn.Connector, error) {
		return nil, errors.New("host not found")
	})
	req := httptest.NewRequest(http.MethodGet, "/api/hosts/nonexistent/pools", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetHostPoolVolumes_WithMockConnector(t *testing.T) {
	t.Parallel()
	mock := &mockConnector{
		volumes: []libvirtconn.StorageVolumeInfo{
			{Name: "disk.qcow2", Path: "/path/disk.qcow2", Capacity: 1024},
		},
	}
	handler, token := authHandler(t, func(ctx context.Context, hostID string) (libvirtconn.Connector, error) {
		return mock, nil
	})
	req := httptest.NewRequest(http.MethodGet, "/api/hosts/local/pools/default/volumes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body []volumeResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 1 || body[0].Name != "disk.qcow2" {
		t.Errorf("unexpected volumes: %+v", body)
	}
}

func TestGetHostNetworks_WithMockConnector(t *testing.T) {
	t.Parallel()
	mock := &mockConnector{
		networks: []libvirtconn.NetworkInfo{
			{Name: "default", UUID: "n1", Active: true},
		},
	}
	handler, token := authHandler(t, func(ctx context.Context, hostID string) (libvirtconn.Connector, error) {
		return mock, nil
	})
	req := httptest.NewRequest(http.MethodGet, "/api/hosts/local/networks", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body []networkResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 1 || body[0].Name != "default" {
		t.Errorf("unexpected networks: %+v", body)
	}
}

func TestIsLocalLibvirtURI(t *testing.T) {
	t.Parallel()
	tests := []struct {
		uri  string
		want bool
	}{
		{"qemu:///system", true},
		{"qemu+unix:///session", true},
		{"qemu+ssh://host/system", false},
		{"  qemu:///system  ", true},
	}
	for _, tt := range tests {
		got := isLocalLibvirtURI(tt.uri)
		if got != tt.want {
			t.Errorf("isLocalLibvirtURI(%q) = %v, want %v", tt.uri, got, tt.want)
		}
	}
}

func TestGetVMDetail_WithMockConnector(t *testing.T) {
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
	loaded, _ := config.Load(configPath)
	database, _ := db.Open(filepath.Join(tempDir, "kui.db"))
	t.Cleanup(func() { _ = database.Close() })
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	disp := "My VM"
	_ = database.InsertVMMetadata(context.Background(), "local", "uuid-claimed", true, &disp)

	mock := &mockConnector{
		domainInfo: libvirtconn.DomainInfo{Name: "claimed-vm", UUID: "uuid-claimed", State: libvirtconn.DomainStateRunning},
	}
	handler := NewRouter(RouterOptions{
		Logger:            nil,
		DB:                database,
		Config:            loaded,
		ConfigPath:        configPath,
		ConfigPresent:     true,
		DBPath:            filepath.Join(tempDir, "kui.db"),
		GitPath:           tempDir,
		ConnectorProvider: func(ctx context.Context, hostID string) (libvirtconn.Connector, error) { return mock, nil },
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: %d", loginRec.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/hosts/local/vms/uuid-claimed", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body vmDetailResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.DisplayName == nil || *body.DisplayName != "My VM" {
		t.Errorf("unexpected display name: %v", body.DisplayName)
	}
}

func TestGetVMDetail_HostNotFound(t *testing.T) {
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
	loaded, _ := config.Load(configPath)
	database, _ := db.Open(filepath.Join(tempDir, "kui.db"))
	t.Cleanup(func() { _ = database.Close() })
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	disp := "My VM"
	_ = database.InsertVMMetadata(context.Background(), "local", "uuid-claimed", true, &disp)

	handler := NewRouter(RouterOptions{
		Logger:            nil,
		DB:                database,
		Config:            loaded,
		ConfigPath:        configPath,
		ConfigPresent:     true,
		DBPath:            filepath.Join(tempDir, "kui.db"),
		GitPath:           tempDir,
		ConnectorProvider: func(ctx context.Context, hostID string) (libvirtconn.Connector, error) {
			return nil, errors.New("host not found")
		},
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)

	req := httptest.NewRequest(http.MethodGet, "/api/hosts/nonexistent/vms/uuid-claimed", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestVMClaim_WithMockConnector(t *testing.T) {
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
	loaded, _ := config.Load(configPath)
	database, _ := db.Open(filepath.Join(tempDir, "kui.db"))
	t.Cleanup(func() { _ = database.Close() })
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)

	mock := &mockConnector{
		domainInfo: libvirtconn.DomainInfo{Name: "orphan-vm", UUID: "uuid-orphan", State: libvirtconn.DomainStateShutoff},
	}
	handler := NewRouter(RouterOptions{
		Logger:            nil,
		DB:                database,
		Config:            loaded,
		ConfigPath:        configPath,
		ConfigPresent:     true,
		DBPath:            filepath.Join(tempDir, "kui.db"),
		GitPath:           tempDir,
		ConnectorProvider: func(ctx context.Context, hostID string) (libvirtconn.Connector, error) { return mock, nil },
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: %d", loginRec.Code)
	}

	claimBody := []byte(`{"display_name":"Claimed VM"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/local/vms/uuid-orphan/claim", bytes.NewReader(claimBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body vmDetailResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.DisplayName == nil || *body.DisplayName != "Claimed VM" {
		t.Errorf("unexpected display name: %v", body.DisplayName)
	}
}

// --- putPreferences ---

func TestPutPreferences_Success(t *testing.T) {
	t.Parallel()
	handler, token := authHandler(t, nil)
	payload := map[string]any{
		"default_host_id": "local",
		"list_view_options": map[string]any{
			"list_view": map[string]any{"sort": "name", "page_size": 25, "group_by": "last_access"},
			"onboarding_dismissed": true,
		},
	}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/api/preferences", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp preferencesResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.DefaultHostID == nil || *resp.DefaultHostID != "local" {
		t.Errorf("unexpected default_host_id: %v", resp.DefaultHostID)
	}
	if resp.ListViewOptions == nil || resp.ListViewOptions.ListView == nil {
		t.Error("expected list_view_options in response")
	}
}

func TestPutPreferences_InvalidHostID(t *testing.T) {
	t.Parallel()
	handler, token := authHandler(t, nil)
	payload := map[string]any{"default_host_id": "nonexistent"}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/api/preferences", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPutPreferences_EmptyDefaultHost(t *testing.T) {
	t.Parallel()
	handler, token := authHandler(t, nil)
	payload := map[string]any{"default_host_id": ""}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPut, "/api/preferences", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- login error paths ---

func TestLogin_BadJSON(t *testing.T) {
	t.Parallel()
	handler, _ := authHandler(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLogin_EmptyCredentials(t *testing.T) {
	t.Parallel()
	handler, _ := authHandler(t, nil)
	payload := map[string]string{"username": "", "password": "x"}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	t.Parallel()
	handler, _ := authHandler(t, nil)
	payload := map[string]string{"username": "admin", "password": "wrong"}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	t.Parallel()
	handler, _ := authHandler(t, nil)
	payload := map[string]string{"username": "nobody", "password": "secret"}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- logout ---

func TestLogout_Success(t *testing.T) {
	t.Parallel()
	handler, token := authHandler(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if c := rec.Result().Cookies(); len(c) == 0 || c[0].Value != "" {
		t.Error("expected session cookie to be cleared")
	}
}

func TestLogout_WithoutAuth(t *testing.T) {
	t.Parallel()
	handler, _ := authHandler(t, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	// Logout clears cookie and returns 200; middleware may return 401 when no user in context
	if rec.Code != http.StatusOK && rec.Code != http.StatusUnauthorized {
		t.Errorf("logout without auth: expected 200 or 401, got %d", rec.Code)
	}
}

// --- patchVMConfig ---

func TestPatchVMConfig_DisplayNameOnly(t *testing.T) {
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
	loaded, _ := config.Load(configPath)
	database, _ := db.Open(filepath.Join(tempDir, "kui.db"))
	t.Cleanup(func() { _ = database.Close() })
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	_ = database.InsertVMMetadata(context.Background(), "local", "uuid-vm", true, nil)

	mock := &mockConnector{
		domainInfo: libvirtconn.DomainInfo{Name: "vm1", UUID: "uuid-vm", State: libvirtconn.DomainStateRunning},
	}
	handler := NewRouter(RouterOptions{
		Logger:            nil,
		DB:                database,
		Config:            loaded,
		ConfigPath:        configPath,
		ConfigPresent:     true,
		DBPath:            filepath.Join(tempDir, "kui.db"),
		GitPath:           tempDir,
		ConnectorProvider: func(ctx context.Context, hostID string) (libvirtconn.Connector, error) { return mock, nil },
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)

	payload := map[string]any{"display_name": "My VM"}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/hosts/local/vms/uuid-vm", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body vmDetailResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.DisplayName == nil || *body.DisplayName != "My VM" {
		t.Errorf("unexpected display name: %v", body.DisplayName)
	}
}

func TestPatchVMConfig_VMNotFound(t *testing.T) {
	t.Parallel()
	handler, token := authHandler(t, func(ctx context.Context, hostID string) (libvirtconn.Connector, error) {
		return &mockConnector{}, nil
	})
	payload := map[string]any{"display_name": "x"}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPatch, "/api/hosts/local/vms/00000000-0000-0000-0000-000000000000", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- vmLifecycleOp (start, pause, resume, destroy) ---

func TestVMStart_Success(t *testing.T) {
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
	loaded, _ := config.Load(configPath)
	database, _ := db.Open(filepath.Join(tempDir, "kui.db"))
	t.Cleanup(func() { _ = database.Close() })
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	_ = database.InsertVMMetadata(context.Background(), "local", "uuid-vm", true, nil)

	mock := &mockConnector{
		domainInfo: libvirtconn.DomainInfo{Name: "vm1", UUID: "uuid-vm", State: libvirtconn.DomainStateShutoff},
		state:      libvirtconn.DomainStateRunning,
	}
	handler := NewRouter(RouterOptions{
		Logger:            nil,
		DB:                database,
		Config:            loaded,
		ConfigPath:        configPath,
		ConfigPresent:     true,
		DBPath:            filepath.Join(tempDir, "kui.db"),
		GitPath:           tempDir,
		ConnectorProvider: func(ctx context.Context, hostID string) (libvirtconn.Connector, error) { return mock, nil },
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)

	req := httptest.NewRequest(http.MethodPost, "/api/hosts/local/vms/uuid-vm/start", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "running" {
		t.Errorf("expected status=running, got %q", body["status"])
	}
}

func TestVMPause_Success(t *testing.T) {
	t.Parallel()
	handler, token := authHandlerWithClaimedVM(t, libvirtconn.DomainStateRunning, libvirtconn.DomainStatePaused)
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/local/vms/uuid-vm/pause", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "paused" {
		t.Errorf("expected status=paused, got %q", body["status"])
	}
}

func TestVMResume_Success(t *testing.T) {
	t.Parallel()
	handler, token := authHandlerWithClaimedVM(t, libvirtconn.DomainStatePaused, libvirtconn.DomainStateRunning)
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/local/vms/uuid-vm/resume", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "running" {
		t.Errorf("expected status=running, got %q", body["status"])
	}
}

func TestVMDestroy_Success(t *testing.T) {
	t.Parallel()
	handler, token := authHandlerWithClaimedVM(t, libvirtconn.DomainStateRunning, libvirtconn.DomainStateShutoff)
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/local/vms/uuid-vm/destroy", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "shutoff" {
		t.Errorf("expected status=shutoff, got %q", body["status"])
	}
}

// authHandlerWithClaimedVM returns a router with a claimed VM and mock connector for lifecycle tests.
func authHandlerWithClaimedVM(t *testing.T, fromState, toState libvirtconn.DomainLifecycleState) (http.Handler, string) {
	t.Helper()
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
	loaded, _ := config.Load(configPath)
	database, _ := db.Open(filepath.Join(tempDir, "kui.db"))
	t.Cleanup(func() { _ = database.Close() })
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	_ = database.InsertVMMetadata(context.Background(), "local", "uuid-vm", true, nil)

	mock := &mockConnector{
		domainInfo: libvirtconn.DomainInfo{Name: "vm1", UUID: "uuid-vm", State: fromState},
		state:      toState,
	}
	handler := NewRouter(RouterOptions{
		Logger:            nil,
		DB:                database,
		Config:            loaded,
		ConfigPath:        configPath,
		ConfigPresent:     true,
		DBPath:            filepath.Join(tempDir, "kui.db"),
		GitPath:           tempDir,
		ConnectorProvider: func(ctx context.Context, hostID string) (libvirtconn.Connector, error) { return mock, nil },
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)
	return handler, loginResp.Token
}

// --- vmRecover ---

func TestVMRecover_Success(t *testing.T) {
	t.Parallel()
	handler, token := authHandlerWithClaimedVM(t, libvirtconn.DomainStateShutoff, libvirtconn.DomainStateShutoff)
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/local/vms/uuid-vm/recover", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "undefined" {
		t.Errorf("expected status=undefined, got %q", body["status"])
	}
}

// --- vmStop ---

func TestVMStop_Success(t *testing.T) {
	t.Parallel()
	handler, token := authHandlerWithClaimedVM(t, libvirtconn.DomainStateRunning, libvirtconn.DomainStateShutoff)
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/local/vms/uuid-vm/stop", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["status"] != "shutoff" {
		t.Errorf("expected status=shutoff, got %q", body["status"])
	}
}

// --- createVM ---

func TestCreateVM_Success(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	cfgYAML := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "` + testJWTSecret + `"
default_host: local
default_pool: default
vm_defaults:
  cpu: 2
  ram_mb: 2048
  network: default
`)
	if err := os.WriteFile(configPath, cfgYAML, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	loaded, _ := config.Load(configPath)
	database, _ := db.Open(filepath.Join(tempDir, "kui.db"))
	t.Cleanup(func() { _ = database.Close() })
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)

	mock := &mockConnector{
		volInfo:  libvirtconn.StorageVolumeInfo{Name: "kui-abc12345.qcow2", Path: "/var/lib/libvirt/images/kui-abc12345.qcow2"},
		domainInfo: libvirtconn.DomainInfo{Name: "kui-abc12345", UUID: "new-uuid-here", State: libvirtconn.DomainStateShutoff},
		pools: []libvirtconn.StoragePoolInfo{{Name: "default", UUID: "p1", Active: true}},
	}
	handler := NewRouter(RouterOptions{
		Logger:            nil,
		DB:                database,
		Config:            loaded,
		ConfigPath:        configPath,
		ConfigPresent:     true,
		DBPath:            filepath.Join(tempDir, "kui.db"),
		GitPath:           tempDir,
		ConnectorProvider: func(ctx context.Context, hostID string) (libvirtconn.Connector, error) { return mock, nil },
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)

	payload := map[string]any{
		"host_id": "local",
		"pool":    "default",
		"disk":    map[string]any{"size_mb": 10240},
		"display_name": "New VM",
	}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/vms", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var body createVMResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.HostID != "local" || body.DisplayName != "New VM" {
		t.Errorf("unexpected response: %+v", body)
	}
}

func TestCreateVM_ExistingVolume(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	cfgYAML := []byte(`hosts:
  - id: local
    uri: qemu:///system
jwt_secret: "` + testJWTSecret + `"
default_host: local
default_pool: default
`)
	if err := os.WriteFile(configPath, cfgYAML, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	loaded, _ := config.Load(configPath)
	database, _ := db.Open(filepath.Join(tempDir, "kui.db"))
	t.Cleanup(func() { _ = database.Close() })
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)

	mock := &mockConnector{
		volumes: []libvirtconn.StorageVolumeInfo{{Name: "existing.qcow2", Path: "/path/existing.qcow2", Capacity: 1024}},
		domainInfo: libvirtconn.DomainInfo{Name: "kui-abc12345", UUID: "new-uuid", State: libvirtconn.DomainStateShutoff},
	}
	handler := NewRouter(RouterOptions{
		Logger:            nil,
		DB:                database,
		Config:            loaded,
		ConfigPath:        configPath,
		ConfigPresent:     true,
		DBPath:            filepath.Join(tempDir, "kui.db"),
		GitPath:           tempDir,
		ConnectorProvider: func(ctx context.Context, hostID string) (libvirtconn.Connector, error) { return mock, nil },
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)

	payload := map[string]any{"host_id": "local", "pool": "default", "disk": map[string]any{"name": "existing.qcow2"}}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/vms", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateVM_MissingPool(t *testing.T) {
	t.Parallel()
	handler, token := authHandler(t, nil)
	payload := map[string]any{"host_id": "local", "pool": ""}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/vms", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- vmClone ---

func TestVMClone_Success(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")
	cfgYAML := []byte(`hosts:
  - id: local
    uri: qemu:///system
  - id: remote
    uri: qemu:///session
jwt_secret: "` + testJWTSecret + `"
`)
	if err := os.WriteFile(configPath, cfgYAML, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	loaded, _ := config.Load(configPath)
	database, _ := db.Open(filepath.Join(tempDir, "kui.db"))
	t.Cleanup(func() { _ = database.Close() })
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	_ = database.InsertVMMetadata(context.Background(), "local", "uuid-src", true, nil)

	domainXML := `<domain><devices><disk><source file="/pool/disk.qcow2"/></disk></devices></domain>`
	sourceMock := &mockConnector{
		state:      libvirtconn.DomainStateShutoff,
		domainInfo: libvirtconn.DomainInfo{Name: "src-vm", UUID: "uuid-src", State: libvirtconn.DomainStateShutoff},
		domainXML:  domainXML,
		pools:      []libvirtconn.StoragePoolInfo{{Name: "default", UUID: "p1", Active: true}},
		volumes:    []libvirtconn.StorageVolumeInfo{{Name: "disk.qcow2", Path: "/pool/disk.qcow2", Capacity: 1024}},
	}
	targetMock := &mockConnector{
		domainInfo: libvirtconn.DomainInfo{Name: "src-vm-clone", UUID: "uuid-clone", State: libvirtconn.DomainStateShutoff},
		volInfo:    libvirtconn.StorageVolumeInfo{Name: "src-vm-clone.qcow2", Path: "/target/src-vm-clone.qcow2", Capacity: 1024},
		pools:      []libvirtconn.StoragePoolInfo{{Name: "images", UUID: "p2", Active: true}},
	}
	cp := func(ctx context.Context, hostID string) (libvirtconn.Connector, error) {
		switch hostID {
		case "local":
			return sourceMock, nil
		case "remote":
			return targetMock, nil
		default:
			return nil, errors.New("host not found")
		}
	}
	handler := NewRouter(RouterOptions{
		Logger:            nil,
		DB:                database,
		Config:            loaded,
		ConfigPath:        configPath,
		ConfigPresent:     true,
		DBPath:            filepath.Join(tempDir, "kui.db"),
		GitPath:           tempDir,
		ConnectorProvider: cp,
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)

	payload := map[string]any{"target_host_id": "remote", "target_pool": "images"}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/local/vms/uuid-src/clone", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var body createVMResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.HostID != "remote" || body.DisplayName != "src-vm-clone" {
		t.Errorf("unexpected response: %+v", body)
	}
}

func TestVMClone_SourceNotStopped(t *testing.T) {
	t.Parallel()
	handler, token := authHandlerWithClaimedVM(t, libvirtconn.DomainStateRunning, libvirtconn.DomainStateRunning)
	payload := map[string]any{"target_host_id": "local", "target_pool": "default"}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/hosts/local/vms/uuid-vm/clone", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- getTemplates ---

func TestGetTemplates_EmptyList(t *testing.T) {
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
	loaded, _ := config.Load(configPath)
	database, _ := db.Open(filepath.Join(tempDir, "kui.db"))
	t.Cleanup(func() { _ = database.Close() })
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	gitDir := filepath.Join(tempDir, "git")
	if err := os.MkdirAll(filepath.Join(gitDir, "templates"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	handler := NewRouter(RouterOptions{
		Logger:        nil,
		DB:            database,
		Config:        loaded,
		ConfigPath:    configPath,
		ConfigPresent: true,
		DBPath:        filepath.Join(tempDir, "kui.db"),
		GitPath:       gitDir,
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)

	req := httptest.NewRequest(http.MethodGet, "/api/templates", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body []templateListItem
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body) != 0 {
		t.Errorf("expected empty list, got %d items", len(body))
	}
}

func TestGetTemplates_ListFails(t *testing.T) {
	t.Parallel()
	// Use a path that causes ListTemplates to fail (e.g. templates is a file, not dir)
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
	loaded, _ := config.Load(configPath)
	database, _ := db.Open(filepath.Join(tempDir, "kui.db"))
	t.Cleanup(func() { _ = database.Close() })
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	gitDir := filepath.Join(tempDir, "git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Create templates as a file so ListTemplates fails
	if err := os.WriteFile(filepath.Join(gitDir, "templates"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	handler := NewRouter(RouterOptions{
		Logger:        nil,
		DB:            database,
		Config:        loaded,
		ConfigPath:    configPath,
		ConfigPresent: true,
		DBPath:        filepath.Join(tempDir, "kui.db"),
		GitPath:       gitDir,
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)

	req := httptest.NewRequest(http.MethodGet, "/api/templates", nil)
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when ListTemplates fails, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- createTemplate ---

func TestCreateTemplate_Success(t *testing.T) {
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
	loaded, _ := config.Load(configPath)
	database, _ := db.Open(filepath.Join(tempDir, "kui.db"))
	t.Cleanup(func() { _ = database.Close() })
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	_, _ = database.SQL.Exec(
		`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		"user-1", "admin", string(hash), "admin", "2026-03-16T00:00:00Z", "2026-03-16T00:00:00Z",
	)
	_ = database.InsertVMMetadata(context.Background(), "local", "uuid-src", true, nil)

	gitDir := filepath.Join(tempDir, "git")
	if err := os.MkdirAll(filepath.Join(gitDir, "templates"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// audit.CommitPaths will init git repo via openOrInitRepo if needed

	domainXML := `<domain type="kvm"><name>src</name><devices><disk><source file="/pool/disk.qcow2"/></disk></devices></domain>`
	mock := &mockConnector{
		state:      libvirtconn.DomainStateShutoff,
		domainInfo: libvirtconn.DomainInfo{Name: "src", UUID: "uuid-src", State: libvirtconn.DomainStateShutoff},
		domainXML:  domainXML,
		pools:      []libvirtconn.StoragePoolInfo{{Name: "default", UUID: "p1", Active: true}},
		volumes:    []libvirtconn.StorageVolumeInfo{{Name: "disk.qcow2", Path: "/pool/disk.qcow2", Capacity: 1024}, {Name: "src.qcow2", Path: "/pool/src.qcow2", Capacity: 1024}},
	}
	handler := NewRouter(RouterOptions{
		Logger:            nil,
		DB:                database,
		Config:            loaded,
		ConfigPath:        configPath,
		ConfigPresent:     true,
		DBPath:            filepath.Join(tempDir, "kui.db"),
		GitPath:           gitDir,
		ConnectorProvider: func(ctx context.Context, hostID string) (libvirtconn.Connector, error) { return mock, nil },
	})
	loginBody, _ := json.Marshal(map[string]string{"username": "admin", "password": "secret"})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	handler.ServeHTTP(loginRec, loginReq)
	var loginResp struct{ Token string }
	_ = json.NewDecoder(loginRec.Body).Decode(&loginResp)

	payload := map[string]any{
		"source_host_id":       "local",
		"source_libvirt_uuid":   "uuid-src",
		"name":                  "My Template",
		"target_pool":           "default",
	}
	bodyBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/templates", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+loginResp.Token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var body createTemplateResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.TemplateID == "" || body.Name != "My Template" {
		t.Errorf("unexpected response: %+v", body)
	}
}
