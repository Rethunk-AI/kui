package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigPath = "/etc/kui/config.yaml"
)

type Duration time.Duration

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Kind != yaml.ScalarNode {
		return errors.New("duration value must be a scalar string")
	}

	parsed, err := time.ParseDuration(strings.TrimSpace(value.Value))
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", value.Value, err)
	}

	*d = Duration(parsed)
	return nil
}

func (d Duration) String() string {
	return time.Duration(d).String()
}

type Host struct {
	ID      string  `yaml:"id"`
	URI     string  `yaml:"uri"`
	Keyfile *string `yaml:"keyfile"`
}

type VMDefaults struct {
	CPU     int    `yaml:"cpu"`
	RAMMB   int    `yaml:"ram_mb"`
	Network string `yaml:"network"`
	DiskMB  *int   `yaml:"disk_mb"`
}

type Git struct {
	Path string `yaml:"path"`
}

type Session struct {
	Timeout Duration `yaml:"timeout"`
}

type VMLifecycle struct {
	GracefulStopTimeout Duration `yaml:"graceful_stop_timeout"`
}

type CORS struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

type DB struct {
	Path string `yaml:"path"`
}

type Config struct {
	Hosts               []Host      `yaml:"hosts"`
	VMDefaults          VMDefaults  `yaml:"vm_defaults"`
	DefaultHost         string      `yaml:"default_host"`
	DefaultPool         *string     `yaml:"default_pool"`
	DefaultNameTemplate string      `yaml:"default_name_template"`
	TemplateStorage     *string     `yaml:"template_storage"`
	Git                 Git         `yaml:"git"`
	Session             Session     `yaml:"session"`
	VMLifecycle         VMLifecycle `yaml:"vm_lifecycle"`
	CORS                CORS        `yaml:"cors"`
	DB                  DB          `yaml:"db"`
	JWTSecret           string      `yaml:"jwt_secret"`
}

const (
	defaultVMDefaultsCPU          = 2
	defaultVMDefaultsRAMMB        = 2048
	defaultVMDefaultsNetwork      = "default"
	defaultDefaultNameTemplate    = "{source}"
	defaultGitPath                = "/var/lib/kui"
	defaultSessionTimeout         = Duration(24 * time.Hour)
	defaultGracefulStopTimeout    = Duration(30 * time.Second)
	defaultDBPath                 = "/var/lib/kui/kui.db"
	defaultAllowedOriginLocalhost = "http://localhost:5173"
)

