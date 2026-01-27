package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	HTTPPort    int      `json:"httpPort"`
	AllowedCors []string `json:"allowedCors"`
}

var (
	configDir  string
	configPath string
	mu         sync.Mutex
	cfg        *Config
)

func init() {
	// Set default config directory
	// Use os.UserConfigDir() for production
	configBase, err := os.UserConfigDir()
	if err != nil {
		// Fallback to local
		cwd, _ := os.Getwd()
		configBase = cwd
	}

	configDir = filepath.Join(configBase, "ts-escpos")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		// Log error?
		println("Failed to create config dir:", err.Error())
	}

	configPath = filepath.Join(configDir, "config.json")

	cfg = &Config{
		HTTPPort:    9100,
		AllowedCors: []string{"*"},
	}
}

func LoadConfig() *Config {
	mu.Lock()
	defer mu.Unlock()

	data, err := os.ReadFile(configPath)
	if err == nil {
		json.Unmarshal(data, cfg)
	}
	return cfg
}

func SaveConfig(c *Config) error {
	mu.Lock()
	defer mu.Unlock()

	cfg = c
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}
