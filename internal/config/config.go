// internal/config/config.go
package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

// Config is the on-disk configuration.
type Config struct {
	Host         string        `toml:"host"`
	Bind         string        `toml:"bind"`
	Port         int           `toml:"port"`
	AdminPort    int           `toml:"admin_port"`
	DataDir      string        `toml:"data_dir"`
	DefaultTTL   time.Duration `toml:"default_ttl"`
	IdleShutdown time.Duration `toml:"idle_shutdown"`
	TLSCert      string        `toml:"tls_cert"`
	TLSKey       string        `toml:"tls_key"`
}

// Load reads a Config from path. If the file does not exist, a template is
// written and an error is returned so the CLI can surface the condition.
func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if werr := writeTemplate(path); werr != nil {
			return Config{}, fmt.Errorf("write template: %w", werr)
		}
		return Config{}, fmt.Errorf("config created at %s — edit it and re-run", path)
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	if err := toml.Unmarshal(b, &c); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	c = withDefaults(c)
	if c.Host == "" {
		return Config{}, errors.New("config.host is empty — set it to a reachable hostname")
	}
	if (c.TLSCert == "") != (c.TLSKey == "") {
		return Config{}, errors.New("config: tls_cert and tls_key must both be set or both be empty")
	}
	return c, nil
}

// Save writes the Config to path.
func Save(path string, c Config) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(c)
}

// Merge applies override's non-zero fields over base.
func Merge(base, override Config) Config {
	out := base
	if override.Host != "" {
		out.Host = override.Host
	}
	if override.Bind != "" {
		out.Bind = override.Bind
	}
	if override.Port != 0 {
		out.Port = override.Port
	}
	if override.AdminPort != 0 {
		out.AdminPort = override.AdminPort
	}
	if override.DataDir != "" {
		out.DataDir = override.DataDir
	}
	if override.DefaultTTL != 0 {
		out.DefaultTTL = override.DefaultTTL
	}
	if override.IdleShutdown != 0 {
		out.IdleShutdown = override.IdleShutdown
	}
	if override.TLSCert != "" {
		out.TLSCert = override.TLSCert
	}
	if override.TLSKey != "" {
		out.TLSKey = override.TLSKey
	}
	return out
}
