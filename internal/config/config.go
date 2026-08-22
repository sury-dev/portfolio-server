package config

import "time"

type Config struct {
	Server  ServerConfig
	Logging LoggingConfig
	Database DatabaseConfig
}

type ServerConfig struct {
	Host            string
	Port            int
	Name            string
	ShutdownTimeout time.Duration
}

type LoggingConfig struct {
	Level  string
	Format string
	Output string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}