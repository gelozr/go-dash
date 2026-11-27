package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/auth"
	"github.com/gelozr/go-dash/internal/config"
)

func TestAuthMiddlewareBearerScheme(t *testing.T) {
	key := uuid.NewString()
	userID := uuid.New()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"UserID": userID.String(),
		"iss":    "go-dash",
		"exp":    time.Now().Add(time.Minute).Unix(),
	}).SignedString([]byte(key))
	if err != nil {
		t.Fatal(err)
	}
	guard := auth.NewGuard(&config.Config{JWTHmacKey: key}, nil, nil, nil)
	app := fiber.New()
	app.Get("/", AuthMiddleware(guard), func(c fiber.Ctx) error {
		if c.Context().Value(userIDCtxKey) != userID {
			return fiber.ErrInternalServerError
		}
		return c.SendStatus(http.StatusNoContent)
	})

	for _, tc := range []struct {
		name   string
		header string
		status int
	}{
		{"canonical scheme", "Bearer " + token, http.StatusNoContent},
		{"lowercase scheme", "bearer " + token, http.StatusNoContent},
		{"multiple spaces", "Bearer   " + token, http.StatusNoContent},
		{"missing header", "", http.StatusUnauthorized},
		{"missing scheme", token, http.StatusUnauthorized},
		{"wrong scheme", "Basic " + token, http.StatusUnauthorized},
		{"missing token", "Bearer", http.StatusUnauthorized},
		{"invalid token", "Bearer invalid", http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.status {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.status)
			}
		})
	}
}

func TestRequestLocale(t *testing.T) {
	app := fiber.New()
	app.Use(RequestLocale())
	app.Get("/", func(c fiber.Ctx) error { return c.SendString(Locale(c.Context())) })

	for header, want := range map[string]string{
		"":                        "en",
		"fr-FR,fr;q=0.9,en;q=0.8": "fr",
		"fr-CA":                   "fr",
		"en-US,en;q=0.9":          "en",
		"de-DE":                   "en",
		"de,fr;q=0.5":             "fr",
	} {
		req := httptest.NewRequest("GET", "/", nil)
		if header != "" {
			req.Header.Set("Accept-Language", header)
		}
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		got, _ := io.ReadAll(resp.Body)
		if string(got) != want {
			t.Errorf("Accept-Language %q: locale %q, want %q", header, got, want)
		}
	}
}
