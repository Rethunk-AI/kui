package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/kui/kui/internal/prefix"

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
	Timeout       Duration `yaml:"timeout"`
	SecureCookies *bool   `yaml:"secure_cookies"`
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

// trackedString is a flag.Value that records whether the flag was set on the command line.
type trackedString struct {
	value string
	set   bool
}

func (t *trackedString) String() string { return t.value }

func (t *trackedString) Set(s string) error {
	t.value = s
	t.set = true
	return nil
}

// LoadOptions configures [LoadWithOptions].
// When Prefix is non-empty (after TrimSpace), the config file is read from
// prefix.Resolve(Prefix, candidatePath) and every local filesystem path in the
// loaded config is normalized under Prefix after env overrides.
// Prefix comes from --prefix / KUI_PREFIX at startup, or from tests (e.g. t.TempDir());
// YAML does not define a runtime prefix.
type LoadOptions struct {
	Prefix string
}

func LoadWithArgs(args []string) (*Config, string, error) {
	configPathFlag := &trackedString{value: DefaultConfigPath}
	prefixFlag := &trackedString{}

	fs := flag.NewFlagSet("kui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Var(configPathFlag, "config", "path to yaml config file")
	fs.Var(prefixFlag, "prefix", "runtime filesystem root (chroot-style path resolution)")
	if err := fs.Parse(args); err != nil {
		return nil, "", err
	}

	// Match cmd/kui parseFlags: KUI_CONFIG applies only when --config was not set.
	configPath := strings.TrimSpace(configPathFlag.value)
	if envConfig := strings.TrimSpace(os.Getenv("KUI_CONFIG")); envConfig != "" && !configPathFlag.set {
		configPath = envConfig
	}
	if configPath == "" {
		configPath = DefaultConfigPath
	}

	bootstrap := strings.TrimSpace(prefixFlag.value)
	if !prefixFlag.set {
		bootstrap = strings.TrimSpace(os.Getenv("KUI_PREFIX"))
	}

	readPath := resolvedConfigReadPath(configPath, bootstrap)
	cfg, err := loadFromResolvedPath(readPath, LoadOptions{Prefix: bootstrap})
	if err != nil {
		return nil, readPath, err
	}
	return cfg, readPath, nil
}

// Load loads YAML from candidate path. When prefix is non-empty, the file is read from
// prefix.Resolve(prefix, candidate) and local path fields are normalized under prefix.
func Load(path string, prefix string) (*Config, error) {
	return LoadWithOptions(path, LoadOptions{Prefix: strings.TrimSpace(prefix)})
}

// LoadWithOptions loads YAML from candidate path (before prefix join for the read path).
func LoadWithOptions(path string, opts LoadOptions) (*Config, error) {
	candidate := strings.TrimSpace(path)
	if candidate == "" {
		candidate = DefaultConfigPath
	}
	readPath := resolvedConfigReadPath(candidate, opts.Prefix)
	return loadFromResolvedPath(readPath, opts)
}

func resolvedConfigReadPath(candidate, bootstrap string) string {
	b := strings.TrimSpace(bootstrap)
	if b == "" {
		return candidate
	}
	return prefix.Resolve(b, candidate)
}

func loadFromResolvedPath(readPath string, opts LoadOptions) (*Config, error) {
	raw, err := os.ReadFile(readPath)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", readPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	applyDefaults(&cfg)
	if err := applyEnvOverrides(&cfg); err != nil {
		return nil, err
	}

	normalizeLocalPathFields(&cfg, strings.TrimSpace(opts.Prefix))

	if err := validate(cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// normalizeLocalPathFields applies prefix.Resolve to every local filesystem path held in [Config].
// Pool identifiers (e.g. template_storage, default_pool) are not paths and are left unchanged.
func normalizeLocalPathFields(cfg *Config, effectivePrefix string) {
	p := strings.TrimSpace(effectivePrefix)
	if p == "" {
		return
	}
	cfg.DB.Path = prefix.Resolve(p, cfg.DB.Path)
	cfg.Git.Path = prefix.Resolve(p, cfg.Git.Path)
	for i := range cfg.Hosts {
		if cfg.Hosts[i].Keyfile == nil {
			continue
		}
		k := strings.TrimSpace(*cfg.Hosts[i].Keyfile)
		if k == "" {
			continue
		}
		resolved := prefix.Resolve(p, k)
		cfg.Hosts[i].Keyfile = &resolved
	}
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
	if cfg.Session.SecureCookies == nil {
		trueVal := true
		cfg.Session.SecureCookies = &trueVal
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
	if v := strings.TrimSpace(os.Getenv("KUI_SECURE_COOKIES")); v != "" {
		p := parseBoolPtr(v)
		if p == nil {
			return fmt.Errorf("invalid KUI_SECURE_COOKIES %q: must be true/false, 1/0, or yes/no", v)
		}
		cfg.Session.SecureCookies = p
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

	return "KUI_HOST_" + strings.Trim(b.String(), "_") + "_KEYFILE"
}

func parseBoolPtr(s string) *bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes":
		v := true
		return &v
	case "false", "0", "no":
		v := false
		return &v
	default:
		return nil
	}
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
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return errors.New("jwt_secret is required")
	}
	if len(cfg.JWTSecret) < 32 {
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

		if strings.HasPrefix(strings.TrimSpace(host.URI), "qemu+ssh://") {
			keyfileSet := host.Keyfile != nil && strings.TrimSpace(*host.Keyfile) != ""
			if !keyfileSet {
				parsed, err := url.Parse(host.URI)
				if err != nil || strings.TrimSpace(parsed.Query().Get("keyfile")) == "" {
					return fmt.Errorf("host %q: keyfile is required for qemu+ssh URI (set in config or %s)", host.ID, hostKeyfileEnvVar(host.ID))
				}
			}
		}
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
