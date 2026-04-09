package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// configFilePath
// ---------------------------------------------------------------------------

func TestConfigFilePath(t *testing.T) {
	t.Parallel()
	// The returned path must end with the well-known relative suffix regardless
	// of where XDG_CONFIG_HOME or HOME points on this machine.
	got := configFilePath()
	const suffix = filepath.Separator
	_ = suffix
	if !strings.HasSuffix(got, filepath.Join("thinksoft", "config.toml")) {
		t.Errorf("configFilePath() = %q, want suffix %q", got, filepath.Join("thinksoft", "config.toml"))
	}
}

// ---------------------------------------------------------------------------
// loadConfig — happy path and error paths via temp file
// ---------------------------------------------------------------------------

// writeConfig writes a TOML config to a temp dir and points XDG_CONFIG_HOME
// (and HOME as fallback) at it so loadConfig() reads the temp file.
func writeConfig(t *testing.T, tomlContent string) {
	t.Helper()
	cfgDir := t.TempDir()
	thinksoftDir := filepath.Join(cfgDir, "thinksoft")
	if err := os.MkdirAll(thinksoftDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	cfgFile := filepath.Join(thinksoftDir, "config.toml")
	if err := os.WriteFile(cfgFile, []byte(tomlContent), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	// XDG_CONFIG_HOME takes precedence in os.UserConfigDir on Linux.
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
}

func TestLoadConfigValid(t *testing.T) {
	writeConfig(t, `
[connection]
hostname = "odoo.example.com"
port     = 8069
schema   = "https"
database = "mydb"
username = "user@example.com"
apikey   = "secret"
`)
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.Hostname != "odoo.example.com" {
		t.Errorf("Hostname = %q, want %q", cfg.Hostname, "odoo.example.com")
	}
	if cfg.Port != 8069 {
		t.Errorf("Port = %d, want 8069", cfg.Port)
	}
	if cfg.Schema != "https" {
		t.Errorf("Schema = %q, want %q", cfg.Schema, "https")
	}
	if cfg.Database != "mydb" {
		t.Errorf("Database = %q, want %q", cfg.Database, "mydb")
	}
	if cfg.Username != "user@example.com" {
		t.Errorf("Username = %q, want %q", cfg.Username, "user@example.com")
	}
	if cfg.Apikey != "secret" {
		t.Errorf("Apikey = %q, want %q", cfg.Apikey, "secret")
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	// Point XDG_CONFIG_HOME at a temp dir that has no config file.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, err := loadConfig()
	if err == nil {
		t.Fatal("loadConfig() with missing file returned nil error")
	}
}

func TestLoadConfigInvalidTOML(t *testing.T) {
	writeConfig(t, "this is not valid toml {{{{")
	_, err := loadConfig()
	if err == nil {
		t.Fatal("loadConfig() with invalid TOML returned nil error")
	}
}

func TestLoadConfigEmptyFile(t *testing.T) {
	// An empty file is valid TOML — all fields should be zero values.
	writeConfig(t, "")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() with empty file error = %v", err)
	}
	if cfg.Hostname != "" || cfg.Port != 0 {
		t.Errorf("expected zero-value config, got %+v", cfg)
	}
}

func TestLoadConfigMissingConnectionSection(t *testing.T) {
	// File has different section — connection fields should be zero.
	writeConfig(t, "[other]\nkey = \"value\"\n")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.Hostname != "" {
		t.Errorf("expected empty hostname, got %q", cfg.Hostname)
	}
}
