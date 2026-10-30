package auth

import (
	"context"
	"fmt"
	"time"

	himoauth "github.com/gelozr/himo/auth"
	"github.com/google/uuid"
)

var (
	errRefreshSessionNotFound = fmt.Errorf("%w: refresh session not found", himoauth.ErrInvalidToken)
	errRefreshSessionUsed     = fmt.Errorf("%w: refresh session already used", himoauth.ErrInvalidToken)
	errRefreshSessionExpired  = fmt.Errorf("%w: refresh session expired", himoauth.ErrInvalidToken)
)

// refreshStore is an internal seam so tests can run without a database.
type refreshStore interface {
	Get(context.Context, uuid.UUID) (refreshSession, error)
	Insert(context.Context, refreshSession) error
	// MarkUsed returns errRefreshSessionUsed if the session was already used.
	MarkUsed(context.Context, uuid.UUID) error
}

type refreshSession struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	Used      bool
	CreatedAt time.Time
}

func (refreshSession) TableName() string {
	return "refresh_sessions"
}
