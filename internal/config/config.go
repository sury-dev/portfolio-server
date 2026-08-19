package config

import "time"

type Config struct {
	Server  ServerConfig
	Logging LoggingConfig
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

