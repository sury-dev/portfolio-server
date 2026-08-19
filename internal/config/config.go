package config

import "time"

type Config struct {
	Server ServerConfig
}

type ServerConfig struct {
	Host            string
	Port            int
	Name            string
	ShutdownTimeout time.Duration
}
