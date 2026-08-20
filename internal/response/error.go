package response

import "github.com/gofiber/fiber/v2"

type ErrorResponse struct {
	Error string `json:"error"` // user facing error
}

func Error(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(ErrorResponse{
		Error: message,
	})
}
