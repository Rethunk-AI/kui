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
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"libvirt.org/go/libvirtxml"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"

	"github.com/kui/kui/internal/audit"
	"github.com/kui/kui/internal/broadcaster"
	"github.com/kui/kui/internal/config"
	"github.com/kui/kui/internal/db"
	"github.com/kui/kui/internal/libvirtconn"
	mw "github.com/kui/kui/internal/middleware"
	"github.com/kui/kui/internal/template"
	"github.com/kui/kui/web"
)

type RouterOptions struct {
	Logger        *slog.Logger
	DB            *db.DB
	Config        *config.Config
	ConfigPath    string
	ConfigPresent bool
	DBPath        string
	GitPath       string
	Broadcaster   *broadcaster.Broadcaster
}

type routerState struct {
	logger          *slog.Logger
	db              *db.DB
	config          *config.Config
	configPath      string
	configPresent   bool
	dbPath          string
	gitPath         string
	broadcaster     *broadcaster.Broadcaster
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

	bc := opts.Broadcaster
	if bc == nil {
		bc = broadcaster.NewBroadcaster()
	}
	state := &routerState{
		logger:         logger,
		db:             opts.DB,
		config:         opts.Config,
		configPath:     opts.ConfigPath,
		configPresent:  opts.ConfigPresent,
		dbPath:         opts.DBPath,
		gitPath:        opts.GitPath,
		broadcaster:    bc,
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
			"/assets/",
		},
	}))

	router.Post("/api/auth/login", state.login(sessionTimeout, secret, secureCookies))
	router.Get("/api/setup/status", state.setupStatus())
	router.Post("/api/setup/validate-host", state.validateHost())
	router.Post("/api/setup/complete", state.setupComplete())
	router.Post("/api/auth/logout", state.logout())
	router.Get("/api/auth/me", state.me())
	router.Get("/api/preferences", state.getPreferences())
	router.Put("/api/preferences", state.putPreferences())
	router.Get("/api/hosts", state.getHosts())

	router.Get("/api/vms", state.getVMs())
	router.Get("/api/hosts/{host_id}/pools", state.getHostPools())
	router.Get("/api/hosts/{host_id}/pools/{pool_name}/volumes", state.getHostPoolVolumes())
	router.Get("/api/hosts/{host_id}/networks", state.getHostNetworks())

	router.Get("/api/hosts/{host_id}/vms/{libvirt_uuid}", state.getVMDetail())
	router.Patch("/api/hosts/{host_id}/vms/{libvirt_uuid}", state.patchVMConfig())
	router.Post("/api/hosts/{host_id}/vms/{libvirt_uuid}/start", state.vmStart())
	router.Post("/api/hosts/{host_id}/vms/{libvirt_uuid}/stop", state.vmStop())
	router.Post("/api/hosts/{host_id}/vms/{libvirt_uuid}/pause", state.vmPause())
	router.Post("/api/hosts/{host_id}/vms/{libvirt_uuid}/resume", state.vmResume())
	router.Post("/api/hosts/{host_id}/vms/{libvirt_uuid}/destroy", state.vmDestroy())
	router.Post("/api/hosts/{host_id}/vms/{libvirt_uuid}/claim", state.vmClaim())
	router.Post("/api/hosts/{host_id}/vms/{libvirt_uuid}/clone", state.vmClone())

	router.Post("/api/vms", state.createVM())

	router.Get("/api/templates", state.getTemplates())
	router.Post("/api/templates", state.createTemplate())

	router.Get("/api/events", state.events())

	webFS := resolveWebFS()
	router.Get("/", staticHandler(webFS))
	router.Get("/*", staticHandler(webFS))

	return router
}

func resolveWebFS() http.FileSystem {
	if dir := strings.TrimSpace(os.Getenv("KUI_WEB_DIR")); dir != "" {
		return http.Dir(dir)
	}
	sub, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		return nil
	}
	return http.FS(sub)
}

func staticHandler(f http.FileSystem) http.HandlerFunc {
	if f == nil {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "web assets not available", http.StatusServiceUnavailable)
		}
	}
	fileServer := http.FileServer(f)
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		name := strings.TrimPrefix(path, "/")
		file, err := f.Open(name)
		if err == nil {
			_ = file.Close()
			fileServer.ServeHTTP(w, r)
			return
		}
		index, err := f.Open("index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer index.Close()
		stat, _ := index.Stat()
		http.ServeContent(w, r, "index.html", stat.ModTime(), index)
	}
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
			r.logger.Warn("login failed", "ip", clientIP)
			_ = audit.RecordEvent(req.Context(), r.db, audit.Event{
				EventType:  "auth",
				EntityType: "auth",
				EntityID:   "anonymous",
				UserID:     nil,
				Payload: map[string]string{
					"action": "login",
					"result": "failure",
					"ip":     clientIP,
				},
			})
			writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(record.PasswordHash), []byte(payload.Password)); err != nil {
			loginLimiter.recordFailure(clientIP)
			r.logger.Warn("login failed", "ip", clientIP)
			_ = audit.RecordEvent(req.Context(), r.db, audit.Event{
				EventType:  "auth",
				EntityType: "auth",
				EntityID:   "anonymous",
				UserID:     nil,
				Payload: map[string]string{
					"action": "login",
					"result": "failure",
					"ip":     clientIP,
				},
			})
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
		_ = audit.RecordEvent(req.Context(), r.db, audit.Event{
			EventType:  "auth",
			EntityType: "auth",
			EntityID:   record.ID,
			UserID:     &record.ID,
			Payload: map[string]string{
				"action": "login",
				"result": "success",
				"ip":     clientIP,
				"username": strings.TrimSpace(payload.Username),
			},
		})
		writeJSON(w, http.StatusOK, loginResponse{
			Token:     token,
			ExpiresAt: expiresAt.Format(time.RFC3339),
		})
	}
}

