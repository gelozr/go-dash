package dashboard

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/gelozr/go-dash/internal/logger"
)

type Service struct {
	store  Store
	logger logger.Logger
}

func NewService(store Store, log logger.Logger) *Service {
	return &Service{
		store:  store,
		logger: log.With("component", "service.dashboard"),
	}
}

func (s *Service) GetOverview(ctx context.Context) (*Overview, error) {
	o, err := s.store.GetOverview(ctx)
	if err != nil {
		return nil, fmt.Errorf("retrieve overview: %w", err)
	}
	return o, nil
}

func (s *Service) GetMonthlyRevenues(ctx context.Context) ([]MonthlyRevenue, error) {
	revs, err := s.store.ListMonthlyRevenues(ctx)
	if err != nil {
		return nil, fmt.Errorf("list monthly revenues: %w", err)
	}

	slices.SortFunc(revs, func(a, b MonthlyRevenue) int {
		return cmp.Compare(monthOrder[a.Month], monthOrder[b.Month])
	})

	return revs, nil
}
