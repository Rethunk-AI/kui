package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kui/kui/internal/config"
	"github.com/kui/kui/internal/db"
	"github.com/kui/kui/internal/git"
	"github.com/kui/kui/internal/libvirtconn"
	"github.com/kui/kui/internal/prefix"
	"github.com/kui/kui/internal/routes"
)

const defaultListenAddress = ":8080"

type trackedFlag struct {
	value string
	set   bool
}

func (f *trackedFlag) String() string {
	return f.value
}

func (f *trackedFlag) Set(value string) error {
	f.value = value
	f.set = true
	return nil
}

type startupOptions struct {
	configPath      string
	configSource    string // "--config", "env", or "default"
	bootstrapPrefix string // trimmed effective prefix (--prefix or KUI_PREFIX)
	listen          string
	tlsCert         string
	tlsKey          string
}

type application struct {
	logger       *slog.Logger
	configPath   string
	config       *config.Config
	configExists bool
	db           *db.DB
	server       *http.Server
	serveErr     chan error
	listener     net.Listener
	dbPath       string
	gitPath      string
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(os.Args[1:], logger); err != nil {
		fatalStartup(err)
	}
}

func fatalStartup(err error) {
	payload := map[string]string{"error": err.Error()}
	enc := json.NewEncoder(os.Stderr)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(payload)
	os.Exit(1)
}

func run(args []string, logger *slog.Logger) error {
	opts, err := parseFlags(args)
	if err != nil {
		return err
	}

	app, err := buildApplication(opts, logger)
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = app.shutdown(ctx)
	}()

	tlsCert, tlsKey := resolveTLSCertKey(opts.bootstrapPrefix, opts.tlsCert, opts.tlsKey)
	listener, err := app.startServer(opts.listen, tlsCert, tlsKey)
	if err != nil {
		return err
	}
	logger.Info("KUI listening on", "addr", listener.Addr().String())
	if tlsCert != "" {
		logger.Info("TLS enabled", "cert", tlsCert, "key", tlsKey)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-app.serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := app.shutdown(shutdownCtx); err != nil {
		return err
	}
	logger.Info("KUI shutdown complete")
	return nil
}