func (r *routerState) logout() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		user, ok := mw.UserFromContext(req)
		if ok {
			_ = audit.RecordEvent(req.Context(), r.db, audit.Event{
				EventType:  "auth",
				EntityType: "auth",
				EntityID:   user.ID,
				UserID:     &user.ID,
				Payload: map[string]string{
					"action": "logout",
					"result": "success",
					"ip":     clientIPFromRequest(req),
				},
			})
		}
		http.SetCookie(w, &http.Cookie{
			Name:     mw.SessionCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		w.WriteHeader(http.StatusOK)
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

func (r *routerState) events() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if r.broadcaster == nil {
			writeJSONError(w, http.StatusInternalServerError, "events not available")
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSONError(w, http.StatusInternalServerError, "streaming not supported")
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		sub := r.broadcaster.Subscribe(req.Context())
		defer sub.Done()

		for {
			select {
			case ev, ok := <-sub.C:
				if !ok {
					return
				}
				data, err := json.Marshal(ev.Data)
				if err != nil {
					r.logger.Warn("events: marshal event data", "err", err, "type", ev.Type)
					continue
				}
				if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, data); err != nil {
					return
				}
				flusher.Flush()
			case <-req.Context().Done():
				return
			}
		}
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

type vmListResponse struct {
	VMs     []vmListItem `json:"vms"`
	Hosts   map[string]string `json:"hosts"`
	Orphans []vmOrphanItem `json:"orphans"`
}

type vmListItem struct {
	HostID            string  `json:"host_id"`
	LibvirtUUID       string  `json:"libvirt_uuid"`
	DisplayName       *string `json:"display_name"`
	Claimed           bool    `json:"claimed"`
	Status            string  `json:"status"`
	ConsolePreference *string `json:"console_preference"`
	LastAccess        *string `json:"last_access"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type vmOrphanItem struct {
	HostID      string `json:"host_id"`
	LibvirtUUID string `json:"libvirt_uuid"`
	Name        string `json:"name"`
}

func (r *routerState) getVMs() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !r.configPresent || r.config == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "setup required")
			return
		}
		metadataRows, err := r.db.ListVMMetadata(req.Context())
		if err != nil {
			r.logger.Error("list vm_metadata failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to list VMs")
			return
		}
		metaByKey := make(map[string]db.VMMetadataRow)
		for _, row := range metadataRows {
			metaByKey[row.HostID+"\x00"+row.LibvirtUUID] = row
		}
		var vms []vmListItem
		var orphans []vmOrphanItem
		hosts := make(map[string]string)
		for _, h := range r.config.Hosts {
			conn, err := r.getConnectorForHost(req.Context(), h.ID)
			if err != nil {
				hosts[h.ID] = "offline"
				continue
			}
			hosts[h.ID] = "online"
			domains, err := conn.ListDomains(req.Context())
			conn.Close()
			if err != nil {
				r.logger.Error("list domains failed", "host_id", h.ID, "error", err)
				hosts[h.ID] = "offline"
				continue
			}
			for _, d := range domains {
				key := h.ID + "\x00" + d.UUID
				meta, hasMeta := metaByKey[key]
				isClaimed := hasMeta && meta.Claimed
				if isClaimed {
					displayName := d.Name
					if meta.DisplayName.Valid && meta.DisplayName.String != "" {
						displayName = meta.DisplayName.String
					}
					var consolePref, lastAccess *string
					if meta.ConsolePreference.Valid {
						cp := meta.ConsolePreference.String
						consolePref = &cp
					}
					if meta.LastAccess.Valid {
						la := meta.LastAccess.String
						lastAccess = &la
					}
					vms = append(vms, vmListItem{
						HostID:            h.ID,
						LibvirtUUID:       d.UUID,
						DisplayName:       &displayName,
						Claimed:           true,
						Status:            string(d.State),
						ConsolePreference: consolePref,
						LastAccess:        lastAccess,
						CreatedAt:         meta.CreatedAt,
						UpdatedAt:         meta.UpdatedAt,
					})
				} else {
					orphans = append(orphans, vmOrphanItem{
						HostID:      h.ID,
						LibvirtUUID: d.UUID,
						Name:        d.Name,
					})
				}
			}
		}
		writeJSON(w, http.StatusOK, vmListResponse{VMs: vms, Hosts: hosts, Orphans: orphans})
	}
}

type vmDetailResponse struct {
	HostID            string  `json:"host_id"`
	LibvirtUUID       string  `json:"libvirt_uuid"`
	DisplayName       *string `json:"display_name"`
	Claimed           bool    `json:"claimed"`
	Status            string  `json:"status"`
	ConsolePreference *string `json:"console_preference"`
	LastAccess        *string `json:"last_access"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
}

func (r *routerState) getVMDetail() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !r.configPresent || r.config == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "setup required")
			return
		}
		hostID := chi.URLParam(req, "host_id")
		libvirtUUID := chi.URLParam(req, "libvirt_uuid")
		if hostID == "" || libvirtUUID == "" {
			writeJSONError(w, http.StatusBadRequest, "host_id and libvirt_uuid required")
			return
		}
		meta, err := r.db.GetVMMetadata(req.Context(), hostID, libvirtUUID)
		if err != nil {
			r.logger.Error("get vm_metadata failed", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to get VM")
			return
		}
		if meta == nil || !meta.Claimed {
			writeJSONError(w, http.StatusNotFound, "VM not found")
			return
		}
		conn, err := r.getConnectorForHost(req.Context(), hostID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "host not found")
			return
		}
		defer conn.Close()
		domainInfo, err := conn.LookupByUUID(req.Context(), libvirtUUID)
		if err != nil {
			r.logger.Error("lookup domain failed", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			writeJSONError(w, http.StatusNotFound, "VM not found")
			return
		}
		if err := r.db.UpdateVMMetadataLastAccess(req.Context(), hostID, libvirtUUID); err != nil {
			r.logger.Error("update last_access failed", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
		}
		displayName := domainInfo.Name
		if meta.DisplayName.Valid && meta.DisplayName.String != "" {
			displayName = meta.DisplayName.String
		}
		var consolePref, lastAccess *string
		if meta.ConsolePreference.Valid {
			cp := meta.ConsolePreference.String
			consolePref = &cp
		}
		now := time.Now().UTC().Format(time.RFC3339)
		lastAccess = &now
		writeJSON(w, http.StatusOK, vmDetailResponse{
			HostID:            hostID,
			LibvirtUUID:       libvirtUUID,
			DisplayName:       &displayName,
			Claimed:           true,
			Status:            string(domainInfo.State),
			ConsolePreference: consolePref,
			LastAccess:        lastAccess,
			CreatedAt:         meta.CreatedAt,
			UpdatedAt:         now,
		})
	}
}

type patchVMConfigRequest struct {
	DisplayName       *string `json:"display_name"`
	ConsolePreference *string `json:"console_preference"`
	CPU               int     `json:"cpu"`
	RAMMB             int     `json:"ram_mb"`
	Network           string  `json:"network"`
}

