package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const configFile = "tasks.json"

func ConfigDir() string {
	if dir := os.Getenv("CRONIX_CONFIG_DIR"); dir != "" {
		return dir
	}
	homedir, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", "cronix")
	}
	return filepath.Join(homedir, ".config", "cronix")
}

func EnsureConfigDir() error {
	return os.MkdirAll(ConfigDir(), 0o755)
}

func Load() (*Config, error) {
	path := filepath.Join(ConfigDir(), configFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".tasks-*.json.tmp")
	if err != nil {
		return fmt.Errorf("config: create temp: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		if _, statErr := os.Stat(tmpName); statErr == nil {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("config: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: close temp: %w", err)
	}
	dest := filepath.Join(dir, configFile)
	if err := os.Rename(tmpName, dest); err != nil {
		return fmt.Errorf("config: rename to %s: %w", dest, err)
	}
	return nil
}
