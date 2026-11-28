package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"

	"github.com/gelozr/go-dash/internal/config"
	"github.com/gelozr/go-dash/internal/http/response"
	"github.com/gelozr/go-dash/internal/logger"
)

type FiberServer struct {
	cfg *config.Config
	app *fiber.App
}

func NewFiberServer(
	cfg *config.Config,
	logger logger.Logger,
) *FiberServer {
	app := fiber.New(fiber.Config{
		ErrorHandler: fiberErrHandler(cfg, logger),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	})

	// global middlewares
	// app.Use(logger.New())
	app.Use(recover.New(recover.Config{}))
	app.Use(RequestLocale())
	app.Use(RequestID())
	app.Use(cors.New(cors.Config{}))
	app.Use(limiter.New(limiter.Config{Max: 60}))
	app.Use(ValidationResponse())

	return &FiberServer{cfg, app}
}

func (f *FiberServer) Serve() error {
	if err := f.app.Listen(":" + f.cfg.AppPort); err != nil {
		return fmt.Errorf("fiber listen: %w", err)
	}
	return nil
}

func (f *FiberServer) Shutdown(ctx context.Context) error {
	if err := f.app.ShutdownWithContext(ctx); err != nil {
		return fmt.Errorf("fiber shutdown: %w", err)
	}
	return nil
}

func fiberErrHandler(cfg *config.Config, logger logger.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		message := "Internal Server Error"

		var e *response.AppError
		var fe *fiber.Error
		var jsonSynErr *json.SyntaxError
		var bindErr *fiber.BindError

		switch {
		case errors.As(err, &e):
			code = e.Code
			message = e.Message
		case errors.As(err, &fe) && fe.Code != code:
			code = fe.Code
			message = fe.Message
		case errors.As(err, &jsonSynErr):
			code = fiber.StatusBadRequest
			message = "Invalid JSON"
		case errors.As(err, &bindErr) && bindErr.Source == fiber.BindSourceBody:
			code = fiber.StatusBadRequest
			message = "Invalid request body"
		}

		if code >= fiber.StatusInternalServerError {
			l := "http"
			if loggerKey, ok := c.Context().Value(loggerCtxKey).(string); ok {
				l = loggerKey
			}
			logger.With("component", l).ErrorContext(c.Context(), message, "error", err.Error())
		}

		var resp struct {
			Message string `json:"message"`
			Error   string `json:"error,omitempty"`
		}
		resp.Message = message

		if cfg.AppDebug {
			resp.Error = err.Error()
		}

		return c.Status(code).JSON(resp)
	}
}
