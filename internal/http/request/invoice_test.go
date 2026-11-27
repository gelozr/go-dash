package request_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/gelozr/go-dash/internal/http/request"
	"github.com/gelozr/go-dash/internal/http/validation"
	"github.com/gelozr/go-dash/internal/http/validation/gp"
	"github.com/gelozr/go-dash/internal/invoice"
)

func TestUpdateInvoiceStatus(t *testing.T) {
	validator, err := gp.New()
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: "u:p@tcp(127.0.0.1:1)/test", SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name       string
		body       string
		wantStatus string
		wantError  bool
	}{
		{name: "omitted", body: `{"amount":1}`},
		{name: "pending", body: `{"amount":1,"status":"pending"}`, wantStatus: "pending"},
		{name: "paid", body: `{"amount":1,"status":"paid"}`, wantStatus: "paid"},
		{name: "empty", body: `{"amount":1,"status":""}`, wantError: true},
		{name: "null", body: `{"amount":1,"status":null}`, wantError: true},
		{name: "unknown", body: `{"amount":1,"status":"done"}`, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var req request.UpdateInvoice
			if err := json.Unmarshal([]byte(tt.body), &req); err != nil {
				t.Fatal(err)
			}
			if err := validator.ValidateStruct(t.Context(), req); tt.wantError {
				var fields validation.Errors
				if !errors.As(err, &fields) || len(fields["status"]) == 0 {
					t.Fatalf("expected a status validation error, got %v", err)
				}
				return
			} else if err != nil {
				t.Fatal(err)
			}

			update, err := req.ToDTO()
			if err != nil {
				t.Fatal(err)
			}
			if update.Status != tt.wantStatus {
				t.Fatalf("status = %q, want %q", update.Status, tt.wantStatus)
			}
			tx := db.Model(&invoice.Invoice{}).Where("id = ?", "test").Updates(update)
			if tx.Error != nil {
				t.Fatal(tx.Error)
			}
			sql := tx.Statement.SQL.String()
			if !strings.Contains(sql, "`amount`=") || strings.Contains(sql, "`status`=") != (tt.wantStatus != "") {
				t.Fatalf("unexpected invoice update SQL: %s", sql)
			}
		})
	}
}

func TestUpdateInvoiceStatusBodyFormats(t *testing.T) {
	validator, err := gp.New()
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	app.Patch("/", func(c fiber.Ctx) error {
		var req request.UpdateInvoice
		if err := c.Bind().Body(&req); err != nil {
			return err
		}
		if err := validator.ValidateStruct(c.Context(), req); err != nil {
			return err
		}
		update, err := req.ToDTO()
		if err != nil {
			return err
		}
		return c.SendString(update.Status)
	})

	for _, tt := range []struct {
		name        string
		contentType string
		body        string
		wantStatus  string
	}{
		{"form status", fiber.MIMEApplicationForm, "amount=1&status=paid", "paid"},
		{"form omitted", fiber.MIMEApplicationForm, "amount=1", ""},
		{"XML status", fiber.MIMEApplicationXML, "<UpdateInvoice><Amount>1</Amount><Status>paid</Status></UpdateInvoice>", "paid"},
		{"XML omitted", fiber.MIMEApplicationXML, "<UpdateInvoice><Amount>1</Amount></UpdateInvoice>", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("PATCH", "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != 200 || string(body) != tt.wantStatus {
				t.Fatalf("got HTTP %d with body %q, want HTTP 200 with body %q", resp.StatusCode, body, tt.wantStatus)
			}
		})
	}
}

func TestUpdateInvoiceAmount(t *testing.T) {
	validator, err := gp.New()
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN: "u:p@tcp(127.0.0.1:1)/test", SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name       string
		body       string
		wantUpdate bool
		wantError  bool
	}{
		{name: "omitted", body: `{"status":"paid"}`},
		{name: "null", body: `{"status":"paid","amount":null}`},
		{name: "zero", body: `{"status":"paid","amount":0}`, wantUpdate: true},
		{name: "positive", body: `{"status":"paid","amount":1.5}`, wantUpdate: true},
		{name: "negative", body: `{"status":"paid","amount":-1}`, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var req request.UpdateInvoice
			if err := json.Unmarshal([]byte(tt.body), &req); err != nil {
				t.Fatal(err)
			}
			if err := validator.ValidateStruct(t.Context(), req); tt.wantError {
				if err == nil {
					t.Fatal("expected validation to reject negative amount")
				}
				return
			} else if err != nil {
				t.Fatal(err)
			}

			update, err := req.ToDTO()
			if err != nil {
				t.Fatal(err)
			}
			tx := db.Model(&invoice.Invoice{}).Where("id = ?", "test").Updates(update)
			if tx.Error != nil {
				t.Fatal(tx.Error)
			}
			sql := tx.Statement.SQL.String()
			if got := strings.Contains(sql, "`amount`="); got != tt.wantUpdate {
				t.Fatalf("amount update = %v, want %v, SQL = %s", got, tt.wantUpdate, sql)
			}
		})
	}
}
