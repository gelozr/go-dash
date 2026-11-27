package gp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
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

func TestCreateCustomerFieldLengths(t *testing.T) {
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	imageURL := "https://example.com/" + strings.Repeat("a", 235)
	req := request.CreateCustomer{
		Name:     strings.Repeat("é", 255),
		Email:    strings.Repeat("a", 57) + "@" + strings.Repeat(strings.Repeat("b", 62)+".", 3) + "test.org",
		ImageURL: &imageURL,
	}
	if err := v.ValidateStruct(t.Context(), req); err != nil {
		t.Fatalf("255-character fields rejected: %v", err)
	}
	req.ImageURL = nil
	if err := v.ValidateStruct(t.Context(), req); err != nil {
		t.Fatalf("omitted image URL rejected: %v", err)
	}

	req.Name += "é"
	req.Email = "a" + req.Email
	imageURL += "a"
	req.ImageURL = &imageURL
	var fields validation.Errors
	if err := v.ValidateStruct(t.Context(), req); !errors.As(err, &fields) {
		t.Fatalf("expected length validation errors, got %v", err)
	}
	for _, name := range []string{"name", "email", "image_url"} {
		if len(fields[name]) == 0 {
			t.Errorf("expected a length validation error for %s, got %v", name, fields)
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

func TestCreateInvoiceRejectsNegativeAmount(t *testing.T) {
	v, err := New()
	if err != nil {
		t.Fatal(err)
	}
	req := request.CreateInvoice{
		CustomerID: "bc2dbfcc-b2ba-4dac-8f41-5bea194e2d20",
		Amount:     -0.01,
		Status:     "pending",
		Date:       "2026-10-31T00:00:00Z",
	}
	var errs validation.Errors
	if err := v.ValidateStruct(t.Context(), req); !errors.As(err, &errs) || len(errs["amount"]) == 0 {
		t.Fatalf("validation error = %v, want an error for amount", err)
	}

	req.Amount = 0.01
	if err := v.ValidateStruct(t.Context(), req); err != nil {
		t.Fatalf("positive amount rejected: %v", err)
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
