package config

import (
	"fmt"
	"time"

	"gopkg.in/ini.v1"
)

const serverSection = "SERVER"

const (
	defaultHost            = "0.0.0.0"
	defaultPort            = 8080
	defaultName            = "Portfolio Server"
	defaultShutdownTimeout = 10 * time.Second
)

func loadServerConfig(section *ini.Section) (ServerConfig, error) {
	host, err := ResolveString(serverSection, "HOST", configKey(section, "HOST"), defaultHost)
	if err != nil {
		return ServerConfig{}, err
	}
	port, err := ResolveInt(serverSection, "PORT", configKey(section, "PORT"), defaultPort)
	if err != nil {
		return ServerConfig{}, err
	}
	name, err := ResolveString(serverSection, "NAME", configKey(section, "NAME"), defaultName)
	if err != nil {
		return ServerConfig{}, err
	}
	shutdownTimeout, err := ResolveSeconds(serverSection, "SHUTDOWN_TIMEOUT_SEC", configKey(section, "SHUTDOWN_TIMEOUT_SEC"), defaultShutdownTimeout)
	if err != nil {
		return ServerConfig{}, err
	}

	serverConfig := ServerConfig{
		Host:            host,
		Port:            port,
		Name:            name,
		ShutdownTimeout: shutdownTimeout,
	}
	if err := serverConfig.validate(); err != nil {
		return ServerConfig{}, err
	}
	return serverConfig, nil
}

// validate rejects values that parse cleanly but cannot be used to run a server.
func (c ServerConfig) validate() error {
	if c.Host == "" {
		return fmt.Errorf("SERVER.HOST must not be empty")
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("SERVER.PORT must be between 1 and 65535, got %d", c.Port)
	}
	if c.Name == "" {
		return fmt.Errorf("SERVER.NAME must not be empty")
	}
	if c.ShutdownTimeout <= 0 {
		return fmt.Errorf("SERVER.SHUTDOWN_TIMEOUT_SEC must be greater than 0, got %s", c.ShutdownTimeout)
	}
	return nil
}
