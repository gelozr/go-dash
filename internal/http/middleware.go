package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	himoauth "github.com/gelozr/himo/auth"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/google/uuid"
	"golang.org/x/text/language"

	"github.com/gelozr/go-dash/internal/auth"
	"github.com/gelozr/go-dash/internal/http/response"
	"github.com/gelozr/go-dash/internal/http/validation"
)

func init() {
	limiter.ConfigDefault.LimitReached = func(ctx fiber.Ctx) error {
		return fiber.NewError(http.StatusTooManyRequests, "Too Many Requests")
	}
}

type ctxKey string

var (
	reqIDCtxKey     = ctxKey("req_id")
	reqLocaleCtxKey = ctxKey("req_locale")
	loggerCtxKey    = ctxKey("logger_key")
	userIDCtxKey    = ctxKey("user_id")
)

func AuthMiddleware(g *auth.Guard) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
		if token == "" {
			return fiber.NewError(http.StatusUnauthorized, "missing authorization header")
		}

		userID, err := g.Verify(c.Context(), token)
		if err != nil {
			switch {
			case errors.Is(err, himoauth.ErrTokenExpired):
				return fiber.NewError(http.StatusUnauthorized, "expired token")
			case errors.Is(err, himoauth.ErrInvalidToken):
				return fiber.NewError(http.StatusUnauthorized, "invalid token")
			default:
				return fmt.Errorf("verify token: %w", err)
			}
		}

		c.SetContext(context.WithValue(c.Context(), userIDCtxKey, userID))
		return c.Next()
	}
}

// ValidationResponse maps the validation errors into a JSON response
func ValidationResponse() fiber.Handler {
	return func(c fiber.Ctx) error {
		err := c.Next()

		if err != nil {
			var vErrs validation.Errors
			var jsonErr *json.UnmarshalTypeError

			switch {
			case errors.As(err, &vErrs):
				return c.Status(fiber.StatusUnprocessableEntity).JSON(response.ValidationError{
					Message: "The given data was invalid.",
					Errors:  vErrs,
				})

			case errors.As(err, &jsonErr):
				return c.Status(fiber.StatusUnprocessableEntity).JSON(response.ValidationError{
					Message: "The given data was invalid.",
					Errors: map[string][]string{
						jsonErr.Field: {
							fmt.Sprintf("%s must be a of type %s", jsonErr.Field, jsonErr.Type.String()),
						},
					},
				})
			}
		}

		return err
	}
}

// RequestID extracts the request id from the request header or generates a new one
func RequestID() fiber.Handler {
	return func(c fiber.Ctx) error {
		const hdr = "X-Request-Id"

		id := c.Get(hdr)
		if len(id) == 0 || len(id) > 64 {
			id = uuid.NewString()
		}

		ctx := context.WithValue(c.Context(), reqIDCtxKey, id)
		c.SetContext(ctx)

		c.Request().Header.Set(hdr, id)
		c.Response().Header.Set(hdr, id)

		return c.Next()
	}
}

// localeMatcher lists the locales the validator has translations for; the first is the fallback.
var localeMatcher = language.NewMatcher([]language.Tag{language.English, language.French})

func RequestLocale() fiber.Handler {
	return func(c fiber.Ctx) error {
		// browsers send e.g. "fr-FR,fr;q=0.9", which must resolve to a translator name like "fr"
		tag, _ := language.MatchStrings(localeMatcher, c.Get("Accept-Language"))
		base, _ := tag.Base()

		ctx := context.WithValue(c.Context(), reqLocaleCtxKey, base.String())
		c.SetContext(ctx)

		return c.Next()
	}
}

func rateLimiter(max ...int) fiber.Handler {
	var m int
	if len(max) > 0 {
		m = max[0]
	}

	return limiter.New(limiter.Config{Max: m})
}

func loggerKeyMiddleware(key string) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx := context.WithValue(c.Context(), loggerCtxKey, key)
		c.SetContext(ctx)
		return c.Next()
	}
}
