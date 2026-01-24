package main

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	SearchLimit int    `toml:"search_limit"`
	DownloadDir string `toml:"download_dir"`
}

var config Config

func LoadConfig() Config {
	// Default to current working directory
	pwd, _ := os.Getwd()

	cfg := Config{
		SearchLimit: 20,
		DownloadDir: pwd,
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

	// Expand ~ in download_dir
	if len(cfg.DownloadDir) > 0 && cfg.DownloadDir[0] == '~' {
		cfg.DownloadDir = filepath.Join(home, cfg.DownloadDir[1:])
	}

	// If still empty, use pwd
	if cfg.DownloadDir == "" {
		cfg.DownloadDir = pwd
	}

	return cfg
}

func init() {
	config = LoadConfig()
}
