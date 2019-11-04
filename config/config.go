package config

import (
	"os"
	"path/filepath"
)

var c *Config

type Config struct {
	SeedEntropyBits int
	AppDataDir string
	DbPath string
}

func Load() (*Config, error ){
	if c == nil {
		c = &Config{
			SeedEntropyBits: 128,
		}
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	c.AppDataDir = filepath.Join(homeDir, ".coldWallet")
	c.DbPath = filepath.Join(c.AppDataDir, "main.db")

	if _, err := os.Stat(c.AppDataDir); os.IsNotExist(err) {
		if err := os.Mkdir(c.AppDataDir, 0755); err != nil {
			return nil, err
		}
	}
	return c, nil
}
