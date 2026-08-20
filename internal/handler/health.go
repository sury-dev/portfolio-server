package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sury-dev/portfolio-server/internal/response"
)

type healthCheckResponse struct {
	Status string `json:"status"`
}

func HealthCheck(c *fiber.Ctx) error {
	return response.JSON(c, fiber.StatusOK, healthCheckResponse{
		Status: "ok",
	})
}
