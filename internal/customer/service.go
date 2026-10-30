package customer

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/event"
	"github.com/gelozr/go-dash/internal/logger"
)

type Service struct {
	store  Store
	event  event.Publisher
	logger logger.Logger
}

func NewService(store Store, evt event.Publisher, log logger.Logger) *Service {
	return &Service{
		store:  store,
		event:  evt,
		logger: log.With("component", "service.customer"),
	}
}

func (s *Service) List(ctx context.Context) ([]Customer, error) {
	customers, err := s.store.List(ctx)

	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}

	return customers, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Customer, error) {
	c, err := s.store.Find(ctx, id)

	if err != nil {
		return nil, fmt.Errorf("find customer: %w", err)
	}

	return c, nil
}

func (s *Service) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	exists, err := s.store.Exists(ctx, id)
	if err != nil {
		return false, fmt.Errorf("exists customer: %w", err)
	}

	return exists, nil
}

func (s *Service) Create(ctx context.Context, c Customer) (*Customer, error) {
	exists, err := s.store.ExistsByEmail(ctx, c.Email)
	if err != nil {
		return nil, fmt.Errorf("exists by email: %w", err)
	}

	if exists {
		s.logger.WarnContext(ctx, "email already taken", "email", c.Email)
		return nil, ErrEmailAlreadyTaken
	}

	var cust *Customer

	if cust, err = s.store.Insert(ctx, c); err != nil {
		return nil, fmt.Errorf("insert customer: %w", err)
	}

	// the customer is already saved; failing here would make the client retry into "email taken"
	if err = s.event.Publish(ctx, Created{ID: cust.ID}); err != nil {
		s.logger.ErrorContext(ctx, "publish customer created", "customer_id", cust.ID, "error", err)
	}

	return cust, nil
}

func (s *Service) SearchWithInvoiceInfo(ctx context.Context, search string) ([]WithInvoiceInfo, error) {
	result, err := s.store.SearchWithInvoiceInfo(ctx, search)
	if err != nil {
		return nil, fmt.Errorf("search with invoice totals: %w", err)
	}

	return result, nil
}