func (r *routerState) patchVMConfig() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		user, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !r.configPresent || r.config == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "setup required")
			return
		}
		hostID := chi.URLParam(req, "host_id")
		libvirtUUID := chi.URLParam(req, "libvirt_uuid")
		if hostID == "" || libvirtUUID == "" {
			writeJSONError(w, http.StatusBadRequest, "host_id and libvirt_uuid required")
			return
		}
		var payload patchVMConfigRequest
		if err := decodeJSON(req.Body, &payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		meta, err := r.db.GetVMMetadata(req.Context(), hostID, libvirtUUID)
		if err != nil || meta == nil || !meta.Claimed {
			writeJSONError(w, http.StatusNotFound, "VM not found")
			return
		}
		conn, err := r.getConnectorForHost(req.Context(), hostID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "host not found")
			return
		}
		defer conn.Close()
		domainEdits := payload.CPU > 0 || payload.RAMMB > 0 || strings.TrimSpace(payload.Network) != ""
		if domainEdits {
			state, err := conn.GetState(req.Context(), libvirtUUID)
			if err != nil {
				writeJSONError(w, http.StatusNotFound, "VM not found")
				return
			}
			if state != libvirtconn.DomainStateShutoff {
				writeJSONError(w, http.StatusBadRequest, "VM must be stopped to change cpu, ram_mb, or network")
				return
			}
		}
		beforeMeta := map[string]interface{}{
			"display_name":       "",
			"console_preference": "",
		}
		if meta.DisplayName.Valid {
			beforeMeta["display_name"] = meta.DisplayName.String
		}
		if meta.ConsolePreference.Valid {
			beforeMeta["console_preference"] = meta.ConsolePreference.String
		}
		var beforeXML, afterXML string
		domainXMLChanged := false
		if domainEdits {
			beforeXML, err = conn.GetDomainXML(req.Context(), libvirtUUID)
			if err != nil {
				r.logger.Error("get domain XML failed", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "failed to update VM")
				return
			}
			if strings.TrimSpace(payload.Network) != "" {
				networks, err := conn.ListNetworks(req.Context())
				if err != nil {
					writeJSONError(w, http.StatusInternalServerError, "failed to list networks")
					return
				}
				found := false
				for _, n := range networks {
					if n.Name == payload.Network {
						found = true
						break
					}
				}
				if !found {
					writeJSONError(w, http.StatusConflict, "network invalid or does not exist on host")
					return
				}
			}
			var dom libvirtxml.Domain
			if err := dom.Unmarshal(beforeXML); err != nil {
				r.logger.Error("unmarshal domain XML failed", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "failed to update VM")
				return
			}
			if payload.CPU > 0 {
				if dom.VCPU == nil {
					dom.VCPU = &libvirtxml.DomainVCPU{}
				}
				dom.VCPU.Value = uint(payload.CPU)
				domainXMLChanged = true
			}
			if payload.RAMMB > 0 {
				kiB := uint(payload.RAMMB) * 1024
				if dom.Memory == nil {
					dom.Memory = &libvirtxml.DomainMemory{Unit: "KiB"}
				}
				dom.Memory.Value = kiB
				dom.Memory.Unit = "KiB"
				if dom.CurrentMemory != nil {
					dom.CurrentMemory.Value = kiB
					dom.CurrentMemory.Unit = "KiB"
				}
				domainXMLChanged = true
			}
			if strings.TrimSpace(payload.Network) != "" {
				if dom.Devices != nil {
					for i := range dom.Devices.Interfaces {
						if dom.Devices.Interfaces[i].Source != nil && dom.Devices.Interfaces[i].Source.Network != nil {
							dom.Devices.Interfaces[i].Source.Network.Network = payload.Network
							domainXMLChanged = true
							break
						}
					}
				}
			}
			if domainXMLChanged {
				afterXML, err = dom.Marshal()
				if err != nil {
					r.logger.Error("marshal domain XML failed", "error", err)
					writeJSONError(w, http.StatusInternalServerError, "failed to update VM")
					return
				}
				if _, err := conn.DefineXML(req.Context(), afterXML); err != nil {
					r.logger.Error("define domain XML failed", "error", err)
					writeJSONError(w, http.StatusInternalServerError, "failed to update VM")
					return
				}
			}
		}
		disp, cons := payload.DisplayName, payload.ConsolePreference
		if disp != nil {
			s := strings.TrimSpace(*disp)
			disp = &s
		}
		if cons != nil {
			s := strings.TrimSpace(*cons)
			cons = &s
		}
		if disp != nil || cons != nil {
			if err := r.db.UpdateVMMetadata(req.Context(), hostID, libvirtUUID, disp, cons); err != nil {
				r.logger.Error("update vm_metadata failed", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "failed to update VM")
				return
			}
		}
		meta, _ = r.db.GetVMMetadata(req.Context(), hostID, libvirtUUID)
		domainInfo, _ := conn.LookupByUUID(req.Context(), libvirtUUID)
		now := time.Now().UTC().Format(time.RFC3339)
		displayName := domainInfo.Name
		if meta != nil && meta.DisplayName.Valid && meta.DisplayName.String != "" {
			displayName = meta.DisplayName.String
		}
		var consolePref, lastAccess *string
		if meta != nil && meta.ConsolePreference.Valid {
			cp := meta.ConsolePreference.String
			consolePref = &cp
		}
		lastAccess = &now
		updatedAt := now
		if meta != nil {
			updatedAt = meta.UpdatedAt
		}
		if domainXMLChanged || disp != nil || cons != nil {
			afterMeta := map[string]interface{}{
				"display_name":       displayName,
				"console_preference": "",
			}
			if meta != nil && meta.ConsolePreference.Valid {
				afterMeta["console_preference"] = meta.ConsolePreference.String
			}
			diffContent := vmConfigChangeDiff(beforeMeta, afterMeta, beforeXML, afterXML, domainXMLChanged)
			ts := time.Now().UTC().Format(audit.TimestampFormat)
			diffPath := fmt.Sprintf("audit/vm/%s/%s/%s.diff", hostID, libvirtUUID, ts)
			userID := user.ID
			ev := audit.Event{
				EventType:  "vm_config_change",
				EntityType: "vm",
				EntityID:   libvirtUUID,
				UserID:     &userID,
				Payload: map[string]interface{}{
					"host_id": hostID,
					"changed": map[string]bool{
						"display_name":       disp != nil,
						"console_preference": cons != nil,
						"domain_xml":         domainXMLChanged,
					},
				},
			}
			if err := audit.RecordEventWithDiff(req.Context(), r.db, r.configuredGitPath(), ev, diffPath, diffContent); err != nil {
				r.logger.Error("audit vm_config_change failed", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "failed to record audit")
				return
			}
		}
		writeJSON(w, http.StatusOK, vmDetailResponse{
			HostID:            hostID,
			LibvirtUUID:       libvirtUUID,
			DisplayName:       &displayName,
			Claimed:           true,
			Status:            string(domainInfo.State),
			ConsolePreference: consolePref,
			LastAccess:        lastAccess,
			CreatedAt:         meta.CreatedAt,
			UpdatedAt:         updatedAt,
		})
	}
}