func parseFlags(args []string) (startupOptions, error) {
	var options startupOptions

	configPathFlag := &trackedFlag{value: config.DefaultConfigPath}
	prefixFlag := &trackedFlag{}
	listenFlag := &trackedFlag{value: defaultListenAddress}
	tlsCertFlag := &trackedFlag{}
	tlsKeyFlag := &trackedFlag{}

	flagSet := flag.NewFlagSet("kui", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	flagSet.Var(configPathFlag, "config", "path to yaml config file")
	flagSet.Var(prefixFlag, "prefix", "runtime filesystem root (chroot-style path resolution)")
	flagSet.Var(listenFlag, "listen", "listen address")
	flagSet.Var(tlsCertFlag, "tls-cert", "path to TLS cert")
	flagSet.Var(tlsKeyFlag, "tls-key", "path to TLS key")
	if err := flagSet.Parse(args); err != nil {
		return startupOptions{}, err
	}

	configPath := strings.TrimSpace(configPathFlag.value)
	listen := strings.TrimSpace(listenFlag.value)
	tlsCert := strings.TrimSpace(tlsCertFlag.value)
	tlsKey := strings.TrimSpace(tlsKeyFlag.value)

	configSource := "default"
	if configPathFlag.set {
		configSource = "--config"
	} else if strings.TrimSpace(os.Getenv("KUI_CONFIG")) != "" {
		configSource = "env"
	}
	if envConfig := strings.TrimSpace(os.Getenv("KUI_CONFIG")); envConfig != "" && !configPathFlag.set {
		configPath = envConfig
	}
	if envListen := strings.TrimSpace(os.Getenv("KUI_LISTEN")); envListen != "" && !listenFlag.set {
		listen = envListen
	}

	bootstrapPrefix := strings.TrimSpace(prefixFlag.value)
	if !prefixFlag.set {
		bootstrapPrefix = strings.TrimSpace(os.Getenv("KUI_PREFIX"))
	}

	if listen == "" {
		return startupOptions{}, fmt.Errorf("listen address is required")
	}
	if tlsCert == "" && tlsKey != "" || tlsCert != "" && tlsKey == "" {
		return startupOptions{}, fmt.Errorf("both --tls-cert and --tls-key must be provided together")
	}

	options = startupOptions{
		configPath:      configPath,
		configSource:    configSource,
		bootstrapPrefix: bootstrapPrefix,
		listen:          listen,
		tlsCert:         tlsCert,
		tlsKey:          tlsKey,
	}
	return options, nil
}

func buildApplication(opts startupOptions, logger *slog.Logger) (*application, error) {
	bootstrap := strings.TrimSpace(opts.bootstrapPrefix)
	if bootstrap != "" {
		st, err := os.Stat(bootstrap)
		if err != nil {
			return nil, fmt.Errorf("prefix %q: %w", bootstrap, err)
		}
		if !st.IsDir() {
			return nil, fmt.Errorf("prefix %q is not a directory", bootstrap)
		}
	}

	candidateCfgPath := strings.TrimSpace(opts.configPath)
	if candidateCfgPath == "" {
		candidateCfgPath = config.DefaultConfigPath
	}
	// Path used for Stat and ReadFile; with bootstrap set this is prefix.Resolve(bootstrap, candidate).
	cfgPath := candidateCfgPath
	if bootstrap != "" {
		cfgPath = prefix.Resolve(bootstrap, candidateCfgPath)
	}

	configExists := false
	if statErr := getFileStatError(cfgPath); statErr == nil {
		configExists = true
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, fmt.Errorf("read config path %q: %w", cfgPath, statErr)
	}

	var appConfig *config.Config
	var dbPath string
	var gitPath string

	if configExists {
		loaded, err := config.LoadWithOptions(candidateCfgPath, config.LoadOptions{Prefix: bootstrap})
		if err != nil {
			logger.Warn("config load failed, falling back to setup mode", "path", cfgPath, "err", err)
			appConfig = nil
		} else {
			appConfig = loaded
			dbPath = loaded.DB.Path
			gitPath = loaded.Git.Path
		}
	}

	mode := "setup"
	if appConfig != nil {
		mode = "configured"
	}
	// config_path is the on-disk path used for stat/load (equals logical candidate when prefix is empty).
	logger.Info("KUI startup", "config_source", opts.configSource, "config_path", cfgPath, "mode", mode)

	if appConfig == nil {
		dbPath = strings.TrimSpace(os.Getenv("KUI_DB_PATH"))
		if dbPath == "" {
			dbPath = "/var/lib/kui/kui.db"
		}
		gitPath = strings.TrimSpace(os.Getenv("KUI_GIT_PATH"))
		if gitPath == "" {
			gitPath = "/var/lib/kui"
		}
		if bootstrap != "" {
			dbPath = prefix.Resolve(bootstrap, dbPath)
			gitPath = prefix.Resolve(bootstrap, gitPath)
		}
	}

	database, err := db.Open(dbPath)
	if err != nil {
		if configExists {
			database = nil
		} else {
			return nil, fmt.Errorf("open db: %w", err)
		}
	}

	if appConfig != nil && database != nil {
		if err := git.Init(appConfig.Git.Path); err != nil {
			_ = database.Close()
			return nil, fmt.Errorf("init git layout: %w", err)
		}
	}

	webPathPrefix := bootstrap
	routerOpts := routes.RouterOptions{
		Logger:        logger,
		DB:            database,
		Config:        appConfig,
		ConfigPath:    cfgPath,
		ConfigPresent: appConfig != nil,
		DBPath:        dbPath,
		GitPath:       gitPath,
		PathPrefix:    webPathPrefix,
	}
	if libvirtconn.SetupTestConnectorEnabled() {
		conn := libvirtconn.SetupTestConnector()
		routerOpts.SetupConnectFunc = func(ctx context.Context, uri, keyfile string) (libvirtconn.Connector, error) {
			return conn, nil
		}
	}
	mux := routes.NewRouter(routerOpts)
	return &application{
		logger:       logger,
		configPath:   cfgPath,
		config:       appConfig,
		configExists: configExists,
		db:           database,
		server: &http.Server{
			Handler: mux,
		},
		dbPath:  dbPath,
		gitPath: gitPath,
	}, nil
}

func getFileStatError(path string) error {
	_, err := os.Stat(path)
	return err
}

// resolveTLSCertKey applies prefix.Resolve to PEM paths when bootstrap prefix is non-empty
// after trim (same rule as static files and config bootstrap).
func resolveTLSCertKey(bootstrap, cert, key string) (string, string) {
	b := strings.TrimSpace(bootstrap)
	if b == "" {
		return cert, key
	}
	outCert, outKey := cert, key
	if strings.TrimSpace(cert) != "" {
		outCert = prefix.Resolve(b, cert)
	}
	if strings.TrimSpace(key) != "" {
		outKey = prefix.Resolve(b, key)
	}
	return outCert, outKey
}

func (app *application) startServer(listen, tlsCert, tlsKey string) (net.Listener, error) {
	listener, err := net.Listen("tcp", listen)
	if err != nil {
		return nil, err
	}
	app.listener = listener
	app.serveErr = make(chan error, 1)

	go func() {
		if tlsCert != "" && tlsKey != "" {
			if err := app.server.ServeTLS(listener, tlsCert, tlsKey); err != nil {
				app.serveErr <- err
				return
			}
		} else {
			if err := app.server.Serve(listener); err != nil {
				app.serveErr <- err
			}
		}
	}()
	return listener, nil
}

func (app *application) shutdown(ctx context.Context) error {
	if app == nil || app.server == nil {
		return nil
	}
	shutdownErr := app.server.Shutdown(ctx)
	dbErr := closeDatabase(app.db)
	if shutdownErr != nil && !errors.Is(shutdownErr, http.ErrServerClosed) {
		return shutdownErr
	}
	return dbErr
}

func closeDatabase(database *db.DB) error {
	if database == nil {
		return nil
	}
	return database.Close()
}
