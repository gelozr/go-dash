package invoice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gelozr/go-dash/internal/db"
	"github.com/gelozr/go-dash/internal/listing"
	"github.com/gelozr/go-dash/internal/logger"
	"github.com/gelozr/go-dash/internal/optional"
)

type invoiceModel struct {
	ID         uuid.UUID
	CustomerID optional.Optional[uuid.UUID]
	Amount     float64
	Status     string
	Date       *time.Time
	IsActive   optional.Optional[bool]
}

func (i *invoiceModel) BeforeCreate(*gorm.DB) (err error) {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}

	return
}

func (*invoiceModel) TableName() string {
	return "invoices"
}

func toModel(i Invoice) invoiceModel {
	return invoiceModel{
		ID:         i.ID,
		CustomerID: optional.FromPtr(i.CustomerID),
		Amount:     i.Amount,
		Status:     i.Status,
		Date:       i.Date,
		IsActive:   optional.FromPtr(i.IsActive),
	}
}

func toEntity(i invoiceModel) Invoice {
	return Invoice{
		ID:         i.ID,
		CustomerID: i.CustomerID.Ptr(),
		Amount:     i.Amount,
		Status:     i.Status,
		Date:       i.Date,
		IsActive:   i.IsActive.Ptr(),
	}
}

func toEntities(m []invoiceModel) []Invoice {
	res := make([]Invoice, len(m))
	for i, e := range m {
		res[i] = toEntity(e)
	}
	return res
}

type GormStore struct {
	db     *gorm.DB
	logger logger.Logger
}

var _ Store = (*GormStore)(nil)

func NewStore(db *gorm.DB, logger logger.Logger) *GormStore {
	return &GormStore{
		db:     db,
		logger: logger.With("component", "store.gorm.invoice"),
	}
}

func (s *GormStore) DB(ctx context.Context) *gorm.DB {
	if gormDB, ok := db.FromCtx(ctx); ok {
		return gormDB.WithContext(ctx)
	}
	return s.db.WithContext(ctx)
}

func (s *GormStore) List(ctx context.Context, sort listing.SortOrder) ([]Invoice, error) {
	var models []invoiceModel
	var sortOrder string

	switch sort {
	case listing.SortLatest:
		sortOrder = "DESC"
	default:
		sortOrder = "ASC"
	}

	if err := s.DB(ctx).Order("date " + sortOrder).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("query invoices: %w", err)
	}

	return toEntities(models), nil
}

func (s *GormStore) Search(ctx context.Context, req SearchFilter, page listing.Page) ([]Invoice, int64, error) {
	var sort string
	switch req.Sort {
	case listing.SortLatest:
		sort = "DESC"
	default:
		sort = "ASC"
	}

	q := s.DB(ctx).
		Model(&invoiceModel{}).
		Joins("JOIN customers ON invoices.customer_id = customers.id").
		Where(`
			customers.name LIKE @search OR
			customers.email LIKE @search OR
			CAST(invoices.amount AS CHAR) LIKE @search OR
			CAST(invoices.date AS CHAR) LIKE @search OR
			invoices.status LIKE @search
		`, sql.Named("search", "%"+req.Text+"%")).
		Order("invoices.date " + sort).
		// gorm statements aren't safe to reuse after a finisher; a session clones per call
		Session(&gorm.Session{})

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count invoices: %w", err)
	}

	var models []invoiceModel
	if err := q.Scopes(page.Scope()).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("query invoices: %w", err)
	}

	return toEntities(models), total, nil
}

func (s *GormStore) Find(ctx context.Context, id uuid.UUID) (*Invoice, error) {
	var i invoiceModel

	if err := s.DB(ctx).First(&i, "id = ?", id).Error; err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, ErrInvoiceNotFound
		default:
			return nil, fmt.Errorf("query invoice: %w", err)
		}
	}

	inv := toEntity(i)
	return &inv, nil
}

func (s *GormStore) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	tx := s.DB(ctx).Model(&invoiceModel{}).Where("id = ?", id)

	exists, err := db.RecordExists(tx)
	if err != nil {
		return false, fmt.Errorf("query invoice exists: %w", err)
	}
	return exists, nil
}

func (s *GormStore) Insert(ctx context.Context, i Invoice) (*Invoice, error) {
	invModel := toModel(i)

	if err := s.DB(ctx).Create(&invModel).Error; err != nil {
		return nil, fmt.Errorf("store invoice: %w", err)
	}

	i = toEntity(invModel)
	return &i, nil
}

func (s *GormStore) Update(ctx context.Context, id uuid.UUID, req UpdateInput) error {
	err := s.DB(ctx).
		Model(&invoiceModel{}).
		Where("id = ?", id).
		Updates(req).Error

	if err != nil {
		return fmt.Errorf("update invoice: %w", err)
	}

	return nil
}

func (s *GormStore) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.DB(ctx).Delete(&invoiceModel{}, id).Error; err != nil {
		return fmt.Errorf("delete invoice: %w", err)
	}
	return nil
}

func (s *GormStore) ListWithCustomerInfo(ctx context.Context, sort listing.SortOrder, limit int) ([]WithCustomerInfo, error) {
	var out []WithCustomerInfo
	var sortOrder string

	switch sort {
	case listing.SortLatest:
		sortOrder = "DESC"
	default:
		sortOrder = "ASC"
	}

	err := s.DB(ctx).
		Model(&invoiceModel{}).
		Select(`
			invoices.*,
			customers.name as customer_name,
			customers.email as customer_email,
			customers.image_url as customer_image_url
		`).
		Joins("LEFT JOIN customers ON invoices.customer_id = customers.id").
		Order("date " + sortOrder).
		Limit(limit).
		Find(&out).Error

	if err != nil {
		return nil, fmt.Errorf("query invoices with customer info: %w", err)
	}

	return out, nil
}
