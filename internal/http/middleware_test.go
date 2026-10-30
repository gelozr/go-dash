package http

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

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
