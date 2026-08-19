package main

import (
	"flag"
	"log"

	"github.com/sury-dev/portfolio-server/internal/config"
)

const defaultConfigPath = "configs/config.conf"

func main() {
	configPath := flag.String("conf", defaultConfigPath, "path to the config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("error loading config: %v", err)
	}

	log.Printf("configuration loaded: %+v", cfg.Server)
}
