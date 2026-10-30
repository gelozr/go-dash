package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gelozr/go-dash/internal/db"
	"github.com/gelozr/go-dash/internal/logger"
)

type userModel struct {
	ID       uuid.UUID `gorm:"type:char(36);not nullable;unique;primary_key"`
	Name     string    `gorm:"type:varchar(255);not nullable"`
	Email    string    `gorm:"type:varchar(255);not nullable;unique"`
	Password string    `gorm:"type:text;not nullable"`
}

type GormStore struct {
	db     *db.DB
	logger logger.Logger
}

var _ Store = (*GormStore)(nil)

func NewStore(db *db.DB, log logger.Logger) *GormStore {
	return &GormStore{
		db:     db,
		logger: log.With("component", "store.gorm.user"),
	}
}

func (s *GormStore) FindByEmail(ctx context.Context, email string) (*User, error) {
	var u User

	if err := s.db.Conn(ctx).First(&u, "email = ?", email).Error; err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return nil, ErrUserNotFound
		default:
			return nil, fmt.Errorf("query by email: %w", err)
		}
	}

	return &u, nil
}
