package auth

import (
	"time"

	himoauth "github.com/gelozr/himo/auth"
	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/config"
	"github.com/gelozr/go-dash/internal/db"
	"github.com/gelozr/go-dash/internal/hashing"
	"github.com/gelozr/go-dash/internal/user"
)

const (
	accessTokenTTL    = 15 * time.Minute
	refreshSessionTTL = 7 * 24 * time.Hour
	jwtIssuer         = "go-dash"
)

// Guard logs users in, refreshes their tokens and verifies bearer tokens.
// Errors match himoauth.ErrInvalidCredentials, ErrTokenExpired or ErrInvalidToken; anything else is internal.
type Guard = himoauth.TokenGuard[PasswordCredentials, uuid.UUID, AccessToken]

type AccessToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type PasswordCredentials struct {
	Email    string
	Password string
}

func NewGuard(cfg *config.Config, users *user.Service, hash hashing.Manager, d *db.DB) *Guard {
	return himoauth.NewTokenGuard(
		newDBUserProvider(users, hash),
		newJWTTokens([]byte(cfg.JWTHmacKey), &gormRefreshStore{db: d}, d),
	)
}
