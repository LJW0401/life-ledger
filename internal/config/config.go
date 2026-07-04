// Package config reads config.toml, applies defaults, and fails fast on unsafe
// or incomplete runtime configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config is the complete runtime configuration loaded from config.toml.
type Config struct {
	Server   ServerConfig   `toml:"server"`
	Data     DataConfig     `toml:"data"`
	Auth     AuthConfig     `toml:"auth"`
	Security SecurityConfig `toml:"security"`
	Export   ExportConfig   `toml:"export"`
	Backup   BackupConfig   `toml:"backup"`
}

type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
}

type DataConfig struct {
	Dir      string `toml:"dir"`
	Database string `toml:"database"`
}

type AuthConfig struct {
	Username      string `toml:"username"`
	PasswordHash  string `toml:"password_hash"`
	SessionSecret string `toml:"session_secret"`
	SessionDays   int    `toml:"session_days"`
}

type SecurityConfig struct {
	TrustedProxies            []string `toml:"trusted_proxies"`
	LoginFailureWindowMinutes int      `toml:"login_failure_window_minutes"`
	LoginFailureLimit         int      `toml:"login_failure_limit"`
	LoginLockMinutes          int      `toml:"login_lock_minutes"`
	CookieSecure              bool     `toml:"cookie_secure"`
}

type ExportConfig struct {
	Timezone      string `toml:"timezone"`
	MaxUploadMB   int    `toml:"max_upload_mb"`
	MaxImportRows int    `toml:"max_import_rows"`
}

type BackupConfig struct {
	Dir string `toml:"dir"`
}

type rawConfig struct {
	Config
	Auth struct {
		Username      string `toml:"username"`
		Password      string `toml:"password"`
		PasswordHash  string `toml:"password_hash"`
		SessionSecret string `toml:"session_secret"`
		SessionDays   int    `toml:"session_days"`
	} `toml:"auth"`
}

// Load reads, decodes, defaults, and validates a config file.
func Load(path string) (Config, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return Config{}, fmt.Errorf("resolve config path: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	if !info.Mode().IsRegular() {
		return Config{}, fmt.Errorf("config path must be a regular file")
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		return Config{}, fmt.Errorf("config.toml permission must be 600, got %03o", perm)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var raw rawConfig
	if err := toml.Unmarshal(content, &raw); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	if raw.Auth.Password != "" {
		return Config{}, fmt.Errorf("auth.password is not allowed; use auth.password_hash")
	}

	cfg := raw.Config
	cfg.Auth.Username = raw.Auth.Username
	cfg.Auth.PasswordHash = raw.Auth.PasswordHash
	cfg.Auth.SessionSecret = raw.Auth.SessionSecret
	cfg.Auth.SessionDays = raw.Auth.SessionDays
	applyDefaults(&cfg)
	resolveRelativePaths(&cfg, filepath.Dir(path))
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Server.Host == "" {
		cfg.Server.Host = "127.0.0.1"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Data.Dir == "" {
		cfg.Data.Dir = "./data"
	}
	if cfg.Data.Database == "" {
		cfg.Data.Database = "life-ledger.db"
	}
	if cfg.Auth.SessionDays == 0 {
		cfg.Auth.SessionDays = 7
	}
	if len(cfg.Security.TrustedProxies) == 0 {
		cfg.Security.TrustedProxies = []string{"127.0.0.1"}
	}
	if cfg.Security.LoginFailureWindowMinutes == 0 {
		cfg.Security.LoginFailureWindowMinutes = 10
	}
	if cfg.Security.LoginFailureLimit == 0 {
		cfg.Security.LoginFailureLimit = 5
	}
	if cfg.Security.LoginLockMinutes == 0 {
		cfg.Security.LoginLockMinutes = 15
	}
	if cfg.Export.Timezone == "" {
		cfg.Export.Timezone = "Asia/Shanghai"
	}
	if cfg.Export.MaxUploadMB == 0 {
		cfg.Export.MaxUploadMB = 5
	}
	if cfg.Export.MaxImportRows == 0 {
		cfg.Export.MaxImportRows = 5000
	}
	if cfg.Backup.Dir == "" {
		cfg.Backup.Dir = "./backups"
	}
}

func resolveRelativePaths(cfg *Config, baseDir string) {
	if !filepath.IsAbs(cfg.Data.Dir) {
		cfg.Data.Dir = filepath.Join(baseDir, cfg.Data.Dir)
	}
	if !filepath.IsAbs(cfg.Backup.Dir) {
		cfg.Backup.Dir = filepath.Join(baseDir, cfg.Backup.Dir)
	}
}

func validate(cfg Config) error {
	if cfg.Auth.Username == "" {
		return fmt.Errorf("auth.username is required")
	}
	if cfg.Auth.PasswordHash == "" {
		return fmt.Errorf("auth.password_hash is required; run \"./life-ledger hash-password\" or \"./life-ledger init-config\" and write the result to config.toml")
	}
	if len(cfg.Auth.SessionSecret) < 32 {
		return fmt.Errorf("auth.session_secret must be at least 32 bytes; run \"./life-ledger generate-secret\" or \"./life-ledger init-config\" and write the result to config.toml")
	}
	if cfg.Auth.SessionDays <= 0 {
		return fmt.Errorf("auth.session_days must be positive")
	}
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if cfg.Data.Database == "" || filepath.Base(cfg.Data.Database) != cfg.Data.Database {
		return fmt.Errorf("data.database must be a file name")
	}
	if cfg.Security.LoginFailureWindowMinutes <= 0 || cfg.Security.LoginFailureLimit <= 0 || cfg.Security.LoginLockMinutes <= 0 {
		return fmt.Errorf("login failure limits must be positive")
	}
	if cfg.Export.MaxUploadMB <= 0 || cfg.Export.MaxImportRows <= 0 {
		return fmt.Errorf("export limits must be positive")
	}
	return nil
}

func (c ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
