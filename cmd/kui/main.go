package main

import (
	"context"
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
	configPath string
	listen    string
	tlsCert   string
	tlsKey    string
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
		logger.Error("failed to start", "error", err)
		os.Exit(1)
	}
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

	listener, err := app.startServer(opts.listen, opts.tlsCert, opts.tlsKey)
	if err != nil {
		return err
	}
	logger.Info("KUI listening on", "addr", listener.Addr().String())
	if opts.tlsCert != "" {
		logger.Info("TLS enabled", "cert", opts.tlsCert, "key", opts.tlsKey)
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
	listenFlag := &trackedFlag{value: defaultListenAddress}
	tlsCertFlag := &trackedFlag{}
	tlsKeyFlag := &trackedFlag{}

	flagSet := flag.NewFlagSet("kui", flag.ContinueOnError)
	flagSet.SetOutput(io.Discard)
	flagSet.Var(configPathFlag, "config", "path to yaml config file")
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

	if envConfig := strings.TrimSpace(os.Getenv("KUI_CONFIG")); envConfig != "" && !configPathFlag.set {
		configPath = envConfig
	}
	if envListen := strings.TrimSpace(os.Getenv("KUI_LISTEN")); envListen != "" && !listenFlag.set {
		listen = envListen
	}

	if listen == "" {
		return startupOptions{}, fmt.Errorf("listen address is required")
	}
	if tlsCert == "" && tlsKey != "" || tlsCert != "" && tlsKey == "" {
		return startupOptions{}, fmt.Errorf("both --tls-cert and --tls-key must be provided together")
	}

	options = startupOptions{
		configPath: configPath,
		listen:     listen,
		tlsCert:    tlsCert,
		tlsKey:     tlsKey,
	}
	return options, nil
}

func buildApplication(opts startupOptions, logger *slog.Logger) (*application, error) {
	configExists := false
	cfgPath := strings.TrimSpace(opts.configPath)
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath
	}

	if statErr := getFileStatError(cfgPath); statErr == nil {
		configExists = true
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, fmt.Errorf("read config path %q: %w", cfgPath, statErr)
	}

	var appConfig *config.Config
	var dbPath string
	var gitPath string

	if configExists {
		loaded, err := config.Load(cfgPath)
		if err != nil {
			return nil, err
		}
		appConfig = loaded
		dbPath = loaded.DB.Path
		gitPath = loaded.Git.Path
	}

	if !configExists {
		dbPath = strings.TrimSpace(os.Getenv("KUI_DB_PATH"))
		if dbPath == "" {
			dbPath = "/var/lib/kui/kui.db"
		}
		gitPath = strings.TrimSpace(os.Getenv("KUI_GIT_PATH"))
		if gitPath == "" {
			gitPath = "/var/lib/kui"
		}
	}

	database, err := db.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if appConfig != nil {
		if err := git.Init(appConfig.Git.Path); err != nil {
			_ = database.Close()
			return nil, fmt.Errorf("init git layout: %w", err)
		}
	}

	mux := routes.NewRouter(routes.RouterOptions{
		Logger:        logger,
		DB:            database,
		Config:        appConfig,
		ConfigPath:    cfgPath,
		ConfigPresent: configExists,
		DBPath:        dbPath,
		GitPath:       gitPath,
	})
	return &application{
		logger:       logger,
		configPath:   cfgPath,
		config:       appConfig,
		configExists: configExists,
		db:           database,
		server: &http.Server{
			Handler: mux,
		},
		dbPath: dbPath,
		gitPath: gitPath,
	}, nil
}

func getFileStatError(path string) error {
	_, err := os.Stat(path)
	return err
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

