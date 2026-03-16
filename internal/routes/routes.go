package routes

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"

	"github.com/kui/kui/internal/config"
	"github.com/kui/kui/internal/db"
	"github.com/kui/kui/internal/libvirtconn"
	mw "github.com/kui/kui/internal/middleware"
)

type RouterOptions struct {
	Logger       *slog.Logger
	DB           *db.DB
	Config       *config.Config
	ConfigPath   string
	ConfigPresent bool
	DBPath       string
	GitPath      string
}

type routerState struct {
	logger          *slog.Logger
	db              *db.DB
	config          *config.Config
	configPath      string
	configPresent   bool
	dbPath          string
	gitPath         string
	setupCompleted  bool
	setupCompletedMu sync.Mutex
}

type setupStatusResponse struct {
	SetupRequired bool    `json:"setup_required"`
	Reason        *string `json:"reason"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

type userRecord struct {
	ID           string
	Username     string
	PasswordHash string
	Role         string
}

type meResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type validateHostRequest struct {
	HostID  string `json:"host_id"`
	URI     string `json:"uri"`
	Keyfile string `json:"keyfile"`
}

type validateHostResponse struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

type listViewOptions struct {
	ListView           *listView `json:"list_view,omitempty"`
	OnboardingDismissed *bool     `json:"onboarding_dismissed,omitempty"`
}

type listView struct {
	Sort    string `json:"sort,omitempty"`
	PageSize int    `json:"page_size,omitempty"`
	GroupBy string `json:"group_by,omitempty"`
}

type preferencesResponse struct {
	DefaultHostID   *string           `json:"default_host_id"`
	ListViewOptions *listViewOptions  `json:"list_view_options"`
}

type preferencesPutRequest struct {
	DefaultHostID   *string           `json:"default_host_id"`
	ListViewOptions *listViewOptions  `json:"list_view_options"`
}

type hostResponse struct {
	ID  string `json:"id"`
	URI string `json:"uri"`
}

type setupCompleteRequest struct {
	Admin struct {
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"admin"`
	Hosts []struct {
		ID      string `json:"id"`
		URI     string `json:"uri"`
		Keyfile string `json:"keyfile"`
	} `json:"hosts"`
	DefaultHost string `json:"default_host"`
}

type setupPersistedConfig struct {
	Hosts             []config.Host `yaml:"hosts"`
	DefaultHost       string        `yaml:"default_host"`
	Session           sessionConfig `yaml:"session"`
	JWTSecret         string        `yaml:"jwt_secret"`
	DB                dbPathConfig  `yaml:"db"`
	Git               gitPathConfig `yaml:"git"`
	DefaultNameTemplate string      `yaml:"default_name_template"`
}

type sessionConfig struct {
	Timeout string `yaml:"timeout"`
}

type dbPathConfig struct {
	Path string `yaml:"path"`
}

type gitPathConfig struct {
	Path string `yaml:"path"`
}

type jwtClaims struct {
	jwt.RegisteredClaims
	Username string `json:"username"`
	Role     string `json:"role"`
}

