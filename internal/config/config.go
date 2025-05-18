package config

import (
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

type (
	RestConfig struct {
		Port       int      `yaml:"port,omitempty"`
		AuthTokens []string `yaml:"auth_tokens,omitempty"`
	}
	Config struct {
		Rest RestConfig `yaml:"rest"`
	}
)

const (
	DefaultPort = 8080
)

func Load(file string, logger *slog.Logger) (*Config, error) {
	config := &Config{
		Rest: RestConfig{
			Port:       DefaultPort,
			AuthTokens: []string{},
		},
	}

	yamlFile, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("Config file does not exist, using default config")

			return config, nil
		}

		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err = yaml.Unmarshal(yamlFile, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file: %w", err)
	}

	return config, nil
}
