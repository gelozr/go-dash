package invoice

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/customer"
	"github.com/gelozr/go-dash/internal/logger"
	"github.com/gelozr/go-dash/internal/optional"
)

type fakeCustomers map[uuid.UUID]bool

func (f fakeCustomers) Exists(_ context.Context, id uuid.UUID) (bool, error) { return f[id], nil }

// fakeStore embeds Store so it only implements what Create and Update reach; anything else panics.
type fakeStore struct {
	Store
	writes *int
}

func (fakeStore) Exists(context.Context, uuid.UUID) (bool, error) { return true, nil }
func (f fakeStore) Insert(_ context.Context, i Invoice) (*Invoice, error) {
	*f.writes++
	return &i, nil
}
func (f fakeStore) Update(context.Context, uuid.UUID, UpdateInput) error {
	*f.writes++
	return nil
}
func (fakeStore) Find(_ context.Context, id uuid.UUID) (*Invoice, error) {
	return &Invoice{ID: id}, nil
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

func newTestService(known uuid.UUID) (*Service, *int) {
	writes := 0
	return NewService(fakeStore{writes: &writes}, fakeCustomers{known: true}, noTx{}, nopLogger{}), &writes
}

func TestCreate_CustomerMustExist(t *testing.T) {
	known, unknown := uuid.New(), uuid.New()
	ctx := context.Background()

	for name, tc := range map[string]struct {
		customerID *uuid.UUID
		wantErr    error
	}{
		"known customer":   {&known, nil},
		"unknown customer": {&unknown, customer.ErrCustomerNotFound},
		"no customer":      {nil, ErrInvalidCustomerID},
	} {
		s, writes := newTestService(known)
		_, err := s.Create(ctx, Invoice{CustomerID: tc.customerID, Status: "paid"})
		if !errors.Is(err, tc.wantErr) || (tc.wantErr == nil && err != nil) {
			t.Errorf("%s: err = %v, want %v", name, err, tc.wantErr)
		}
		if want := btoi(tc.wantErr == nil); *writes != want {
			t.Errorf("%s: %d inserts, want %d", name, *writes, want)
		}
	}
}

func TestUpdate_CustomerMustExist(t *testing.T) {
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
		s, writes := newTestService(known)
		_, err := s.Update(ctx, uuid.New(), UpdateInput{CustomerID: tc.customerID, Status: "paid"})
		if !errors.Is(err, tc.wantErr) || (tc.wantErr == nil && err != nil) {
			t.Errorf("%s: err = %v, want %v", name, err, tc.wantErr)
		}
		if want := btoi(tc.wantErr == nil); *writes != want {
			t.Errorf("%s: %d updates, want %d", name, *writes, want)
		}
	}
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}
