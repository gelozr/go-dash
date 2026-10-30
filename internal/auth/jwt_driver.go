package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	himoauth "github.com/gelozr/himo/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/db"
)

type jwtClaims struct {
	UserID uuid.UUID
	jwt.RegisteredClaims
}

// jwtTokens issues a short-lived JWT plus a single-use refresh token backed by a Refresh Session row.
type jwtTokens struct {
	hmacKey []byte
	store   refreshStore
	txm     db.TxManager
}

var _ himoauth.TokenProvider[uuid.UUID, AccessToken] = (*jwtTokens)(nil)

func newJWTTokens(hmacKey []byte, store refreshStore, txm db.TxManager) *jwtTokens {
	return &jwtTokens{hmacKey: hmacKey, store: store, txm: txm}
}

func (t *jwtTokens) Issue(ctx context.Context, uid uuid.UUID) (AccessToken, error) {
	sess, err := t.newRefreshSession(ctx, uid)
	if err != nil {
		return AccessToken{}, err
	}
	return t.accessToken(sess)
}

func (t *jwtTokens) Refresh(ctx context.Context, refreshToken string) (AccessToken, error) {
	id, err := uuid.Parse(refreshToken)
	if err != nil {
		return AccessToken{}, fmt.Errorf("%w: malformed refresh token", himoauth.ErrInvalidToken)
	}

	var next refreshSession
	// one transaction, so a failed insert doesn't leave the user with a used token and no new one
	err = t.txm.Do(ctx, func(ctx context.Context) error {
		curr, err := t.store.Get(ctx, id)
		if err != nil {
			return err
		}
		if curr.ExpiresAt.Before(time.Now()) {
			return errRefreshSessionExpired
		}
		if err = t.store.MarkUsed(ctx, id); err != nil {
			return err
		}
		next, err = t.newRefreshSession(ctx, curr.UserID)
		return err
	})
	if err != nil {
		return AccessToken{}, err
	}

	return t.accessToken(next)
}

func (t *jwtTokens) Verify(_ context.Context, bearer string) (uuid.UUID, error) {
	tok, err := jwt.ParseWithClaims(bearer, &jwtClaims{}, func(*jwt.Token) (any, error) {
		return t.hmacKey, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(jwtIssuer))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, fmt.Errorf("%w: %w", himoauth.ErrTokenExpired, err)
		}
		// the key func never fails, so every other parse error is a bad token
		return uuid.Nil, fmt.Errorf("%w: %w", himoauth.ErrInvalidToken, err)
	}

	return tok.Claims.(*jwtClaims).UserID, nil
}

func (t *jwtTokens) newRefreshSession(ctx context.Context, uid uuid.UUID) (refreshSession, error) {
	now := time.Now()
	sess := refreshSession{
		ID:        uuid.New(),
		UserID:    uid,
		ExpiresAt: now.Add(refreshSessionTTL),
		CreatedAt: now,
	}
	if err := t.store.Insert(ctx, sess); err != nil {
		return refreshSession{}, fmt.Errorf("insert refresh session: %w", err)
	}
	return sess, nil
}

func (t *jwtTokens) accessToken(sess refreshSession) (AccessToken, error) {
	now := time.Now()
	claims := jwtClaims{
		UserID: sess.UserID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    jwtIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.hmacKey)
	if err != nil {
		return AccessToken{}, fmt.Errorf("sign jwt: %w", err)
	}

	return AccessToken{
		AccessToken:  signed,
		RefreshToken: sess.ID.String(),
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	}, nil
}
