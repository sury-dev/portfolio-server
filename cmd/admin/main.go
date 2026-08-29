package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/sury-dev/portfolio-server/internal/config"
	"github.com/sury-dev/portfolio-server/internal/database"
	"github.com/sury-dev/portfolio-server/internal/logger"
	"github.com/sury-dev/portfolio-server/internal/utils"
	"golang.org/x/term"
)

const defaultConfigPath = "configs/config.conf"

func main() {
	configPath := flag.String("conf", defaultConfigPath, "path to the config file")
	action := flag.String("a", "set-password", "action to perform: set-password")
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

	db, err := database.Connect(context.Background(), cfg.Database)
	if err != nil {
		log.Fatalf("error connecting to database: %v", err)
	}
	defer db.Close()

	if err := database.Ping(context.Background(), db); err != nil {
		log.Fatalf("error pinging database: %v", err)
	}

	switch *action {
	case "set-password":
		password, err := readPasswordTwice()
		if err != nil {
			log.Fatalf("error reading password: %v", err)
		}

		hashedPassword, err := utils.HashString(password)
		if err != nil {
			log.Fatalf("error hashing password: %v", err)
		}

		ctx := context.Background()
		_, err = db.Exec(ctx, `
			INSERT INTO admin (
				password_hash,
				lock_key
			)
			VALUES ($1, TRUE)
			ON CONFLICT (lock_key)
			DO UPDATE SET
				password_hash = EXCLUDED.password_hash,
				access_token_hash = NULL,
				refresh_token_hash = NULL,
				access_token_expires_at = NULL,
				refresh_token_expires_at = NULL,
				updated_at = NOW()`,
			hashedPassword,
		)
		if err != nil {
			log.Fatalf("error setting admin password: %v", err)
		}
		appLogger.Info().Str("action", *action).Msg("admin password set; session cleared")

	default:
		appLogger.Error().Str("action", *action).Msg("unknown action; use set-password")
		os.Exit(1)
	}
}

func readPasswordTwice() (string, error) {
	fmt.Fprint(os.Stderr, "Password: ")
	firstBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}

	fmt.Fprint(os.Stderr, "Confirm password: ")
	secondBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}

	first := strings.TrimSpace(string(firstBytes))
	second := strings.TrimSpace(string(secondBytes))

	if first == "" {
		return "", fmt.Errorf("password must not be empty")
	}
	if first != second {
		return "", fmt.Errorf("passwords do not match")
	}

	return first, nil
}
