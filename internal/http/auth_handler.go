package http

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"

	himoauth "github.com/gelozr/himo/auth"

	"github.com/gelozr/go-dash/internal/auth"
	"github.com/gelozr/go-dash/internal/http/request"
	"github.com/gelozr/go-dash/internal/http/response"
	"github.com/gelozr/go-dash/internal/http/validation"
)

type AuthHandler struct {
	auth      *auth.Guard
	validator validation.Validator
}

func NewAuthHandler(auth *auth.Guard, validator validation.Validator) *AuthHandler {
	return &AuthHandler{
		auth:      auth,
		validator: validator,
	}
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req request.Login

	if err := c.Bind().Body(&req); err != nil {
		return fmt.Errorf("parsing login request: %w", err)
	}

	ctx := c.Context()

	if err := h.validator.ValidateStruct(ctx, req); err != nil {
		return fmt.Errorf("login request validation: %w", err)
	}

	token, err := h.auth.Login(ctx, auth.PasswordCredentials{Email: req.Username, Password: req.Password})
	if err != nil {
		if errors.Is(err, himoauth.ErrInvalidCredentials) {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid credentials")
		}
		return fmt.Errorf("login: %w", err)
	}

	return c.JSON(
		response.New(response.ToAccessToken(token)),
	)
}

func (h *AuthHandler) Refresh(c fiber.Ctx) error {
	var req request.Refresh

	if err := c.Bind().Body(&req); err != nil {
		return fmt.Errorf("parsing refresh request: %w", err)
	}

	ctx := c.Context()

	if err := h.validator.ValidateStruct(ctx, req); err != nil {
		return fmt.Errorf("refresh request validation: %w", err)
	}

	token, err := h.auth.Refresh(ctx, req.RefreshToken)
	if err != nil {
		if errors.Is(err, himoauth.ErrInvalidToken) {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid refresh token")
		}
		return fmt.Errorf("refresh access token: %w", err)
	}

	return c.JSON(
		response.New(response.ToAccessToken(token)),
	)
}
