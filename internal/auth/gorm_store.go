package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/gelozr/go-dash/internal/db"
)

type gormRefreshStore struct {
	db *gorm.DB
}

var _ refreshStore = (*gormRefreshStore)(nil)

func (s *gormRefreshStore) DB(ctx context.Context) *gorm.DB {
	if gormDB, ok := db.FromCtx(ctx); ok {
		return gormDB.WithContext(ctx)
	}
	return s.db.WithContext(ctx)
}

func (s *gormRefreshStore) Get(ctx context.Context, id uuid.UUID) (refreshSession, error) {
	var r refreshSession

	if err := s.DB(ctx).First(&r, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r, errRefreshSessionNotFound
		}
		return r, fmt.Errorf("query refresh session: %w", err)
	}

	return r, nil
}

func (s *gormRefreshStore) Insert(ctx context.Context, r refreshSession) error {
	if err := s.DB(ctx).Create(&r).Error; err != nil {
		return fmt.Errorf("create refresh session: %w", err)
	}
	return nil
}

func (s *gormRefreshStore) MarkUsed(ctx context.Context, id uuid.UUID) error {
	// conditional update so two concurrent refreshes can't both claim the same session
	res := s.DB(ctx).Model(&refreshSession{}).Where("id = ? AND used = ?", id, false).Update("used", true)
	if res.Error != nil {
		return fmt.Errorf("mark refresh session used: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return errRefreshSessionUsed
	}
	return nil
}
