package main

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	_default_config_file = "./default.yaml"
	_default_backup_dir  = "./telefs"
)

type Config struct {
	ApiKey        string `yaml:"api_key"`
	TelegramToken string `yaml:"telegram_token"`
	BackupDir     string `yaml:"backup_dir"`
}

func (c *Config) validate() error {
	if c.ApiKey == "" {
		return fmt.Errorf("required api-key")
	}
	if c.TelegramToken == "" {
		return fmt.Errorf("required telegram token key")
	}

	return nil
}

func LoadConfig(toFile string) (*Config, error) {
	f, err := os.Open(toFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file :%v", err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file :%v", err)
	}

	in := Config{}
	if err := yaml.Unmarshal(b, &in); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	if err := in.validate(); err != nil {
		return nil, err
	}

	if in.BackupDir == "" {
		in.BackupDir = _default_backup_dir
	}
	return &in, nil
}
