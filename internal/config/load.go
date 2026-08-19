package config

import (
	"fmt"

	"gopkg.in/ini.v1"
)

// Load reads the config file at configPath and returns fully resolved,
// validated configuration. Environment variables override file values.
func Load(configPath string) (*Config, error) {
	file, err := ini.LoadSources(ini.LoadOptions{
		IgnoreInlineComment: true,
		Insensitive:         true,
	}, configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config file %q: %w", configPath, err)
	}

	serverConfig, err := loadServerConfig(file.Section(serverSection))
	if err != nil {
		return nil, fmt.Errorf("invalid server configuration: %w", err)
	}

	loggingConfig, err := loadLoggingConfig(file.Section(loggingSection))
	if err != nil {
		return nil, fmt.Errorf("invalid logging configuration: %w", err)
	}

	return &Config{
		Server:  serverConfig,
		Logging: loggingConfig,
	}, nil
}
