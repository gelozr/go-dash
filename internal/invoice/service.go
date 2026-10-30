package invoice

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/listing"
	"github.com/gelozr/go-dash/internal/logger"
)

var ErrInvalidCustomerID = fmt.Errorf("invalid customer id")

type Service struct {
	store  Store
	logger logger.Logger
}

func NewService(store Store, logger logger.Logger) *Service {
	return &Service{
		store:  store,
		logger: logger.With("component", "service.invoice"),
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

func (s *Service) Create(ctx context.Context, inv Invoice) (*Invoice, error) {
	if inv.CustomerID == nil {
		return nil, ErrInvalidCustomerID
	}

	i, err := s.store.Insert(ctx, inv)
	if err != nil {
		return nil, fmt.Errorf("save invoice: %w", err)
	}

	return i, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateInput) (*Invoice, error) {
	exists, err := s.Exists(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("exists invoice: %w", err)
	}
	if !exists {
		return nil, ErrInvoiceNotFound
	}

	if err = s.store.Update(ctx, id, req); err != nil {
		return nil, fmt.Errorf("update invoice: %w", err)
	}

	inv, err := s.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get invoice: %w", err)
	}

	return inv, nil
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