func vmConfigChangeDiff(beforeMeta, afterMeta map[string]interface{}, beforeXML, afterXML string, domainChanged bool) string {
	var sb strings.Builder
	sb.WriteString("--- vm_metadata (before)\n")
	sb.WriteString("+++ vm_metadata (after)\n")
	sb.WriteString("@@ -1,2 +1,2 @@\n")
	sb.WriteString(fmt.Sprintf("-display_name: %v\n", beforeMeta["display_name"]))
	sb.WriteString(fmt.Sprintf("-console_preference: %v\n", beforeMeta["console_preference"]))
	sb.WriteString(fmt.Sprintf("+display_name: %v\n", afterMeta["display_name"]))
	sb.WriteString(fmt.Sprintf("+console_preference: %v\n", afterMeta["console_preference"]))
	if domainChanged && beforeXML != "" && afterXML != "" {
		sb.WriteString("\n--- domain.xml (before)\n")
		sb.WriteString("+++ domain.xml (after)\n")
		sb.WriteString(unifiedDiffLines(beforeXML, afterXML))
	}
	return sb.String()
}

func unifiedDiffLines(before, after string) string {
	beforeLines := strings.Split(before, "\n")
	afterLines := strings.Split(after, "\n")
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("@@ -1,%d +1,%d @@\n", len(beforeLines), len(afterLines)))
	for _, l := range beforeLines {
		sb.WriteString("-")
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	for _, l := range afterLines {
		sb.WriteString("+")
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	return sb.String()
}

func (r *routerState) vmLifecycleOp(op string, fn func(conn libvirtconn.Connector, ctx context.Context, uuid string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !r.configPresent || r.config == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "setup required")
			return
		}
		hostID := chi.URLParam(req, "host_id")
		libvirtUUID := chi.URLParam(req, "libvirt_uuid")
		if hostID == "" || libvirtUUID == "" {
			writeJSONError(w, http.StatusBadRequest, "host_id and libvirt_uuid required")
			return
		}
		meta, err := r.db.GetVMMetadata(req.Context(), hostID, libvirtUUID)
		if err != nil || meta == nil || !meta.Claimed {
			writeJSONError(w, http.StatusNotFound, "VM not found")
			return
		}
		conn, err := r.getConnectorForHost(req.Context(), hostID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "host not found")
			return
		}
		defer conn.Close()
		if err := fn(conn, req.Context(), libvirtUUID); err != nil {
			r.logger.Error(op+" failed", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			writeJSONError(w, http.StatusInternalServerError, op+" failed")
			return
		}
		state, _ := conn.GetState(req.Context(), libvirtUUID)
		writeJSON(w, http.StatusOK, map[string]string{"status": string(state)})
	}
}

func (r *routerState) vmStart() http.HandlerFunc {
	return r.vmLifecycleOp("start", func(conn libvirtconn.Connector, ctx context.Context, uuid string) error {
		return conn.Create(ctx, uuid)
	})
}

func (r *routerState) vmPause() http.HandlerFunc {
	return r.vmLifecycleOp("pause", func(conn libvirtconn.Connector, ctx context.Context, uuid string) error {
		return conn.Suspend(ctx, uuid)
	})
}

func (r *routerState) vmResume() http.HandlerFunc {
	return r.vmLifecycleOp("resume", func(conn libvirtconn.Connector, ctx context.Context, uuid string) error {
		return conn.Resume(ctx, uuid)
	})
}

func (r *routerState) vmDestroy() http.HandlerFunc {
	return r.vmLifecycleOp("destroy", func(conn libvirtconn.Connector, ctx context.Context, uuid string) error {
		return conn.Destroy(ctx, uuid)
	})
}

func (r *routerState) vmStop() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !r.configPresent || r.config == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "setup required")
			return
		}
		hostID := chi.URLParam(req, "host_id")
		libvirtUUID := chi.URLParam(req, "libvirt_uuid")
		if hostID == "" || libvirtUUID == "" {
			writeJSONError(w, http.StatusBadRequest, "host_id and libvirt_uuid required")
			return
		}
		meta, err := r.db.GetVMMetadata(req.Context(), hostID, libvirtUUID)
		if err != nil || meta == nil || !meta.Claimed {
			writeJSONError(w, http.StatusNotFound, "VM not found")
			return
		}
		conn, err := r.getConnectorForHost(req.Context(), hostID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "host not found")
			return
		}
		defer conn.Close()
		timeout := 30 * time.Second
		if r.config != nil && r.config.VMLifecycle.GracefulStopTimeout > 0 {
			timeout = time.Duration(r.config.VMLifecycle.GracefulStopTimeout)
		}
		if err := conn.Shutdown(req.Context(), libvirtUUID); err != nil {
			r.logger.Error("stop failed", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "stop failed")
			return
		}
		deadline := time.Now().Add(timeout)
		for time.Now().Before(deadline) {
			state, err := conn.GetState(req.Context(), libvirtUUID)
			if err != nil {
				break
			}
			if state == libvirtconn.DomainStateShutoff {
				writeJSON(w, http.StatusOK, map[string]string{"status": string(state)})
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		if err := conn.Destroy(req.Context(), libvirtUUID); err != nil {
			r.logger.Error("force stop failed", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "stop failed")
			return
		}
		state, _ := conn.GetState(req.Context(), libvirtUUID)
		writeJSON(w, http.StatusOK, map[string]string{"status": string(state)})
	}
}

type vmClaimRequest struct {
	DisplayName *string `json:"display_name"`
}

func (r *routerState) vmClaim() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !r.configPresent || r.config == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "setup required")
			return
		}
		hostID := chi.URLParam(req, "host_id")
		libvirtUUID := chi.URLParam(req, "libvirt_uuid")
		if hostID == "" || libvirtUUID == "" {
			writeJSONError(w, http.StatusBadRequest, "host_id and libvirt_uuid required")
			return
		}
		conn, err := r.getConnectorForHost(req.Context(), hostID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "host not found")
			return
		}
		defer conn.Close()
		domainInfo, err := conn.LookupByUUID(req.Context(), libvirtUUID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "VM not found")
			return
		}
		displayName := domainInfo.Name
		var payload vmClaimRequest
		_ = decodeJSON(req.Body, &payload)
		if payload.DisplayName != nil && strings.TrimSpace(*payload.DisplayName) != "" {
			displayName = strings.TrimSpace(*payload.DisplayName)
		}
		if err := r.db.UpsertVMMetadataClaim(req.Context(), hostID, libvirtUUID, displayName); err != nil {
			r.logger.Error("claim failed", "host_id", hostID, "libvirt_uuid", libvirtUUID, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "claim failed")
			return
		}
		meta, _ := r.db.GetVMMetadata(req.Context(), hostID, libvirtUUID)
		createdAt, updatedAt := "", ""
		if meta != nil {
			createdAt, updatedAt = meta.CreatedAt, meta.UpdatedAt
		}
		writeJSON(w, http.StatusOK, vmDetailResponse{
			HostID:      hostID,
			LibvirtUUID: libvirtUUID,
			DisplayName: &displayName,
			Claimed:     true,
			Status:      string(domainInfo.State),
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})
	}
}

