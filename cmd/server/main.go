package main

import (
	"flag"
	"log"

	"github.com/sury-dev/portfolio-server/internal/config"
	"github.com/sury-dev/portfolio-server/internal/logger"
	"github.com/sury-dev/portfolio-server/internal/server"
)

const defaultConfigPath = "configs/config.conf"

func main() {
	configPath := flag.String("conf", defaultConfigPath, "path to the config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	logger, closer, err := logger.NewLogger(cfg.Logging, cfg.Server.Name)
	if err != nil {
		log.Fatalf("error creating logger: %v", err)
	}
	defer func() {
		if err := closer(); err != nil {
			log.Fatalf("error closing logger: %v", err)
		}
	}()

	srv, err := server.NewServer(cfg, logger)
	if err != nil {
		log.Fatalf("error creating server: %v", err)
	}

	if err := srv.Start(); err != nil {
		log.Fatalf("error starting server: %v", err)
	}
}
