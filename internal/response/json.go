package response

import "github.com/gofiber/fiber/v2"

func JSON(c *fiber.Ctx, status int, data interface{}) error {
	return c.Status(status).JSON(fiber.Map{
			"message": "success",
			"data": data,
		})
}