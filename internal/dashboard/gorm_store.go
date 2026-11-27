package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/gelozr/go-dash/internal/db"
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
	var row struct {
		InvoiceCount  int64
		CustomerCount int64
		InvoiceStatus
	}
	start := time.Now()

	err := s.db.Conn(ctx).Raw(`
		SELECT COUNT(*) AS invoice_count,
			(SELECT COUNT(*) FROM customers) AS customer_count,
			COALESCE(SUM(CASE WHEN status = 'paid' THEN amount ELSE 0 END), 0) AS paid,
			COALESCE(SUM(CASE WHEN status = 'pending' THEN amount ELSE 0 END), 0) AS pending
		FROM invoices
	`).Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("query overview: %w", err)
	}

	s.logger.DebugContext(ctx, "fetch overview", "elapsed", time.Since(start).String())

	return &Overview{
		InvoiceCount:  row.InvoiceCount,
		CustomerCount: row.CustomerCount,
		InvoiceStatus: row.InvoiceStatus,
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