func NewRouter(opts RouterOptions) http.Handler {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	state := &routerState{
		logger:         logger,
		db:             opts.DB,
		config:         opts.Config,
		configPath:     opts.ConfigPath,
		configPresent:  opts.ConfigPresent,
		dbPath:         opts.DBPath,
		gitPath:        opts.GitPath,
	}

	sessionTimeout := 24 * time.Hour
	secret := ""
	secureCookies := true
	allowedOrigins := []string{"http://localhost:5173"}
	if opts.Config != nil {
		sessionTimeout = time.Duration(opts.Config.Session.Timeout)
		secret = opts.Config.JWTSecret
		if opts.Config.Session.SecureCookies != nil {
			secureCookies = *opts.Config.Session.SecureCookies
		}
		if len(opts.Config.CORS.AllowedOrigins) > 0 {
			allowedOrigins = opts.Config.CORS.AllowedOrigins
		}
	}

	router := chi.NewRouter()
	router.Use(mw.RequestID())
	router.Use(mw.Logging(logger))
	router.Use(mw.CORS(allowedOrigins))
	router.Use(mw.Recovery(logger))
	router.Use(mw.Auth(secret, mw.AuthOptions{
		SkipExactPaths: []string{"/", "/api/auth/login"},
		SkipPrefixPaths: []string{
			"/api/setup/",
		},
	}))

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("KUI API"))
	})
	router.Post("/api/auth/login", state.login(sessionTimeout, secret, secureCookies))
	router.Get("/api/setup/status", state.setupStatus())
	router.Post("/api/setup/validate-host", state.validateHost())
	router.Post("/api/setup/complete", state.setupComplete())
	router.Post("/api/auth/logout", state.logout())
	router.Get("/api/auth/me", state.me())
	router.Get("/api/preferences", state.getPreferences())
	router.Put("/api/preferences", state.putPreferences())
	router.Get("/api/hosts", state.getHosts())

	return router
}

func (r *routerState) login(sessionTimeout time.Duration, secret string, secureCookies bool) http.HandlerFunc {
	loginLimiter := newLoginRateLimiter(5, time.Minute)
	return func(w http.ResponseWriter, req *http.Request) {
		if secret == "" {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		clientIP := clientIPFromRequest(req)
		if !loginLimiter.allow(clientIP) {
			r.logger.Warn("login rate limit exceeded", "ip", clientIP)
			writeJSONError(w, http.StatusTooManyRequests, "too many login attempts")
			return
		}

		var payload loginRequest
		if err := decodeJSON(req.Body, &payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "username and password required")
			return
		}
		if strings.TrimSpace(payload.Username) == "" || strings.TrimSpace(payload.Password) == "" {
			writeJSONError(w, http.StatusBadRequest, "username and password required")
			return
		}

		record, err := r.findUserByUsername(req.Context(), payload.Username)
		if err != nil {
			loginLimiter.recordFailure(clientIP)
			r.logger.Warn("login failed", "ip", clientIP, "username", payload.Username)
			writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(record.PasswordHash), []byte(payload.Password)); err != nil {
			loginLimiter.recordFailure(clientIP)
			r.logger.Warn("login failed", "ip", clientIP, "username", payload.Username)
			writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		now := time.Now().UTC()
		expiresAt := now.Add(sessionTimeout)
		token, err := signJWT(record, now, expiresAt, secret)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to sign token")
			return
		}

		secure := secureCookies || req.TLS != nil
		http.SetCookie(w, &http.Cookie{
			Name:     mw.SessionCookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(sessionTimeout.Seconds()),
			Expires:  expiresAt,
		})
		writeJSON(w, http.StatusOK, loginResponse{
			Token:     token,
			ExpiresAt: expiresAt.Format(time.RFC3339),
		})
	}
}

func (r *routerState) logout() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     mw.SessionCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		w.WriteHeader(http.StatusOK)
		_ = req
	}
}

func (r *routerState) me() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		user, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		writeJSON(w, http.StatusOK, meResponse{
			ID:       user.ID,
			Username: user.Username,
			Role:     user.Role,
		})
	}
}

