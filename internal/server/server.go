package server

import (
	"context"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/rs/zerolog"
	"github.com/sury-dev/portfolio-server/internal/config"
	"github.com/sury-dev/portfolio-server/internal/handler"
)

type Server struct {
	cfg    *config.Config
	logger zerolog.Logger
	app    *fiber.App
}

func NewServer(cfg *config.Config, logger zerolog.Logger) (*Server, error) {
	app := fiber.New(fiber.Config{
		AppName: cfg.Server.Name,
		ErrorHandler: handler.NewErrorHandler(logger),

	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Session-ID",
		AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
	}))

	app.Get("/health", handler.HealthCheck)

	return &Server{
		cfg:    cfg,
		logger: logger,
		app:    app,
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
