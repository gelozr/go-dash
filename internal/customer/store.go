package customer

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var ErrCustomerNotFound = errors.New("customer not found")
var ErrEmailAlreadyTaken = errors.New("email already exists")

type Store interface {
	List(ctx context.Context) ([]Customer, error)
	Find(ctx context.Context, id uuid.UUID) (*Customer, error)
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	Insert(ctx context.Context, c Customer) (*Customer, error)
	SearchWithInvoiceInfo(ctx context.Context, search string) ([]WithInvoiceInfo, error)
}

type WithInvoiceInfo struct {
	ID            uuid.UUID
	Name          string
	Email         string
	ImageURL      *string
	TotalInvoices int64
	TotalPending  float64
	TotalPaid     float64
}
