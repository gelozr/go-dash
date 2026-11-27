package request_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

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
