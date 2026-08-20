package handler

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	"github.com/sury-dev/portfolio-server/internal/response"
)

func NewErrorHandler(log zerolog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		status := fiber.StatusInternalServerError
		message := "internal server error"

		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			status = fiberErr.Code
			message = fiberErr.Message
		}

		event := log.Error()
		if status < 500 {
			event = log.Warn()
		}
		event.
			Err(err).
			Int("status", status).
			Str("method", c.Method()).
			Str("path", c.Path()).
			Msg("request failed")

		return response.Error(c, status, message)
	}
}