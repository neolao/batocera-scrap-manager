// Package config manages the persisted configuration of batocera-scrap-manager:
// the registry path and the Batocera ROMs folders to watch.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// EnvConfigPath is the environment variable used to override the default
// configuration file location. Mainly useful for tests and isolated runs.
const EnvConfigPath = "BATOCERA_SCRAP_MANAGER_CONFIG"

// Config holds the persisted settings of batocera-scrap-manager.
type Config struct {
	RegistryPath string   `json:"registry_path"`
	RomsFolders  []string `json:"roms_folders"`
}

// DefaultPath returns the configuration file path: the EnvConfigPath
// environment variable if set, otherwise a path under the OS user config dir.
func DefaultPath() (string, error) {
	if p := os.Getenv(EnvConfigPath); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "batocera-scrap-manager", "config.json"), nil
}

// Load reads the configuration from path. If the file does not exist, it
// returns an empty Config with no error.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save writes cfg to path as JSON, creating parent directories as needed.
func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// SetRegistryPath sets the registry path, resolved to an absolute path.
func (c *Config) SetRegistryPath(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	c.RegistryPath = abs
	return nil
}

// AddRomsFolder adds path (resolved to an absolute path) to the configured
// ROMs folders. It reports whether the folder was newly added, i.e. it
// returns false when the (absolute) folder was already configured.
func (c *Config) AddRomsFolder(path string) (added bool, err error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}

	for _, existing := range c.RomsFolders {
		if existing == abs {
			return false, nil
		}
	}

	c.RomsFolders = append(c.RomsFolders, abs)
	return true, nil
}
