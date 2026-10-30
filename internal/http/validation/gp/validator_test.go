package gp

import (
	"context"
	"errors"
	"testing"

	"github.com/gelozr/go-dash/internal/http/request"
	"github.com/gelozr/go-dash/internal/http/validation"
)

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
