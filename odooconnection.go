package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/ppreeper/odoorpc"
	"github.com/ppreeper/odoorpc/odoojrpc"
)

type serverConfig struct {
	Hostname string `toml:"hostname"`
	Port     int    `toml:"port"`
	Schema   string `toml:"schema"`
	Database string `toml:"database"`
	Username string `toml:"username"`
	Apikey   string `toml:"apikey"`
}

// configFilePath returns the expected location of the configuration file.
func configFilePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "~/.config/thinksoft/config.toml"
	}
	return filepath.Join(configDir, "thinksoft", "config.toml")
}

// loadConfig reads and parses the config file, returning the connection settings.
func loadConfig() (serverConfig, error) {
	type configFile struct {
		Connection serverConfig `toml:"connection"`
	}
	doc, err := os.ReadFile(configFilePath())
	if err != nil {
		return serverConfig{}, fmt.Errorf("reading config: %w", err)
	}
	var cfg configFile
	if err := toml.Unmarshal(doc, &cfg); err != nil {
		return serverConfig{}, fmt.Errorf("parsing config TOML: %w", err)
	}
	return cfg.Connection, nil
}

// NewConn creates and authenticates a new Odoo connection using the config file.
func NewConn() (odoorpc.Odoo, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	conn := odoojrpc.NewOdoo().
		WithHostname(cfg.Hostname).
		WithPort(cfg.Port).
		WithSchema(cfg.Schema).
		WithDatabase(cfg.Database).
		WithUsername(cfg.Username).
		WithPassword(cfg.Apikey)
	if err := conn.Login(); err != nil {
		return nil, fmt.Errorf("odoo login: %w", err)
	}
	return conn, nil
}