func LoadWithArgs(args []string) (*Config, string, error) {
	var configPath string
	fs := flag.NewFlagSet("kui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&configPath, "config", DefaultConfigPath, "path to yaml config file")
	if err := fs.Parse(args); err != nil {
		return nil, "", err
	}

	if envConfigPath := strings.TrimSpace(os.Getenv("KUI_CONFIG")); envConfigPath != "" {
		configPath = envConfigPath
	}

	cfg, err := Load(configPath)
	if err != nil {
		return nil, configPath, err
	}

	return cfg, configPath, nil
}

func Load(path string) (*Config, error) {
	configPath := strings.TrimSpace(path)
	if configPath == "" {
		configPath = DefaultConfigPath
	}

	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", configPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	applyDefaults(&cfg)
	if err := applyEnvOverrides(&cfg); err != nil {
		return nil, err
	}

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.DefaultNameTemplate == "" {
		cfg.DefaultNameTemplate = defaultDefaultNameTemplate
	}

	if cfg.VMDefaults.CPU == 0 {
		cfg.VMDefaults.CPU = defaultVMDefaultsCPU
	}
	if cfg.VMDefaults.RAMMB == 0 {
		cfg.VMDefaults.RAMMB = defaultVMDefaultsRAMMB
	}
	if cfg.VMDefaults.Network == "" {
		cfg.VMDefaults.Network = defaultVMDefaultsNetwork
	}

	if cfg.Git.Path == "" {
		cfg.Git.Path = defaultGitPath
	}
	if cfg.Session.Timeout == 0 {
		cfg.Session.Timeout = defaultSessionTimeout
	}
	if cfg.VMLifecycle.GracefulStopTimeout == 0 {
		cfg.VMLifecycle.GracefulStopTimeout = defaultGracefulStopTimeout
	}

	if len(cfg.CORS.AllowedOrigins) == 0 {
		cfg.CORS.AllowedOrigins = []string{defaultAllowedOriginLocalhost}
	}

	if cfg.DB.Path == "" {
		cfg.DB.Path = defaultDBPath
	}

	if cfg.DefaultHost == "" && len(cfg.Hosts) > 0 {
		cfg.DefaultHost = cfg.Hosts[0].ID
	}
}

func applyEnvOverrides(cfg *Config) error {
	if dbPath := strings.TrimSpace(os.Getenv("KUI_DB_PATH")); dbPath != "" {
		cfg.DB.Path = dbPath
	}
	if gitPath := strings.TrimSpace(os.Getenv("KUI_GIT_PATH")); gitPath != "" {
		cfg.Git.Path = gitPath
	}
	if defaultHost := strings.TrimSpace(os.Getenv("KUI_DEFAULT_HOST")); defaultHost != "" {
		cfg.DefaultHost = defaultHost
	}
	if defaultPool := strings.TrimSpace(os.Getenv("KUI_DEFAULT_POOL")); defaultPool != "" {
		cfg.DefaultPool = &defaultPool
	}
	if timeout := strings.TrimSpace(os.Getenv("KUI_SESSION_TIMEOUT")); timeout != "" {
		parsed, err := time.ParseDuration(timeout)
		if err != nil {
			return fmt.Errorf("invalid KUI_SESSION_TIMEOUT %q: %w", timeout, err)
		}
		cfg.Session.Timeout = Duration(parsed)
	}
	if jwtSecret := strings.TrimSpace(os.Getenv("KUI_JWT_SECRET")); jwtSecret != "" {
		cfg.JWTSecret = jwtSecret
	}
	if corsOrigins := strings.TrimSpace(os.Getenv("KUI_CORS_ORIGINS")); corsOrigins != "" {
		cfg.CORS.AllowedOrigins = splitCommaList(corsOrigins)
	}

	applyHostKeyfileEnvOverrides(cfg)

	return nil
}

func applyHostKeyfileEnvOverrides(cfg *Config) {
	for i := range cfg.Hosts {
		key := hostKeyfileEnvVar(cfg.Hosts[i].ID)
		if override, ok := os.LookupEnv(key); ok {
			trimmed := strings.TrimSpace(override)
			if trimmed == "" {
				cfg.Hosts[i].Keyfile = nil
			} else {
				cfg.Hosts[i].Keyfile = &trimmed
			}
		}
	}
}

func hostKeyfileEnvVar(hostID string) string {
	var b strings.Builder

	var previousUnderscore bool
	for _, char := range strings.ToUpper(hostID) {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			b.WriteRune(char)
			previousUnderscore = false
			continue
		}
		if !previousUnderscore {
			b.WriteRune('_')
			previousUnderscore = true
		}
	}

	return "KUI_" + strings.Trim(b.String(), "_") + "_KEYFILE"
}

func splitCommaList(input string) []string {
	parts := strings.Split(input, ",")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			clean = append(clean, value)
		}
	}
	return clean
}

func validate(cfg Config) error {
	if err := validateHosts(cfg.Hosts); err != nil {
		return err
	}
	if cfg.DefaultHost != "" {
		if _, ok := findHostByID(cfg.Hosts, cfg.DefaultHost); !ok {
			return fmt.Errorf("invalid default_host %q", cfg.DefaultHost)
		}
	}

	if cfg.Session.Timeout <= 0 {
		return errors.New("session.timeout must be a positive duration")
	}
	if cfg.VMLifecycle.GracefulStopTimeout < 0 {
		return errors.New("vm_lifecycle.graceful_stop_timeout must not be negative")
	}
	if cfg.VMDefaults.CPU <= 0 {
		return errors.New("vm_defaults.cpu must be greater than zero")
	}
	if cfg.VMDefaults.RAMMB <= 0 {
		return errors.New("vm_defaults.ram_mb must be greater than zero")
	}
	if strings.TrimSpace(cfg.VMDefaults.Network) == "" {
		return errors.New("vm_defaults.network is required")
	}
	if cfg.JWTSecret != "" && len(cfg.JWTSecret) < 32 {
		return errors.New("jwt_secret must be at least 32 bytes")
	}

	return nil
}

func validateHosts(hosts []Host) error {
	if len(hosts) == 0 {
		return errors.New("hosts is required")
	}

	seen := map[string]struct{}{}
	for _, host := range hosts {
		if strings.TrimSpace(host.ID) == "" {
			return errors.New("host id is required")
		}
		if strings.TrimSpace(host.URI) == "" {
			return fmt.Errorf("host %q is missing uri", host.ID)
		}
		if _, exists := seen[host.ID]; exists {
			return fmt.Errorf("duplicate host id %q", host.ID)
		}
		seen[host.ID] = struct{}{}
	}

	return nil
}

func findHostByID(hosts []Host, id string) (Host, bool) {
	for _, host := range hosts {
		if host.ID == id {
			return host, true
		}
	}
	return Host{}, false
}
