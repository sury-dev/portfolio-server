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

	appLogger, closer, err := logger.NewLogger(cfg.Logging, cfg.Server.Name)
	if err != nil {
		log.Fatalf("error creating logger: %v", err)
	}
	defer func() {
		if err := closer(); err != nil {
			log.Printf("error closing logger: %v", err)
		}
	}()

	srv, err := server.NewServer(cfg, appLogger)
	if err != nil {
		log.Printf("error creating server: %v", err)
		return
	}
	defer srv.Close()

	if err := srv.Start(); err != nil {
		log.Printf("error starting server: %v", err)
	}
}
