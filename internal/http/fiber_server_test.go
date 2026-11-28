package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/gelozr/go-dash/internal/config"
	"github.com/gelozr/go-dash/internal/logger"
)

func TestCreateCustomerBodyErrors(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: fiberErrHandler(&config.Config{}, bodyErrorLogger{})})
	handler := &CustomerHandler{}
	app.Post("/api/customers", handler.Create)

	for _, tc := range []struct {
		name        string
		contentType string
		body        string
		status      int
		message     string
	}{
		{"wrong JSON field type", fiber.MIMEApplicationJSON, `{"name":123}`, http.StatusBadRequest, "Invalid request body"},
		{"malformed XML", fiber.MIMEApplicationXML, "<CreateCustomer><Name>", http.StatusBadRequest, "Invalid request body"},
		{"malformed JSON syntax", fiber.MIMEApplicationJSON, `{"name":}`, http.StatusBadRequest, "Invalid JSON"},
		{"unsupported content type", fiber.MIMETextPlain, "name=Ada", http.StatusUnprocessableEntity, "Unprocessable Entity"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/customers", strings.NewReader(tc.body))
			req.Header.Set(fiber.HeaderContentType, tc.contentType)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.status {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.status)
			}
			var body struct {
				Message string `json:"message"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Message != tc.message {
				t.Errorf("message = %q, want %q", body.Message, tc.message)
			}
		})
	}
}

type bodyErrorLogger struct{ logger.Logger }

func (log bodyErrorLogger) With(...any) logger.Logger { return log }

func (bodyErrorLogger) ErrorContext(context.Context, string, ...any) {}
