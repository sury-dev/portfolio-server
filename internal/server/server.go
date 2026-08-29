package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/sury-dev/portfolio-server/internal/config"
	"github.com/sury-dev/portfolio-server/internal/database"
	"github.com/sury-dev/portfolio-server/internal/handler"
)

type Server struct {
	cfg    *config.Config
	logger zerolog.Logger
	app    *fiber.App
	db     *pgxpool.Pool
}

func NewServer(cfg *config.Config, logger zerolog.Logger) (*Server, error) {

	ctx := context.Background()

	app := fiber.New(fiber.Config{
		AppName:      cfg.Server.Name,
		ErrorHandler: handler.NewErrorHandler(logger),
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Session-ID",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	app.Get("/health", handler.HealthCheck)

	db, err := database.Connect(ctx, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := database.Ping(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Server{
		cfg:    cfg,
		logger: logger,
		app:    app,
		db:     db,
	}, nil
}

func (s *Server) Start() error {

	errCh := make(chan error, 1)

	go func() {
		addr := net.JoinHostPort(s.cfg.Server.Host, strconv.Itoa(s.cfg.Server.Port))
		s.logger.Info().Str("addr", addr).Msg("starting server")
		if err := s.app.Listen(addr); err != nil {
			s.logger.Error().Err(err).Msg("failed to start server")
			errCh <- err
		}
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		return err

	case <-signalCtx.Done():
		s.logger.Info().Msg("shutdown signal received")
	}

	shutdownCtx, shutdown := context.WithTimeout(context.Background(), s.cfg.Server.ShutdownTimeout)
	defer shutdown()

	if err := s.app.ShutdownWithContext(shutdownCtx); err != nil {
		s.logger.Error().Err(err).Msg("failed to shutdown server")
		return err
	}

	s.logger.Info().Msg("server shutdown complete")
	return nil
}

// Close releases resources owned by the server. Call it after Start returns
// so in-flight requests can finish using the database first.
func (s *Server) Close() {
	if s.db != nil {
		s.db.Close()
	}
	s.logger.Info().Msg("database pool closed")
}
