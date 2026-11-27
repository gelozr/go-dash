package gp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v3"

	"github.com/gelozr/go-dash/internal/http"
	"github.com/gelozr/go-dash/internal/http/request"
	"github.com/gelozr/go-dash/internal/http/response"
	"github.com/gelozr/go-dash/internal/http/validation"
)

func TestValidateStructRejectsInvalidInput(t *testing.T) {
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}

	for _, input := range []any{nil, (*request.CreateInvoice)(nil), "invoice"} {
		var invalid *validator.InvalidValidationError
		if err := v.ValidateStruct(t.Context(), input); !errors.As(err, &invalid) {
			t.Errorf("ValidateStruct(%T) = %v, want invalid validation error", input, err)
		}
	}
}

func TestInvoiceStatusMustBeKnown(t *testing.T) {
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	valid := request.CreateInvoice{CustomerID: "x", Amount: 1, Date: "2026-10-31T00:00:00Z"}

	for status, ok := range map[string]bool{"pending": true, "paid": true, "Paid": false, "done": false} {
		req := valid
		req.Status = status
		err := v.ValidateStruct(ctx, req)

		var errs validation.Errors
		hasStatusErr := errors.As(err, &errs) && len(errs["status"]) > 0
		if hasStatusErr == ok {
			t.Errorf("status %q: validation error = %v, want ok=%v", status, err, ok)
		}
	}
}

func TestInvoiceDateValidationUsesRequestLocale(t *testing.T) {
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	app.Use(http.RequestLocale(), http.ValidationResponse())
	app.Post("/invoices", func(c fiber.Ctx) error {
		return v.ValidateStruct(c.Context(), request.CreateInvoice{
			CustomerID: "bc2dbfcc-b2ba-4dac-8f41-5bea194e2d20",
			Amount:     1,
			Status:     "pending",
			Date:       "2026-11-27",
		})
	})

	for locale, want := range map[string]string{
		"en-US": "date must be a valid RFC-3339 date-time",
		"fr-FR": "date doit être une date et une heure valides au format RFC 3339",
	} {
		t.Run(locale, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/invoices", nil)
			req.Header.Set("Accept-Language", locale)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != fiber.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want %d", resp.StatusCode, fiber.StatusUnprocessableEntity)
			}
			var body response.ValidationError
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if got := body.Errors["date"]; len(got) != 1 || got[0] != want {
				t.Errorf("date errors = %q, want %q", got, want)
			}
		})
	}
}
