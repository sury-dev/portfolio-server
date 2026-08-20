package server

import (
	"net"
	"strconv"

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
	addr := net.JoinHostPort(s.cfg.Server.Host, strconv.Itoa(s.cfg.Server.Port))
	s.logger.Info().Str("addr", addr).Msg("starting server")
	return s.app.Listen(addr)
}
