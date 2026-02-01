package main

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	SearchLimit  int    `toml:"search_limit"`
	DownloadDir  string `toml:"download_dir"`
	OMDBAPIKey   string `toml:"omdb_api_key"`
	SearchSource string `toml:"search_source"` // "yts" or "torrents-csv"
}

var config Config

func LoadConfig() Config {
	// Default to current working directory
	pwd, _ := os.Getwd()

	cfg := Config{
		SearchLimit: 50,
		DownloadDir: pwd,
		OMDBAPIKey:  os.Getenv("OMDB_API_KEY"),
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
