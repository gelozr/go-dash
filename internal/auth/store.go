package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrRefreshSessionNotFound = errors.New("refresh session not found")
)

type RefreshStore interface {
	Get(context.Context, uuid.UUID) (RefreshSession, error)
	Insert(context.Context, RefreshSession) (RefreshSession, error)
	Update(context.Context, RefreshSession) error
}

type RefreshSession struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}

type UpdateRefreshInput struct {
	UserID    uuid.UUID
	ExpiresAt time.Time
	Used      bool
}
