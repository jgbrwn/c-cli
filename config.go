package main

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	DefaultAction string `toml:"default_action"`
	SearchLimit   int    `toml:"search_limit"`
}

func LoadConfig() Config {
	cfg := Config{
		DefaultAction: "magnet",
		SearchLimit:   20,
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	configPath := filepath.Join(home, ".config", "c-cli", "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return cfg
	}

	_, _ = toml.DecodeFile(configPath, &cfg)
	return cfg
}
