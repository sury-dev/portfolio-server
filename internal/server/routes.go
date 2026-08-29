package server

import (
	"github.com/sury-dev/portfolio-server/internal/auth"
	"github.com/sury-dev/portfolio-server/internal/handler"
)

func (s *Server) registerRoutes() {
	s.app.Get("/health", handler.HealthCheck)

	authRepo := auth.NewPostgresRepository(s.db)
	authService := auth.NewService(authRepo, s.cfg.Auth)
	authHandler := auth.NewHandler(authService, s.cfg.Auth)
	auth.RegisterRoutes(s.app, authHandler)
}
