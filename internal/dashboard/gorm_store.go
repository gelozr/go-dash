package dashboard

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/gelozr/go-dash/internal/customer"
	"github.com/gelozr/go-dash/internal/db"
	"github.com/gelozr/go-dash/internal/invoice"
	"github.com/gelozr/go-dash/internal/logger"
)

type revenueModel struct {
	Month   string
	Revenue float64
}

func (m *revenueModel) TableName() string {
	return "revenues"
}

type GormStore struct {
	db     *db.DB
	logger logger.Logger
}

var _ Store = (*GormStore)(nil)

func NewStore(db *db.DB, logger logger.Logger) *GormStore {
	return &GormStore{
		db:     db,
		logger: logger.With("component", "store.gorm.dash"),
	}
}

func (s *GormStore) GetOverview(ctx context.Context) (*Overview, error) {
	var (
		invoiceCount  int64
		customerCount int64
		invoiceStatus InvoiceStatus
	)

	g, egCtx := errgroup.WithContext(ctx)

	start := time.Now()

	g.Go(func() error {
		if err := s.db.Conn(egCtx).Model(&invoice.Invoice{}).Count(&invoiceCount).Error; err != nil {
			return fmt.Errorf("query invoice count: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		if err := s.db.Conn(egCtx).Model(&customer.Customer{}).Count(&customerCount).Error; err != nil {
			return fmt.Errorf("query customer count: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		err := s.db.Conn(egCtx).Model(&invoice.Invoice{}).Select(`
			SUM(CASE WHEN status = 'paid' THEN amount ELSE 0 END) AS "paid",
			SUM(CASE WHEN status = 'pending' THEN amount ELSE 0 END) AS "pending"
		`).Scan(&invoiceStatus).Error

		if err != nil {
			return fmt.Errorf("query invoice status: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("query overview: %w", err)
	}

	s.logger.DebugContext(ctx, "fetch overview", "elapsed", time.Since(start).String())

	return &Overview{
		InvoiceCount:  invoiceCount,
		CustomerCount: customerCount,
		InvoiceStatus: invoiceStatus,
	}, nil
}

func (s *GormStore) ListMonthlyRevenues(ctx context.Context) ([]MonthlyRevenue, error) {
	var models []revenueModel

	if err := s.db.Conn(ctx).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("query monthly revenues: %w", err)
	}

	out := make([]MonthlyRevenue, len(models))
	for i, v := range models {
		out[i] = MonthlyRevenue{
			Month:  v.Month,
			Amount: v.Revenue,
		}
	}

	return out, nil
}
