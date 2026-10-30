package auth

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	himoauth "github.com/gelozr/himo/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/config"
	"github.com/gelozr/go-dash/internal/hashing"
	"github.com/gelozr/go-dash/internal/logger"
	"github.com/gelozr/go-dash/internal/user"
)

type memRefreshStore struct {
	mu   sync.Mutex
	rows map[uuid.UUID]refreshSession
}

func (m *memRefreshStore) Get(_ context.Context, id uuid.UUID) (refreshSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.rows[id]
	if !ok {
		return r, errRefreshSessionNotFound
	}
	return r, nil
}

func (m *memRefreshStore) Insert(_ context.Context, r refreshSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rows[r.ID] = r
	return nil
}

func (m *memRefreshStore) MarkUsed(_ context.Context, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.rows[id]
	if r.Used {
		return errRefreshSessionUsed
	}
	r.Used = true
	m.rows[id] = r
	return nil
}

type noTx struct{}

func (noTx) Do(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

type usersByEmail map[string]*user.User

func (u usersByEmail) FindByEmail(_ context.Context, email string) (*user.User, error) {
	if usr, ok := u[email]; ok {
		return usr, nil
	}
	return nil, user.ErrUserNotFound
}

type nopLogger struct{}

func (nopLogger) With(...any) logger.Logger                    { return nopLogger{} }
func (nopLogger) Debug(string, ...any)                         {}
func (nopLogger) Info(string, ...any)                          {}
func (nopLogger) Warn(string, ...any)                          {}
func (nopLogger) Error(string, ...any)                         {}
func (nopLogger) DebugContext(context.Context, string, ...any) {}
func (nopLogger) InfoContext(context.Context, string, ...any)  {}
func (nopLogger) WarnContext(context.Context, string, ...any)  {}
func (nopLogger) ErrorContext(context.Context, string, ...any) {}

func newTestGuard(t *testing.T) (*Guard, *user.User, *memRefreshStore) {
	t.Helper()
	hash := hashing.NewManager(&config.Config{})
	pw, err := hash.Hash("secret")
	if err != nil {
		t.Fatal(err)
	}
	u := &user.User{ID: uuid.New(), Email: "a@example.com", Password: pw}
	store := &memRefreshStore{rows: map[uuid.UUID]refreshSession{}}
	g := himoauth.NewTokenGuard(
		newDBUserProvider(user.NewService(usersByEmail{u.Email: u}, nopLogger{}), hash),
		newJWTTokens([]byte("test-key-at-least-32-bytes-long!"), store, noTx{}),
	)
	return g, u, store
}

func TestGuard_LoginRefreshVerify(t *testing.T) {
	ctx := context.Background()
	g, u, _ := newTestGuard(t)

	tok, err := g.Login(ctx, PasswordCredentials{Email: u.Email, Password: "secret"})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if id, err := g.Verify(ctx, tok.AccessToken); err != nil || id != u.ID {
		t.Fatalf("Verify = %v, %v", id, err)
	}

	next, err := g.Refresh(ctx, tok.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if next.RefreshToken == tok.RefreshToken {
		t.Error("refresh token was not rotated")
	}
	if _, err := g.Refresh(ctx, tok.RefreshToken); !errors.Is(err, himoauth.ErrInvalidToken) {
		t.Errorf("reused refresh token: got %v, want ErrInvalidToken", err)
	}
	if id, err := g.Verify(ctx, next.AccessToken); err != nil || id != u.ID {
		t.Errorf("Verify after refresh = %v, %v", id, err)
	}
}

func TestGuard_Rejections(t *testing.T) {
	ctx := context.Background()
	g, u, store := newTestGuard(t)

	for _, c := range []PasswordCredentials{{u.Email, "wrong"}, {"nobody@example.com", "secret"}} {
		if _, err := g.Login(ctx, c); !errors.Is(err, himoauth.ErrInvalidCredentials) {
			t.Errorf("Login(%s/%s): got %v, want ErrInvalidCredentials", c.Email, c.Password, err)
		}
	}

	for _, bad := range []string{"not-a-uuid", uuid.NewString()} {
		if _, err := g.Refresh(ctx, bad); !errors.Is(err, himoauth.ErrInvalidToken) {
			t.Errorf("Refresh(%s): got %v, want ErrInvalidToken", bad, err)
		}
	}

	expired := refreshSession{ID: uuid.New(), UserID: u.ID, ExpiresAt: time.Now().Add(-time.Minute)}
	_ = store.Insert(ctx, expired)
	if _, err := g.Refresh(ctx, expired.ID.String()); !errors.Is(err, himoauth.ErrInvalidToken) {
		t.Errorf("expired refresh session: got %v, want ErrInvalidToken", err)
	}

	if _, err := g.Verify(ctx, "garbage"); !errors.Is(err, himoauth.ErrInvalidToken) {
		t.Errorf("garbage bearer: got %v, want ErrInvalidToken", err)
	}

	old := jwtClaims{UserID: u.ID, RegisteredClaims: jwt.RegisteredClaims{
		Issuer:    jwtIssuer,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
	}}
	oldTok, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, old).SignedString([]byte("test-key-at-least-32-bytes-long!"))
	if _, err := g.Verify(ctx, oldTok); !errors.Is(err, himoauth.ErrTokenExpired) {
		t.Errorf("expired bearer: got %v, want ErrTokenExpired", err)
	}
}
