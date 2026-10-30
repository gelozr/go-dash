package customer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gelozr/go-dash/internal/db"
	"github.com/gelozr/go-dash/internal/logger"
)

type customerModel struct {
	ID       uuid.UUID `gorm:"type:char(36);not nullable;unique;primary_key"`
	Name     string    `gorm:"type:varchar(255);not nullable"`
	Email    string    `gorm:"type:varchar(255);not nullable;unique"`
	ImageURL *string   `gorm:"type:varchar(255)"`
}

func (c *customerModel) BeforeCreate(*gorm.DB) (err error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}

	return
}

func (c *customerModel) TableName() string {
	return "customers"
}

func toModel(c Customer) customerModel {
	return customerModel{
		ID:       c.ID,
		Name:     c.Name,
		Email:    c.Email,
		ImageURL: c.ImageURL,
	}
}

func toEntity(c customerModel) Customer {
	return Customer{
		ID:       c.ID,
		Name:     c.Name,
		Email:    c.Email,
		ImageURL: c.ImageURL,
	}
}

type GormStore struct {
	db     *db.DB
	logger logger.Logger
}

// compile‑time check
var _ Store = (*GormStore)(nil)

func NewStore(db *db.DB, log logger.Logger) *GormStore {
	return &GormStore{
		db:     db,
		logger: log.With("component", "store.gorm.customer"),
	}
}

func (s *GormStore) List(ctx context.Context) ([]Customer, error) {
	var models []customerModel

	if err := s.db.Conn(ctx).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("query customers: %w", err)
	}

	customers := make([]Customer, len(models))

	for i, m := range models {
		customers[i] = toEntity(m)
	}

	return customers, nil
}

func (s *GormStore) Find(ctx context.Context, id uuid.UUID) (*Customer, error) {
	var model customerModel

	if err := s.db.Conn(ctx).First(&model, "id = ?", id).Error; err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, ErrCustomerNotFound
		default:
			return nil, fmt.Errorf("query customer: %w", err)
		}
	}

	c := toEntity(model)
	return &c, nil
}

func (s *GormStore) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	tx := s.db.Conn(ctx).Model(&customerModel{}).Where("id = ?", id)

	exists, err := db.RecordExists(tx)
	if err != nil {
		return false, fmt.Errorf("query customer exists: %w", err)
	}

	return exists, nil
}

func (s *GormStore) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	tx := s.db.Conn(ctx).Model(&customerModel{}).Where("email = ?", email)

	exists, err := db.RecordExists(tx)
	if err != nil {
		return false, fmt.Errorf("exists by email: %w", err)
	}

	return exists, nil
}

func (s *GormStore) Insert(ctx context.Context, c Customer) (*Customer, error) {
	model := toModel(c)

	if err := s.db.Conn(ctx).Create(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrEmailAlreadyTaken
		}
		return nil, fmt.Errorf("store customer: %w", err)
	}

	c = toEntity(model)

	return &c, nil
}

func (s *GormStore) SearchWithInvoiceInfo(ctx context.Context, search string) ([]WithInvoiceInfo, error) {
	var out []WithInvoiceInfo

	start := time.Now()

	err := s.db.Conn(ctx).
		Model(&customerModel{}).
		Select(`
            customers.id,
            customers.name,
            customers.email,
            customers.image_url,
            COUNT(invoices.id) AS total_invoices,
            SUM(CASE WHEN invoices.status = 'pending' THEN invoices.amount ELSE 0 END) AS total_pending,
            SUM(CASE WHEN invoices.status = 'paid'    THEN invoices.amount ELSE 0 END) AS total_paid
        `).
		Joins("LEFT JOIN invoices ON customers.id = invoices.customer_id").
		Where("customers.name LIKE @s OR customers.email LIKE @s", sql.Named("s", "%"+search+"%")).
		Group("customers.id, customers.name, customers.email, customers.image_url").
		Order("customers.name").
		Scan(&out).Error

	if err != nil {
		return nil, fmt.Errorf("customer with invoice info query: %w", err)
	}

	s.logger.DebugContext(ctx, "fetch customer with invoice info", "elapsed", time.Since(start).String())

	return out, nil
}