func (r *routerState) getPreferences() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		user, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		var defaultHostID sql.NullString
		var listViewOptionsRaw sql.NullString
		err := r.db.SQL.QueryRowContext(req.Context(),
			`SELECT default_host_id, list_view_options FROM preferences WHERE user_id = ?`,
			user.ID,
		).Scan(&defaultHostID, &listViewOptionsRaw)
		if err == sql.ErrNoRows {
			writeJSON(w, http.StatusOK, preferencesResponse{
				DefaultHostID:   nil,
				ListViewOptions: nil,
			})
			return
		}
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to load preferences")
			return
		}
		resp := preferencesResponse{}
		if defaultHostID.Valid && defaultHostID.String != "" {
			resp.DefaultHostID = &defaultHostID.String
		}
		if listViewOptionsRaw.Valid && listViewOptionsRaw.String != "" {
			var opts listViewOptions
			if err := json.Unmarshal([]byte(listViewOptionsRaw.String), &opts); err == nil {
				resp.ListViewOptions = &opts
			}
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func (r *routerState) putPreferences() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		user, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		var payload preferencesPutRequest
		if err := decodeJSON(req.Body, &payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		var defaultHostID sql.NullString
		var listViewOptionsRaw sql.NullString
		err := r.db.SQL.QueryRowContext(req.Context(),
			`SELECT default_host_id, list_view_options FROM preferences WHERE user_id = ?`,
			user.ID,
		).Scan(&defaultHostID, &listViewOptionsRaw)
		if err != nil && err != sql.ErrNoRows {
			writeJSONError(w, http.StatusInternalServerError, "failed to load preferences")
			return
		}
		mergedDefaultHost := defaultHostID.String
		if payload.DefaultHostID != nil {
			if strings.TrimSpace(*payload.DefaultHostID) == "" {
				mergedDefaultHost = ""
			} else {
				hostID := strings.TrimSpace(*payload.DefaultHostID)
				if r.config == nil || !containsHost(r.config.Hosts, hostID) {
					writeJSONError(w, http.StatusBadRequest, "default_host_id is not configured")
					return
				}
				mergedDefaultHost = hostID
			}
		}
		var mergedOpts listViewOptions
		if listViewOptionsRaw.Valid && listViewOptionsRaw.String != "" {
			_ = json.Unmarshal([]byte(listViewOptionsRaw.String), &mergedOpts)
		}
		if payload.ListViewOptions != nil {
			if payload.ListViewOptions.ListView != nil {
				mergedOpts.ListView = payload.ListViewOptions.ListView
				if mergedOpts.ListView.GroupBy != "" && mergedOpts.ListView.GroupBy != "last_access" && mergedOpts.ListView.GroupBy != "created_at" {
					mergedOpts.ListView.GroupBy = "last_access"
				}
			}
			if payload.ListViewOptions.OnboardingDismissed != nil {
				mergedOpts.OnboardingDismissed = payload.ListViewOptions.OnboardingDismissed
			}
		}
		optsJSON, err := json.Marshal(mergedOpts)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to save preferences")
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		_, err = r.db.SQL.ExecContext(req.Context(),
			`INSERT INTO preferences (user_id, default_host_id, list_view_options, updated_at)
			 VALUES (?, ?, ?, ?)
			 ON CONFLICT(user_id) DO UPDATE SET
			   default_host_id = excluded.default_host_id,
			   list_view_options = excluded.list_view_options,
			   updated_at = excluded.updated_at`,
			user.ID, nullIfEmpty(mergedDefaultHost), string(optsJSON), now,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to save preferences")
			return
		}
		resp := preferencesResponse{}
		if mergedDefaultHost != "" {
			resp.DefaultHostID = &mergedDefaultHost
		}
		resp.ListViewOptions = &mergedOpts
		writeJSON(w, http.StatusOK, resp)
	}
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (r *routerState) getHosts() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if r.config == nil || len(r.config.Hosts) == 0 {
			writeJSON(w, http.StatusOK, []hostResponse{})
			return
		}
		out := make([]hostResponse, 0, len(r.config.Hosts))
		for _, h := range r.config.Hosts {
			out = append(out, hostResponse{ID: h.ID, URI: h.URI})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func (r *routerState) setupStatus() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var reason *string

		if !r.configPresent || r.config == nil {
			value := "config_missing"
			reason = &value
		} else {
			hasAdmin, err := r.hasAdminUser(req.Context())
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "failed to evaluate setup status")
				return
			}
			if !hasAdmin {
				value := "no_admin"
				reason = &value
			}
		}

		writeJSON(w, http.StatusOK, setupStatusResponse{
			SetupRequired: reason != nil,
			Reason:        reason,
		})
	}
}

