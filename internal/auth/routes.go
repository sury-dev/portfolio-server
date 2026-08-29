package auth

import "github.com/gofiber/fiber/v2"

// RegisterRoutes mounts admin auth endpoints on the given router.
func RegisterRoutes(router fiber.Router, h *Handler) {
	group := router.Group("/admin/auth")
	group.Post("/login", h.Login)
}
