package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/sury-dev/portfolio-server/internal/config"
	"github.com/sury-dev/portfolio-server/internal/database"
	"github.com/sury-dev/portfolio-server/internal/logger"
)

const defaultConfigPath = "configs/config.conf"
const defaultMigratePath = "migrations"

func main() {
	configPath := flag.String("conf", defaultConfigPath, "path to the config file")
	migratePath := flag.String("migrate", defaultMigratePath, "path to the migrate file")
	action := flag.String("action", "up", "action to perform: up, down, version")
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

	absDir, err := filepath.Abs(*migratePath)
	if err != nil {
		log.Fatalf("error getting absolute path: %v", err)
	}

	sourceURL := fmt.Sprintf("file://%s", absDir)

	migrator, err := migrate.New(sourceURL, database.DSN(cfg.Database))
	if err != nil {
		log.Fatalf("error creating migrator: %v", err)
	}
	defer func() {
		srcErr, dbErr := migrator.Close()
		if srcErr != nil {
			log.Printf("error closing migration source: %v", srcErr)
		}
		if dbErr != nil {
			log.Printf("error closing migration database: %v", dbErr)
		}
	}()

	switch *action {
	case "up":
		err = migrator.Up()
		if errors.Is(err, migrate.ErrNoChange) {
			appLogger.Info().Msg("migrations already up to date")
			return
		}
		if err != nil {
			appLogger.Error().Err(err).Msg("migrate up failed")
			os.Exit(1)
		}
		appLogger.Info().Msg("migrations applied")
	case "down":
		err = migrator.Steps(-1)
		if errors.Is(err, migrate.ErrNoChange) {
			appLogger.Info().Msg("no migrations to roll back")
			return
		}
		if err != nil {
			appLogger.Error().Err(err).Msg("migrate down failed")
			os.Exit(1)
		}
		appLogger.Info().Msg("rolled back one migration")
	case "version":
		version, dirty, err := migrator.Version()
		if errors.Is(err, migrate.ErrNilVersion) {
			appLogger.Info().Msg("no migrations applied yet")
			return
		}
		if err != nil {
			appLogger.Error().Err(err).Msg("migrate version failed")
			os.Exit(1)
		}
		appLogger.Info().Uint("version", version).Bool("dirty", dirty).Msg("migration version")
	default:
		appLogger.Error().Str("command", *action).Msg("unknown command; use up, down, or version")
		os.Exit(1)
	}
}