type createVMDisk struct {
	Name   string `json:"name"`
	SizeMB int    `json:"size_mb"`
}

type createVMRequest struct {
	HostID      string        `json:"host_id"`
	Pool        string        `json:"pool"`
	Disk        createVMDisk  `json:"disk"`
	CPU         int           `json:"cpu"`
	RAMMB       int           `json:"ram_mb"`
	Network     string        `json:"network"`
	DisplayName string        `json:"display_name"`
}

type createVMResponse struct {
	HostID      string `json:"host_id"`
	LibvirtUUID string `json:"libvirt_uuid"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
	Status      string `json:"status"`
}

func (r *routerState) createVM() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !r.configPresent || r.config == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "setup required")
			return
		}
		var payload createVMRequest
		if err := decodeJSON(req.Body, &payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		hostID := strings.TrimSpace(payload.HostID)
		if hostID == "" && r.config.DefaultHost != "" {
			hostID = r.config.DefaultHost
		}
		if hostID == "" {
			writeJSONError(w, http.StatusBadRequest, "host_id required")
			return
		}
		pool := strings.TrimSpace(payload.Pool)
		if pool == "" && r.config.DefaultPool != nil {
			pool = strings.TrimSpace(*r.config.DefaultPool)
		}
		if pool == "" {
			writeJSONError(w, http.StatusBadRequest, "pool required")
			return
		}
		cpu := payload.CPU
		if cpu <= 0 {
			cpu = r.config.VMDefaults.CPU
		}
		if cpu <= 0 {
			cpu = 2
		}
		ramMB := payload.RAMMB
		if ramMB <= 0 {
			ramMB = r.config.VMDefaults.RAMMB
		}
		if ramMB <= 0 {
			ramMB = 2048
		}
		network := strings.TrimSpace(payload.Network)
		if network == "" {
			network = r.config.VMDefaults.Network
		}
		if network == "" {
			network = "default"
		}
		conn, err := r.getConnectorForHost(req.Context(), hostID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "host not found")
			return
		}
		defer conn.Close()
		if err := conn.ValidatePool(req.Context(), pool); err != nil {
			r.logger.Error("validate pool failed", "host_id", hostID, "pool", pool, "error", err)
			writeJSONError(w, http.StatusBadRequest, "pool invalid or inactive")
			return
		}
		var diskPath string
		existingName := strings.TrimSpace(payload.Disk.Name)
		sizeMB := payload.Disk.SizeMB
		if existingName != "" {
			if err := conn.ValidateVolume(req.Context(), pool, existingName); err != nil {
				writeJSONError(w, http.StatusBadRequest, "volume not found")
				return
			}
			vols, err := conn.ListVolumes(req.Context(), pool)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "failed to list volumes")
				return
			}
			for _, v := range vols {
				if v.Name == existingName {
					diskPath = v.Path
					break
				}
			}
		} else if sizeMB > 0 {
			u, _ := randomUUID()
			volName := fmt.Sprintf("kui-%s.qcow2", u[:8])
			volXML := fmt.Sprintf(`<volume><name>%s</name><capacity unit="bytes">%d</capacity><target><format type="qcow2"/></target></volume>`, volName, uint64(sizeMB)*1024*1024)
			vol, err := conn.CreateVolumeFromXML(req.Context(), pool, volXML)
			if err != nil {
				r.logger.Error("create volume failed", "host_id", hostID, "pool", pool, "error", err)
				writeJSONError(w, http.StatusInternalServerError, "failed to create disk")
				return
			}
			diskPath = vol.Path
		} else {
			writeJSONError(w, http.StatusBadRequest, "disk name or size_mb required")
			return
		}
		vmUUID, err := randomUUID()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to generate UUID")
			return
		}
		domainName := fmt.Sprintf("kui-%s", vmUUID[:8])
		displayName := strings.TrimSpace(payload.DisplayName)
		if displayName == "" {
			displayName = domainName
		}
		domainXML := fmt.Sprintf(`<domain type="kvm">
  <name>%s</name>
  <uuid>%s</uuid>
  <memory unit="KiB">%d</memory>
  <vcpu>%d</vcpu>
  <os><type arch="x86_64" machine="pc">hvm</type><boot dev="hd"/></os>
  <devices>
    <disk type="file" device="disk">
      <driver name="qemu" type="qcow2"/>
      <source file="%s"/>
      <target dev="vda" bus="virtio"/>
    </disk>
    <interface type="network"><source network="%s"/><model type="virtio"/></interface>
  </devices>
</domain>`, domainName, vmUUID, ramMB*1024, cpu, diskPath, network)
		domainInfo, err := conn.DefineXML(req.Context(), domainXML)
		if err != nil {
			r.logger.Error("define domain failed", "host_id", hostID, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create VM")
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		if err := r.db.InsertVMMetadata(req.Context(), hostID, domainInfo.UUID, true, &displayName); err != nil {
			r.logger.Error("insert vm_metadata failed", "host_id", hostID, "libvirt_uuid", domainInfo.UUID, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create VM")
			return
		}
		writeJSON(w, http.StatusCreated, createVMResponse{
			HostID:      hostID,
			LibvirtUUID: domainInfo.UUID,
			DisplayName: displayName,
			CreatedAt:   now,
			Status:      string(domainInfo.State),
		})
	}
}

func (r *routerState) getTemplates() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !r.configPresent || r.config == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "setup required")
			return
		}
		gitBase := r.configuredGitPath()
		if gitBase == "" {
			writeJSONError(w, http.StatusServiceUnavailable, "git path not configured")
			return
		}
		list, err := template.ListTemplates(gitBase)
		if err != nil {
			r.logger.Error("list templates failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to list templates")
			return
		}
		defaultCPU, defaultRAM, defaultNet := 2, 2048, "default"
		if r.config != nil {
			if r.config.VMDefaults.CPU > 0 {
				defaultCPU = r.config.VMDefaults.CPU
			}
			if r.config.VMDefaults.RAMMB > 0 {
				defaultRAM = r.config.VMDefaults.RAMMB
			}
			if r.config.VMDefaults.Network != "" {
				defaultNet = r.config.VMDefaults.Network
			}
		}
		out := make([]templateListItem, 0, len(list))
		for _, t := range list {
			cpu, ram, net := t.CPU, t.RAMMB, t.Network
			if cpu <= 0 {
				cpu = defaultCPU
			}
			if ram <= 0 {
				ram = defaultRAM
			}
			if net == "" {
				net = defaultNet
			}
			out = append(out, templateListItem{
				TemplateID: t.TemplateID,
				Name:       t.Name,
				BaseImage:  t.BaseImage,
				CPU:        cpu,
				RAMMB:      ram,
				Network:    net,
				CreatedAt:  t.CreatedAt,
			})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

type templateListItem struct {
	TemplateID string             `json:"template_id"`
	Name       string             `json:"name"`
	BaseImage  template.BaseImage `json:"base_image"`
	CPU        int                `json:"cpu"`
	RAMMB      int                `json:"ram_mb"`
	Network    string             `json:"network"`
	CreatedAt  string             `json:"created_at"`
}

type createTemplateRequest struct {
	SourceHostID    string `json:"source_host_id"`
	SourceLibvirtUUID string `json:"source_libvirt_uuid"`
	Name            string `json:"name"`
	TargetPool      string `json:"target_pool"`
}

type createTemplateResponse struct {
	TemplateID string             `json:"template_id"`
	Name       string             `json:"name"`
	BaseImage  template.BaseImage `json:"base_image"`
	CreatedAt  string             `json:"created_at"`
}

func (r *routerState) createTemplate() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		user, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !r.configPresent || r.config == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "setup required")
			return
		}
		gitBase := r.configuredGitPath()
		if gitBase == "" {
			writeJSONError(w, http.StatusServiceUnavailable, "git path not configured")
			return
		}
		var payload createTemplateRequest
		if err := decodeJSON(req.Body, &payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		sourceHostID := strings.TrimSpace(payload.SourceHostID)
		sourceUUID := strings.TrimSpace(payload.SourceLibvirtUUID)
		name := strings.TrimSpace(payload.Name)
		if sourceHostID == "" || sourceUUID == "" || name == "" {
			writeJSONError(w, http.StatusBadRequest, "source_host_id, source_libvirt_uuid, and name are required")
			return
		}
		templateID := template.Slugify(name)
		if templateID == "" {
			writeJSONError(w, http.StatusBadRequest, "name must produce a valid template_id")
			return
		}
		exists, err := template.TemplateExists(gitBase, templateID)
		if err != nil {
			r.logger.Error("check template exists failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create template")
			return
		}
		if exists {
			writeJSONError(w, http.StatusConflict, "template already exists")
			return
		}
		conn, err := r.getConnectorForHost(req.Context(), sourceHostID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "host not found")
			return
		}
		defer conn.Close()
		state, err := conn.GetState(req.Context(), sourceUUID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "source VM not found")
			return
		}
		if state != libvirtconn.DomainStateShutoff {
			writeJSONError(w, http.StatusConflict, "source VM must be stopped to save as template")
			return
		}
		domainXML, err := conn.GetDomainXML(req.Context(), sourceUUID)
		if err != nil {
			r.logger.Error("get domain XML failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create template")
			return
		}
		diskPath := extractFirstDiskPath(domainXML)
		if diskPath == "" {
			writeJSONError(w, http.StatusBadRequest, "could not determine source disk")
			return
		}
		pools, err := conn.ListPools(req.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list pools")
			return
		}
		var sourceVolName, sourcePoolName string
		for _, p := range pools {
			vols, err := conn.ListVolumes(req.Context(), p.Name)
			if err != nil {
				continue
			}
			for _, v := range vols {
				if v.Path == diskPath {
					sourceVolName = v.Name
					sourcePoolName = p.Name
					break
				}
			}
			if sourceVolName != "" {
				break
			}
		}
		if sourceVolName == "" {
			writeJSONError(w, http.StatusBadRequest, "source disk not found in any pool")
			return
		}
		targetPool := strings.TrimSpace(payload.TargetPool)
		if targetPool == "" && r.config.TemplateStorage != nil {
			targetPool = strings.TrimSpace(*r.config.TemplateStorage)
		}
		if targetPool == "" {
			targetPool = sourcePoolName
		}
		if err := conn.ValidatePool(req.Context(), targetPool); err != nil {
			r.logger.Error("validate target pool failed", "pool", targetPool, "error", err)
			writeJSONError(w, http.StatusServiceUnavailable, "target pool invalid or inactive")
			return
		}
		domainInfo, _ := conn.LookupByUUID(req.Context(), sourceUUID)
		diskName := domainInfo.Name + ".qcow2"
		var targetDiskPath string
		if sourcePoolName == targetPool {
			if err := conn.CloneVolume(req.Context(), targetPool, sourceVolName, diskName); err != nil {
				r.logger.Error("clone volume failed", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "failed to copy disk")
				return
			}
			vols, _ := conn.ListVolumes(req.Context(), targetPool)
			for _, v := range vols {
				if v.Name == diskName {
					targetDiskPath = v.Path
					break
				}
			}
		} else {
			data, err := conn.CopyVolume(req.Context(), sourcePoolName, sourceVolName)
			if err != nil {
				r.logger.Error("copy volume failed", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "failed to copy disk")
				return
			}
			vol, err := conn.CreateVolumeFromBytes(req.Context(), targetPool, diskName, data, "qcow2")
			if err != nil {
				r.logger.Error("create volume failed", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "failed to create template disk")
				return
			}
			targetDiskPath = vol.Path
		}
		if targetDiskPath == "" {
			writeJSONError(w, http.StatusInternalServerError, "failed to resolve disk path")
			return
		}
		var dom libvirtxml.Domain
		if err := dom.Unmarshal(domainXML); err != nil {
			r.logger.Error("unmarshal domain XML failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create template")
			return
		}
		dom.UUID = ""
		dom.Name = "template-" + templateID
		if dom.Devices != nil {
			for i := range dom.Devices.Disks {
				d := &dom.Devices.Disks[i]
				if d.Source != nil {
					if d.Source.File != nil {
						d.Source.File.File = targetDiskPath
					} else {
						d.Source.File = &libvirtxml.DomainDiskSourceFile{File: targetDiskPath}
					}
					break
				}
			}
		}
		templateDomainXML, err := dom.Marshal()
		if err != nil {
			r.logger.Error("marshal domain XML failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create template")
			return
		}
		defaultCPU, defaultRAM, defaultNet := 2, 2048, "default"
		if r.config.VMDefaults.CPU > 0 {
			defaultCPU = r.config.VMDefaults.CPU
		}
		if r.config.VMDefaults.RAMMB > 0 {
			defaultRAM = r.config.VMDefaults.RAMMB
		}
		if r.config.VMDefaults.Network != "" {
			defaultNet = r.config.VMDefaults.Network
		}
		meta := &template.Meta{
			Name:      name,
			BaseImage: template.BaseImage{Pool: targetPool, Volume: diskName},
			CPU:       defaultCPU,
			RAMMB:     defaultRAM,
			Network:   defaultNet,
		}
		templateDir, err := template.CreateTemplateDir(gitBase, templateID)
		if err != nil {
			r.logger.Error("create template dir failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create template")
			return
		}
		if err := template.WriteMeta(filepath.Join(templateDir, "meta.yaml"), meta); err != nil {
			r.logger.Error("write meta failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create template")
			return
		}
		if err := os.WriteFile(filepath.Join(templateDir, "domain.xml"), []byte(templateDomainXML), 0o644); err != nil {
			r.logger.Error("write domain.xml failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create template")
			return
		}
		templateMetaPath := filepath.ToSlash(filepath.Join("templates", templateID, "meta.yaml"))
		templateDomainPath := filepath.ToSlash(filepath.Join("templates", templateID, "domain.xml"))
		if _, err := audit.CommitPaths(gitBase, []string{templateMetaPath, templateDomainPath}, "template: create "+templateID); err != nil {
			r.logger.Error("commit template files failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create template")
			return
		}
		diffContent := "--- /dev/null\n+++ templates/" + templateID + "/meta.yaml\n" + templateDiffLines("", metaYAMLString(meta)) + "\n--- /dev/null\n+++ templates/" + templateID + "/domain.xml\n" + templateDiffLines("", templateDomainXML)
		ts := time.Now().UTC().Format(audit.TimestampFormat)
		diffPath := fmt.Sprintf("audit/template/%s/%s.diff", templateID, ts)
		userID := user.ID
		ev := audit.Event{
			EventType:  "template_create",
			EntityType: "template",
			EntityID:   templateID,
			UserID:     &userID,
			Payload: map[string]interface{}{
				"source_host_id": sourceHostID,
				"source_libvirt_uuid": sourceUUID,
				"name": name,
			},
		}
		if err := audit.RecordEventWithDiff(req.Context(), r.db, gitBase, ev, diffPath, diffContent); err != nil {
			r.logger.Error("audit template_create failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to record audit")
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		writeJSON(w, http.StatusCreated, createTemplateResponse{
			TemplateID: templateID,
			Name:       name,
			BaseImage:  meta.BaseImage,
			CreatedAt:  now,
		})
	}
}

func metaYAMLString(m *template.Meta) string {
	// Minimal YAML for diff
	var bi string
	if m.BaseImage.Path != "" {
		bi = fmt.Sprintf("  path: %s", m.BaseImage.Path)
	} else {
		bi = fmt.Sprintf("  volume: %s", m.BaseImage.Volume)
	}
	return fmt.Sprintf("name: %s\nbase_image:\n  pool: %s\n%s\ncpu: %d\nram_mb: %d\nnetwork: %s",
		m.Name, m.BaseImage.Pool, bi, m.CPU, m.RAMMB, m.Network)
}

func templateDiffLines(before, after string) string {
	afterLines := strings.Split(after, "\n")
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(afterLines)))
	for _, l := range afterLines {
		sb.WriteString("+")
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	return sb.String()
}

var diskSourceFileRe = regexp.MustCompile(`<source\s+file=['"]([^'"]+)['"]`)

func extractFirstDiskPath(domainXML string) string {
	m := diskSourceFileRe.FindStringSubmatch(domainXML)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

type vmCloneRequest struct {
	TargetHostID string `json:"target_host_id"`
	TargetPool  string `json:"target_pool"`
	TargetName  string `json:"target_name"`
}

func (r *routerState) vmClone() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !r.configPresent || r.config == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "setup required")
			return
		}
		sourceHostID := chi.URLParam(req, "host_id")
		sourceUUID := chi.URLParam(req, "libvirt_uuid")
		if sourceHostID == "" || sourceUUID == "" {
			writeJSONError(w, http.StatusBadRequest, "host_id and libvirt_uuid required")
			return
		}
		var payload vmCloneRequest
		if err := decodeJSON(req.Body, &payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		targetHostID := strings.TrimSpace(payload.TargetHostID)
		targetPool := strings.TrimSpace(payload.TargetPool)
		if targetHostID == "" || targetPool == "" {
			writeJSONError(w, http.StatusBadRequest, "target_host_id and target_pool required")
			return
		}
		sourceConn, err := r.getConnectorForHost(req.Context(), sourceHostID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "host not found")
			return
		}
		defer sourceConn.Close()
		state, err := sourceConn.GetState(req.Context(), sourceUUID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "source VM not found")
			return
		}
		if state != libvirtconn.DomainStateShutoff {
			writeJSONError(w, http.StatusBadRequest, "source VM must be stopped to clone")
			return
		}
		domainXML, err := sourceConn.GetDomainXML(req.Context(), sourceUUID)
		if err != nil {
			r.logger.Error("get domain XML failed", "host_id", sourceHostID, "libvirt_uuid", sourceUUID, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to clone")
			return
		}
		diskPath := extractFirstDiskPath(domainXML)
		if diskPath == "" {
			writeJSONError(w, http.StatusBadRequest, "could not determine source disk")
			return
		}
		sourceInfo, _ := sourceConn.LookupByUUID(req.Context(), sourceUUID)
		cloneName := strings.TrimSpace(payload.TargetName)
		if cloneName == "" {
			cloneName = sourceInfo.Name + "-clone"
		}
		pools, err := sourceConn.ListPools(req.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to list pools")
			return
		}
		var sourceVolName, sourcePoolName string
		for _, p := range pools {
			vols, err := sourceConn.ListVolumes(req.Context(), p.Name)
			if err != nil {
				continue
			}
			for _, v := range vols {
				if v.Path == diskPath {
					sourceVolName = v.Name
					sourcePoolName = p.Name
					break
				}
			}
			if sourceVolName != "" {
				break
			}
		}
		if sourceVolName == "" {
			writeJSONError(w, http.StatusBadRequest, "source disk not found in any pool")
			return
		}
		targetConn, err := r.getConnectorForHost(req.Context(), targetHostID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "target host not found")
			return
		}
		defer targetConn.Close()
		if err := targetConn.ValidatePool(req.Context(), targetPool); err != nil {
			writeJSONError(w, http.StatusBadRequest, "target pool invalid or inactive")
			return
		}
		targetVolName := cloneName + ".qcow2"
		var targetDiskPath string
		if sourceHostID == targetHostID && sourcePoolName == targetPool {
			if err := sourceConn.CloneVolume(req.Context(), targetPool, sourceVolName, targetVolName); err != nil {
				r.logger.Error("clone volume failed", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "failed to clone disk")
				return
			}
			vols, _ := sourceConn.ListVolumes(req.Context(), targetPool)
			for _, v := range vols {
				if v.Name == targetVolName {
					targetDiskPath = v.Path
					break
				}
			}
		} else {
			data, err := sourceConn.CopyVolume(req.Context(), sourcePoolName, sourceVolName)
			if err != nil {
				r.logger.Error("copy volume failed", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "failed to copy disk")
				return
			}
			vol, err := targetConn.CreateVolumeFromBytes(req.Context(), targetPool, targetVolName, data, "qcow2")
			if err != nil {
				r.logger.Error("create volume from bytes failed", "error", err)
				writeJSONError(w, http.StatusInternalServerError, "failed to create cloned disk")
				return
			}
			targetDiskPath = vol.Path
		}
		if targetDiskPath == "" {
			writeJSONError(w, http.StatusInternalServerError, "failed to resolve cloned disk path")
			return
		}
		vmUUID, err := randomUUID()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to generate UUID")
			return
		}
		domainName := cloneName
		domainXMLNew := fmt.Sprintf(`<domain type="kvm">
  <name>%s</name>
  <uuid>%s</uuid>
  <memory unit="KiB">2097152</memory>
  <vcpu>2</vcpu>
  <os><type arch="x86_64" machine="pc">hvm</type><boot dev="hd"/></os>
  <devices>
    <disk type="file" device="disk">
      <driver name="qemu" type="qcow2"/>
      <source file="%s"/>
      <target dev="vda" bus="virtio"/>
    </disk>
    <interface type="network"><source network="default"/><model type="virtio"/></interface>
  </devices>
</domain>`, domainName, vmUUID, targetDiskPath)
		domainInfo, err := targetConn.DefineXML(req.Context(), domainXMLNew)
		if err != nil {
			r.logger.Error("define clone domain failed", "target_host_id", targetHostID, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create cloned VM")
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)
		if err := r.db.InsertVMMetadata(req.Context(), targetHostID, domainInfo.UUID, true, &cloneName); err != nil {
			r.logger.Error("insert vm_metadata for clone failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to create cloned VM")
			return
		}
		writeJSON(w, http.StatusCreated, createVMResponse{
			HostID:      targetHostID,
			LibvirtUUID: domainInfo.UUID,
			DisplayName: cloneName,
			CreatedAt:   now,
			Status:      string(domainInfo.State),
		})
	}
}

type poolResponse struct {
	Name  string `json:"name"`
	UUID  string `json:"uuid"`
	State string `json:"state"`
}

type volumeResponse struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Capacity uint64 `json:"capacity"`
}

type networkResponse struct {
	Name   string `json:"name"`
	UUID   string `json:"uuid"`
	Active bool   `json:"active"`
}

func (r *routerState) getConnectorForHost(ctx context.Context, hostID string) (libvirtconn.Connector, error) {
	if r.config == nil || len(r.config.Hosts) == 0 {
		return nil, errors.New("no hosts configured")
	}
	for _, h := range r.config.Hosts {
		if h.ID == hostID {
			keyfile := ""
			if h.Keyfile != nil {
				keyfile = *h.Keyfile
			}
			return libvirtconn.Connect(ctx, h.URI, keyfile)
		}
	}
	return nil, errors.New("host not found")
}

func (r *routerState) getHostPools() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !r.configPresent || r.config == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "setup required")
			return
		}
		hostID := chi.URLParam(req, "host_id")
		if hostID == "" {
			writeJSONError(w, http.StatusBadRequest, "host_id required")
			return
		}
		conn, err := r.getConnectorForHost(req.Context(), hostID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "host not found")
			return
		}
		defer conn.Close()
		pools, err := conn.ListPools(req.Context())
		if err != nil {
			r.logger.Error("list pools failed", "host_id", hostID, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to list pools")
			return
		}
		out := make([]poolResponse, 0, len(pools))
		for _, p := range pools {
			state := "inactive"
			if p.Active {
				state = "active"
			}
			out = append(out, poolResponse{Name: p.Name, UUID: p.UUID, State: state})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func (r *routerState) getHostPoolVolumes() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !r.configPresent || r.config == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "setup required")
			return
		}
		hostID := chi.URLParam(req, "host_id")
		poolName := chi.URLParam(req, "pool_name")
		if hostID == "" || poolName == "" {
			writeJSONError(w, http.StatusBadRequest, "host_id and pool_name required")
			return
		}
		conn, err := r.getConnectorForHost(req.Context(), hostID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "host not found")
			return
		}
		defer conn.Close()
		volumes, err := conn.ListVolumes(req.Context(), poolName)
		if err != nil {
			r.logger.Error("list volumes failed", "host_id", hostID, "pool", poolName, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to list volumes")
			return
		}
		out := make([]volumeResponse, 0, len(volumes))
		for _, v := range volumes {
			out = append(out, volumeResponse{Name: v.Name, Path: v.Path, Capacity: v.Capacity})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func (r *routerState) getHostNetworks() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		_, ok := mw.UserFromContext(req)
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if !r.configPresent || r.config == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "setup required")
			return
		}
		hostID := chi.URLParam(req, "host_id")
		if hostID == "" {
			writeJSONError(w, http.StatusBadRequest, "host_id required")
			return
		}
		conn, err := r.getConnectorForHost(req.Context(), hostID)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "host not found")
			return
		}
		defer conn.Close()
		networks, err := conn.ListNetworks(req.Context())
		if err != nil {
			r.logger.Error("list networks failed", "host_id", hostID, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to list networks")
			return
		}
		out := make([]networkResponse, 0, len(networks))
		for _, n := range networks {
			out = append(out, networkResponse{Name: n.Name, UUID: n.UUID, Active: n.Active})
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
			r.logger.Debug("validate-host failed", "host_id", payload.HostID)
			writeJSON(w, http.StatusOK, validateHostResponse{
				Valid: false,
				Error: "validation failed",
			})
			return
		}
		if err := conn.Close(); err != nil {
			r.logger.Debug("validate-host close failed", "host_id", payload.HostID)
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
		if _, statErr := os.Stat(r.configPath); statErr == nil {
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

		configYAML, err := yaml.Marshal(persist)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to prepare audit")
			return
		}
		ts := time.Now().UTC().Format(audit.TimestampFormat)
		diffPath := "audit/wizard/" + ts + ".diff"
		diffContent := audit.WizardDiff(string(configYAML))
		ev := audit.Event{
			EventType:  "wizard_complete",
			EntityType: "wizard",
			EntityID:   "latest",
			UserID:     &userID,
			Payload: map[string]interface{}{
				"action":          "wizard_complete",
				"result":          "success",
				"admin_username": strings.TrimSpace(payload.Admin.Username),
				"host_id":         payload.DefaultHost,
				"config_path":     r.configPath,
				"git_path":        r.configuredGitPath(),
			},
		}
		if err := audit.RecordEventWithDiff(req.Context(), r.db, r.configuredGitPath(), ev, diffPath, diffContent); err != nil {
			r.logger.Error("audit wizard_complete failed", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "failed to record audit")
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

