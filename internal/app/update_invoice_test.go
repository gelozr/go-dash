package app

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/customer"
	"github.com/gelozr/go-dash/internal/invoice"
	"github.com/gelozr/go-dash/internal/logger"
	"github.com/gelozr/go-dash/internal/optional"
)

// The fakes embed the store interfaces so they only implement what UpdateInvoice reaches; anything else panics.
type fakeCustomers struct {
	customer.Store
	ids map[uuid.UUID]bool
}

func (f fakeCustomers) Exists(_ context.Context, id uuid.UUID) (bool, error) { return f.ids[id], nil }

type fakeInvoices struct {
	invoice.Store
	updated *bool
}

func (fakeInvoices) Exists(context.Context, uuid.UUID) (bool, error) { return true, nil }
func (f fakeInvoices) Update(context.Context, uuid.UUID, invoice.UpdateInput) error {
	*f.updated = true
	return nil
}
func (fakeInvoices) Find(_ context.Context, id uuid.UUID) (*invoice.Invoice, error) {
	return &invoice.Invoice{ID: id}, nil
}

type noTx struct{}

func (noTx) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type nopLogger struct{}

func (nopLogger) With(...any) logger.Logger                    { return nopLogger{} }
func (nopLogger) Debug(string, ...any)                         {}
func (nopLogger) Info(string, ...any)                          {}
func (nopLogger) Warn(string, ...any)                          {}
func (nopLogger) Error(string, ...any)                         {}
func (nopLogger) DebugContext(context.Context, string, ...any) {}
func (nopLogger) InfoContext(context.Context, string, ...any)  {}
func (nopLogger) WarnContext(context.Context, string, ...any)  {}
func (nopLogger) ErrorContext(context.Context, string, ...any) {}

func TestUpdateInvoice_CustomerMustExist(t *testing.T) {
	known := uuid.New()
	ctx := context.Background()

	for name, tc := range map[string]struct {
		customerID optional.Optional[uuid.UUID]
		wantErr    error
	}{
		"unknown customer": {optional.Of(uuid.New()), customer.ErrCustomerNotFound},
		"known customer":   {optional.Of(known), nil},
		"customer omitted": {optional.Optional[uuid.UUID]{}, nil},
		"customer cleared": {optional.Optional[uuid.UUID]{IsPresent: true, IsNull: true}, nil},
	} {
		updated := false
		u := NewUpdateInvoice(
			customer.NewService(fakeCustomers{ids: map[uuid.UUID]bool{known: true}}, nil, nopLogger{}),
			invoice.NewService(fakeInvoices{updated: &updated}, nopLogger{}),
			noTx{},
		)

		_, err := u.Execute(ctx, uuid.New(), invoice.UpdateInput{CustomerID: tc.customerID, Status: "paid"})
		if !errors.Is(err, tc.wantErr) || (tc.wantErr == nil && err != nil) {
			t.Errorf("%s: err = %v, want %v", name, err, tc.wantErr)
		}
		if updated != (tc.wantErr == nil) {
			t.Errorf("%s: updated = %v, want %v", name, updated, tc.wantErr == nil)
		}
	}
}
