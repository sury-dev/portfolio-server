package auth

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/sury-dev/portfolio-server/internal/config"
	"github.com/sury-dev/portfolio-server/internal/response"
)

const (
	cookieAccessToken  = "access_token"
	cookieRefreshToken = "refresh_token"
	cookieAccessPath   = "/admin"
	cookieRefreshPath  = "/admin/auth/refresh"
)

type LoginRequest struct {
	Password string `json:"password"`
}

type LoginResponse struct {
	Message string `json:"message"`
}

type Handler struct {
	svc *Service
	cfg config.AuthConfig
}

func NewHandler(svc *Service, cfg config.AuthConfig) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body")
	}
	if req.Password == "" {
		return response.Error(c, fiber.StatusBadRequest, "password is required")
	}

	result, err := h.svc.Login(c.Context(), req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return response.Error(c, fiber.StatusUnauthorized, "invalid credentials")
		}
		return err
	}

	c.Cookie(&fiber.Cookie{
		Name:     cookieAccessToken,
		Value:    result.AccessToken,
		Path:     cookieAccessPath,
		Expires:  result.AccessExp,
		HTTPOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: "Lax",
	})
	c.Cookie(&fiber.Cookie{
		Name:     cookieRefreshToken,
		Value:    result.RefreshToken,
		Path:     cookieRefreshPath,
		Expires:  result.RefreshExp,
		HTTPOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: "Lax",
	})

	return c.Status(fiber.StatusOK).JSON(LoginResponse{
		Message: "LOGIN Successful",
	})
}
