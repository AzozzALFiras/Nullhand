package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	configmodel "github.com/AzozzALFiras/nullhand/internal/model/config"
)

const (
	configDir  = ".nullhand"
	configFile = "config.json"
	configPerm = 0600
	dirPerm    = 0700
)

// configPath returns the absolute path to the config file.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, configDir, configFile), nil
}

// Exists reports whether the config file exists on disk.
func Exists() bool {
	path, err := configPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Load reads the config file and returns the decoded Config.
func Load() (*configmodel.Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}

	var cfg configmodel.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse config: %w", err)
	}
	return &cfg, nil
}

// Save writes the config to disk with 0600 permissions.
func Save(cfg *configmodel.Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot encode config: %w", err)
	}

	if err := os.WriteFile(path, data, configPerm); err != nil {
		return fmt.Errorf("cannot write config: %w", err)
	}
	return nil
}
