package config

import (
	"fmt"

	"gopkg.in/ini.v1"
)

const databaseSection = "DATABASE"

const (
	defaultDatabaseHost = "127.0.0.1"
	defaultDatabasePort = 5432
	defaultDatabaseName = "portfolio"
	defaultDatabaseUser = "surydev"
	defaultDatabasePassword = ""
	defaultDatabaseSSLMode = "disable"
)

func loadDatabaseConfig(section *ini.Section) (DatabaseConfig, error) {
	host, err := ResolveString(databaseSection, "HOST", configKey(section, "HOST"), defaultDatabaseHost)
	if err != nil {
		return DatabaseConfig{}, err
	}

	port, err := ResolveInt(databaseSection, "PORT", configKey(section, "PORT"), defaultDatabasePort)
	if err != nil {
		return DatabaseConfig{}, err
	}

	name, err := ResolveString(databaseSection, "NAME", configKey(section, "NAME"), defaultDatabaseName)
	if err != nil {
		return DatabaseConfig{}, err
	}
	
	user, err := ResolveString(databaseSection, "USER", configKey(section, "USER"), defaultDatabaseUser)
	if err != nil {
		return DatabaseConfig{}, err
	}

	password, err := ResolveString(databaseSection, "PASSWORD", configKey(section, "PASSWORD"), defaultDatabasePassword)
	if err != nil {
		return DatabaseConfig{}, err
	}
	
	sslMode, err := ResolveString(databaseSection, "SSL_MODE", configKey(section, "SSL_MODE"), defaultDatabaseSSLMode)
	if err != nil {
		return DatabaseConfig{}, err
	}

	databaseConfig := DatabaseConfig{
		Host: host,
		Port: port,
		Name: name,
		User: user,
		Password: password,
		SSLMode: sslMode,
	}
	if err := databaseConfig.validate(); err != nil {
		return DatabaseConfig{}, err
	}
	return databaseConfig, nil
}

func (c DatabaseConfig) validate() error {
	switch c.SSLMode {
	case "disable", "prefer", "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("DATABASE.SSL_MODE must be one of disable, prefer, require, verify-ca, verify-full, got %q", c.SSLMode)
	}

	if c.Password == "" {
		return fmt.Errorf("DATABASE.PASSWORD is required")
	}

	if c.User == "" {
		return fmt.Errorf("DATABASE.USER is required")
	}

	if c.Name == "" {
		return fmt.Errorf("DATABASE.NAME is required")
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("DATABASE.PORT must be between 1 and 65535, got %d", c.Port)
	}

	if c.Host == "" {
		return fmt.Errorf("DATABASE.HOST is required")
	}

	return nil
}