package user

import (
	"context"
	"errors"
)

var ErrUserNotFound = errors.New("user not found")

type Store interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
}