func (r *routerState) validateHost() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if r.configPresent {
			writeJSONError(w, http.StatusForbidden, "validate-host is only available during setup")
			return
		}
		var payload validateHostRequest
		if err := decodeJSON(req.Body, &payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if strings.TrimSpace(payload.URI) == "" {
			writeJSONError(w, http.StatusBadRequest, "uri required")
			return
		}

		conn, err := libvirtconn.Connect(req.Context(), payload.URI, payload.Keyfile)
		if err != nil {
			r.logger.Debug("validate-host failed", "host_id", payload.HostID, "err", err)
			writeJSON(w, http.StatusOK, validateHostResponse{
				Valid: false,
				Error: "validation failed",
			})
			return
		}
		if err := conn.Close(); err != nil {
			r.logger.Debug("validate-host close failed", "host_id", payload.HostID, "err", err)
			writeJSON(w, http.StatusOK, validateHostResponse{
				Valid: false,
				Error: "validation failed",
			})
			return
		}

		writeJSON(w, http.StatusOK, validateHostResponse{
			Valid: true,
		})
	}
}

func (r *routerState) setupComplete() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var payload setupCompleteRequest
		if err := decodeJSON(req.Body, &payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if strings.TrimSpace(payload.Admin.Username) == "" {
			writeJSONError(w, http.StatusBadRequest, "admin username required")
			return
		}
		if strings.TrimSpace(payload.Admin.Password) == "" {
			writeJSONError(w, http.StatusBadRequest, "admin password required")
			return
		}
		if len(payload.Hosts) == 0 {
			writeJSONError(w, http.StatusBadRequest, "at least one host is required")
			return
		}
		if strings.TrimSpace(payload.DefaultHost) == "" {
			writeJSONError(w, http.StatusBadRequest, "default_host required")
			return
		}

		hosts, ok := normalizeHosts(payload.Hosts)
		if !ok {
			writeJSONError(w, http.StatusBadRequest, "invalid host payload")
			return
		}
		if !containsHost(hosts, payload.DefaultHost) {
			writeJSONError(w, http.StatusBadRequest, "default_host must be in hosts")
			return
		}

		r.setupCompletedMu.Lock()
		alreadyDone := r.setupCompleted
		r.setupCompletedMu.Unlock()
		if alreadyDone {
			writeJSONError(w, http.StatusConflict, "setup already complete")
			return
		}
		if r.configPresent {
			writeJSONError(w, http.StatusConflict, "setup already complete")
			return
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(payload.Admin.Password), bcrypt.DefaultCost)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to prepare credentials")
			return
		}

		now := time.Now().UTC().Format(time.RFC3339)
		userID, err := randomUUID()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to create admin user")
			return
		}
		if _, err := r.db.SQL.ExecContext(
			req.Context(),
			`INSERT INTO users (id, username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
			userID, strings.TrimSpace(payload.Admin.Username), string(passwordHash), "admin", now, now,
		); err != nil {
			writeJSONError(w, http.StatusConflict, "admin user already exists")
			return
		}

		jwtSecret, err := randomSecret(32)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to generate jwt secret")
			return
		}

		persist := setupPersistedConfig{
			Hosts:               hosts,
			DefaultHost:         payload.DefaultHost,
			Session:             sessionConfig{Timeout: "24h"},
			JWTSecret:           jwtSecret,
			DB:                  dbPathConfig{Path: r.configuredDBPath()},
			Git:                 gitPathConfig{Path: r.configuredGitPath()},
			DefaultNameTemplate: "{source}",
		}
		if err := writeConfigFile(r.configPath, persist); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to write config")
			return
		}
		if err := os.Chmod(r.configPath, 0o600); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to update config permissions")
			return
		}

		r.setupCompletedMu.Lock()
		r.setupCompleted = true
		r.setupCompletedMu.Unlock()

		w.WriteHeader(http.StatusCreated)
	}
}

func (r *routerState) findUserByUsername(ctx context.Context, username string) (userRecord, error) {
	if r.db == nil || r.db.SQL == nil {
		return userRecord{}, errors.New("database not initialized")
	}

	var record userRecord
	err := r.db.SQL.QueryRowContext(ctx, `
SELECT id, username, password_hash, role
  FROM users
 WHERE username = ?`, username).Scan(&record.ID, &record.Username, &record.PasswordHash, &record.Role)
	if err == sql.ErrNoRows {
		return userRecord{}, errors.New("user not found")
	}
	return record, err
}

func (r *routerState) hasAdminUser(ctx context.Context) (bool, error) {
	var count int
	if err := r.db.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *routerState) configuredDBPath() string {
	if strings.TrimSpace(r.dbPath) != "" {
		return r.dbPath
	}
	return "/var/lib/kui/kui.db"
}

func (r *routerState) configuredGitPath() string {
	if strings.TrimSpace(r.gitPath) != "" {
		return r.gitPath
	}
	return "/var/lib/kui"
}

func decodeJSON(body io.Reader, target any) error {
	return json.NewDecoder(body).Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	raw, err := json.Marshal(payload)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}

func normalizeHosts(in []struct {
	ID      string `json:"id"`
	URI     string `json:"uri"`
	Keyfile string `json:"keyfile"`
}) ([]config.Host, bool) {
	seen := map[string]struct{}{}
	out := make([]config.Host, 0, len(in))

	for _, host := range in {
		id := strings.TrimSpace(host.ID)
		uri := strings.TrimSpace(host.URI)
		keyfile := strings.TrimSpace(host.Keyfile)

		if id == "" || uri == "" {
			return nil, false
		}
		if strings.HasPrefix(uri, "qemu+ssh://") && keyfile == "" {
			return nil, false
		}
		if _, exists := seen[id]; exists {
			return nil, false
		}
		seen[id] = struct{}{}

		var pointer *string
		if keyfile != "" {
			pointer = &keyfile
		}
		out = append(out, config.Host{
			ID:      id,
			URI:     uri,
			Keyfile: pointer,
		})
	}

	return out, true
}

func containsHost(hosts []config.Host, id string) bool {
	for _, host := range hosts {
		if host.ID == id {
			return true
		}
	}
	return false
}

func randomSecret(size int) (string, error) {
	secret := make([]byte, size)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(secret), nil
}

func randomUUID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}

	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hex.EncodeToString(raw[:4]),
		hex.EncodeToString(raw[4:6]),
		hex.EncodeToString(raw[6:8]),
		hex.EncodeToString(raw[8:10]),
		hex.EncodeToString(raw[10:]),
	), nil
}

func writeConfigFile(path string, payload any) error {
	raw, err := yaml.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return fmt.Errorf("write config temp file: %w", err)
	}
	return os.Rename(tmp, path)
}

func clientIPFromRequest(r *http.Request) string {
	if x := r.Header.Get("X-Real-IP"); x != "" {
		return strings.TrimSpace(strings.Split(x, ",")[0])
	}
	if x := r.Header.Get("X-Forwarded-For"); x != "" {
		return strings.TrimSpace(strings.Split(x, ",")[0])
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	if host != "" {
		return host
	}
	return r.RemoteAddr
}

type loginRateLimiter struct {
	mu        sync.Mutex
	failures  map[string][]time.Time
	limit     int
	window    time.Duration
}

func newLoginRateLimiter(limit int, window time.Duration) *loginRateLimiter {
	return &loginRateLimiter{
		failures: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (l *loginRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked(ip)
	return len(l.failures[ip]) < l.limit
}

func (l *loginRateLimiter) recordFailure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now().UTC()
	l.failures[ip] = append(l.failures[ip], now)
	l.pruneLocked(ip)
}

func (l *loginRateLimiter) pruneLocked(ip string) {
	cutoff := time.Now().UTC().Add(-l.window)
	var kept []time.Time
	for _, t := range l.failures[ip] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) == 0 {
		delete(l.failures, ip)
	} else {
		l.failures[ip] = kept
	}
}

func signJWT(user userRecord, now time.Time, expiresAt time.Time, secret string) (string, error) {
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		Username: user.Username,
		Role:     user.Role,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

