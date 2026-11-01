package invoice

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/customer"
	"github.com/gelozr/go-dash/internal/db"
	"github.com/gelozr/go-dash/internal/listing"
	"github.com/gelozr/go-dash/internal/logger"
)

var ErrInvalidCustomerID = fmt.Errorf("invalid customer id")

// Customers is what the Invoice module needs from the Customer module.
type Customers interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type Service struct {
	store     Store
	customers Customers
	txm       db.TxManager
	logger    logger.Logger
}

func NewService(store Store, customers Customers, txm db.TxManager, logger logger.Logger) *Service {
	return &Service{
		store:     store,
		customers: customers,
		txm:       txm,
		logger:    logger.With("component", "service.invoice"),
	}
}

func (s *Service) ListWithCustomerInfo(ctx context.Context, sort listing.SortOrder, limit int) ([]WithCustomerInfo, error) {
	out, err := s.store.ListWithCustomerInfo(ctx, sort, limit)
	if err != nil {
		return nil, fmt.Errorf("list invoices: %w", err)
	}

	return out, nil
}

func (s *Service) Search(ctx context.Context, req SearchFilter, page listing.Page) (listing.Result[Invoice], error) {
	invs, total, err := s.store.Search(ctx, req, page)
	if err != nil {
		return listing.Result[Invoice]{}, fmt.Errorf("search invoices: %w", err)
	}
	return listing.NewResult(invs, page, total), nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*Invoice, error) {
	inv, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find invoice: %w", err)
	}
	return inv, nil
}

func (s *Service) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	exists, err := s.store.Exists(ctx, id)
	if err != nil {
		return false, fmt.Errorf("exists invoice: %w", err)
	}
	return exists, nil
}

// Create saves an Invoice for an existing Customer.
func (s *Service) Create(ctx context.Context, inv Invoice) (*Invoice, error) {
	if inv.CustomerID == nil {
		return nil, ErrInvalidCustomerID
	}

	var out *Invoice
	err := s.txm.Do(ctx, func(ctx context.Context) error {
		if err := s.customerMustExist(ctx, *inv.CustomerID); err != nil {
			return err
		}

		var err error
		if out, err = s.store.Insert(ctx, inv); err != nil {
			return fmt.Errorf("save invoice: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

// Update changes an existing Invoice. A new customer_id must point at an existing Customer; null unlinks it.
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateInput) (*Invoice, error) {
	var out *Invoice
	err := s.txm.Do(ctx, func(ctx context.Context) error {
		exists, err := s.Exists(ctx, id)
		if err != nil {
			return err
		}
		if !exists {
			return ErrInvoiceNotFound
		}

		if req.CustomerID.IsPresent && !req.CustomerID.IsNull {
			if err = s.customerMustExist(ctx, req.CustomerID.Val); err != nil {
				return err
			}
		}

		if err = s.store.Update(ctx, id, req); err != nil {
			return fmt.Errorf("update invoice: %w", err)
		}

		out, err = s.Get(ctx, id)
		return err
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Service) customerMustExist(ctx context.Context, id uuid.UUID) error {
	exists, err := s.customers.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("exists customer: %w", err)
	}
	if !exists {
		return customer.ErrCustomerNotFound
	}
	return nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	exists, err := s.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("exists invoice: %w", err)
	}
	if !exists {
		return ErrInvoiceNotFound
	}

	if err = s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete invoice: %w", err)
	}
	return nil
}
