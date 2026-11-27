package request_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	dashhttp "github.com/gelozr/go-dash/internal/http"
	"github.com/gelozr/go-dash/internal/http/request"
	"github.com/gelozr/go-dash/internal/http/validation"
	"github.com/gelozr/go-dash/internal/http/validation/gp"
	"github.com/gelozr/go-dash/internal/invoice"
	"github.com/gelozr/go-dash/internal/logger"
	"github.com/gelozr/go-dash/internal/optional"
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

func TestUpdateInvoiceCustomerAndDateBodyFormats(t *testing.T) {
	const id = "ef7f5245-14ea-4b8b-9869-7f0f86dc7bdc"
	const date = "2026-11-27T12:00:00Z"
	wantID := optional.Of(uuid.MustParse(id))
	wantDate := optional.Of(time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC))
	validator, err := gp.New()
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name, contentType, body string
		status                  int
		customerID              optional.Optional[uuid.UUID]
		date                    optional.Optional[time.Time]
	}{
		{"form", fiber.MIMEApplicationForm, "customer_id=" + id + "&date=" + date, http.StatusOK, wantID, wantDate},
		{"XML", fiber.MIMEApplicationXML, "<UpdateInvoice><CustomerID>" + id + "</CustomerID><Date>" + date + "</Date></UpdateInvoice>", http.StatusOK, wantID, wantDate},
		{"JSON values", fiber.MIMEApplicationJSON, `{"customer_id":"` + id + `","date":"` + date + `"}`, http.StatusOK, wantID, wantDate},
		{"JSON omitted", fiber.MIMEApplicationJSON, `{}`, http.StatusOK, optional.Optional[uuid.UUID]{}, optional.Optional[time.Time]{}},
		{"JSON null", fiber.MIMEApplicationJSON, `{"customer_id":null,"date":null}`, http.StatusOK, optional.FromPtr[uuid.UUID](nil), optional.FromPtr[time.Time](nil)},
		{"invalid form customer", fiber.MIMEApplicationForm, "customer_id=invalid", http.StatusUnprocessableEntity, optional.Optional[uuid.UUID]{}, optional.Optional[time.Time]{}},
		{"invalid XML date", fiber.MIMEApplicationXML, "<UpdateInvoice><Date>invalid</Date></UpdateInvoice>", http.StatusUnprocessableEntity, optional.Optional[uuid.UUID]{}, optional.Optional[time.Time]{}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := &invoiceUpdateStore{}
			log := invoiceUpdateLogger{}
			service := invoice.NewService(store, store, store, log)
			handler := dashhttp.NewInvoiceHandler(service, validator, log)
			app := fiber.New()
			app.Use(dashhttp.ValidationResponse())
			app.Patch("/:id", handler.Update)
			req := httptest.NewRequest(http.MethodPatch, "/"+uuid.NewString(), strings.NewReader(tt.body))
			req.Header.Set("Content-Type", tt.contentType)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.status {
				body, err := io.ReadAll(resp.Body)
				t.Fatalf("HTTP status = %d, want %d. Body %q, read error %v", resp.StatusCode, tt.status, body, err)
			}
			if store.update.CustomerID != tt.customerID || store.update.Date != tt.date {
				t.Fatalf("update fields = (%+v, %+v), want (%+v, %+v)", store.update.CustomerID, store.update.Date, tt.customerID, tt.date)
			}
		})
	}
}

type invoiceUpdateStore struct {
	invoice.Store
	update invoice.UpdateInput
}

func (*invoiceUpdateStore) Exists(context.Context, uuid.UUID) (bool, error) { return true, nil }

func (*invoiceUpdateStore) Find(_ context.Context, id uuid.UUID) (*invoice.Invoice, error) {
	return &invoice.Invoice{ID: id}, nil
}

func (s *invoiceUpdateStore) Update(_ context.Context, _ uuid.UUID, update invoice.UpdateInput) error {
	s.update = update
	return nil
}

func (*invoiceUpdateStore) Do(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type invoiceUpdateLogger struct{ logger.Logger }

func (log invoiceUpdateLogger) With(...any) logger.Logger { return log }
