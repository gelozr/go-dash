package auth

import (
	"context"
	"errors"
	"fmt"
	"sync"

	himoauth "github.com/gelozr/himo/auth"
	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/hashing"
	"github.com/gelozr/go-dash/internal/user"
)

type dbUserProvider struct {
	userSvc   *user.Service
	hash      hashing.Hasher
	dummyHash func() (string, error)
}

var _ himoauth.UserProvider[PasswordCredentials, uuid.UUID] = (*dbUserProvider)(nil)

func newDBUserProvider(userSvc *user.Service, hash hashing.Hasher) *dbUserProvider {
	return &dbUserProvider{
		userSvc: userSvc,
		hash:    hash,
		dummyHash: sync.OnceValues(func() (string, error) {
			return hash.Hash("dummy-password-for-timing")
		}),
	}
}

func (p *dbUserProvider) FindByCredentials(ctx context.Context, creds PasswordCredentials) (uuid.UUID, error) {
	u, err := p.userSvc.GetByEmail(ctx, creds.Email)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			// spend the same hashing time as a real check so response time doesn't reveal which emails exist
			if h, hashErr := p.dummyHash(); hashErr == nil {
				_, _ = p.hash.Check(creds.Password, h)
			}
			return uuid.Nil, fmt.Errorf("%w: %w", himoauth.ErrInvalidCredentials, err)
		}
		return uuid.Nil, fmt.Errorf("get user by email: %w", err)
	}

	match, err := p.hash.Check(creds.Password, u.Password)
	if err != nil {
		return uuid.Nil, fmt.Errorf("check password hash: %w", err)
	}
	if !match {
		return uuid.Nil, fmt.Errorf("%w: password incorrect", himoauth.ErrInvalidCredentials)
	}

	return u.ID, nil
}
