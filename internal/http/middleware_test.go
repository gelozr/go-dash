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
	claims := jwt.MapClaims{
		"UserID": userID.String(),
		"iss":    "go-dash",
		"exp":    time.Now().Add(time.Minute).Unix(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(key))
	if err != nil {
		t.Fatal(err)
	}
	claims["exp"] = time.Now().Add(-time.Minute).Unix()
	expiredToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(key))
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{JWTHmacKey: key}
	guard := auth.NewGuard(cfg, nil, nil, nil)
	app := fiber.New(fiber.Config{ErrorHandler: fiberErrHandler(cfg, nil)})
	app.Get("/", AuthMiddleware(guard), func(c fiber.Ctx) error {
		if c.Context().Value(userIDCtxKey) != userID {
			return fiber.ErrInternalServerError
		}
		return c.SendStatus(http.StatusNoContent)
	})

	for _, tc := range []struct {
		name      string
		header    string
		status    int
		challenge string
	}{
		{"canonical scheme", "Bearer " + token, http.StatusNoContent, ""},
		{"lowercase scheme", "bearer " + token, http.StatusNoContent, ""},
		{"multiple spaces", "Bearer   " + token, http.StatusNoContent, ""},
		{"missing header", "", http.StatusUnauthorized, `Bearer realm="go-dash"`},
		{"missing scheme", token, http.StatusUnauthorized, `Bearer realm="go-dash"`},
		{"wrong scheme", "Basic " + token, http.StatusUnauthorized, `Bearer realm="go-dash"`},
		{"missing token", "Bearer", http.StatusUnauthorized, `Bearer realm="go-dash"`},
		{"invalid token", "Bearer invalid", http.StatusUnauthorized, `Bearer realm="go-dash", error="invalid_token"`},
		{"expired token", "Bearer " + expiredToken, http.StatusUnauthorized, `Bearer realm="go-dash", error="invalid_token"`},
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
			if challenge := resp.Header.Get(fiber.HeaderWWWAuthenticate); challenge != tc.challenge {
				t.Errorf("WWW-Authenticate = %q, want %q", challenge, tc.challenge)
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
