package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sury-dev/portfolio-server/internal/config"
)

func Connect(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	connConfig, err := pgxpool.ParseConfig(
		fmt.Sprintf(
			"user=%s password=%s host=%s port=%d dbname=%s sslmode=%s",
			cfg.User,
			cfg.Password,
			cfg.Host,
			cfg.Port,
			cfg.Name,
			cfg.SSLMode,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database configuration: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	return pool, nil
}

func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}