package registry

import (
	"context"
	"sync"
	"testing"
	"time"

	himoevent "github.com/gelozr/himo/event"
	"github.com/google/uuid"

	"github.com/gelozr/go-dash/internal/customer"
	"github.com/gelozr/go-dash/internal/logger"
	"github.com/gelozr/go-dash/internal/mail"
)

type oneCustomer struct {
	customer.Store
	c customer.Customer
}

func (o oneCustomer) Find(_ context.Context, id uuid.UUID) (*customer.Customer, error) {
	if id != o.c.ID {
		return nil, customer.ErrCustomerNotFound
	}
	return &o.c, nil
}

type fakeMailer struct {
	mu   sync.Mutex
	sent []*mail.Message
}

func (f *fakeMailer) Send(_ context.Context, m *mail.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, m)
	return nil
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

func TestCustomerCreatedSendsWelcomeEmail(t *testing.T) {
	c := customer.Customer{ID: uuid.New(), Name: "Ada", Email: "ada@example.com"}
	broker := himoevent.NewBroker()
	mailer := &fakeMailer{}
	subs := RegisterAll(broker, customer.NewService(oneCustomer{c: c}, broker, nopLogger{}), mailer)

	if err := broker.Publish(context.Background(), customer.Created{ID: c.ID}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := subs.Wait(ctx); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	if len(mailer.sent) != 1 || mailer.sent[0].To[0].Address != c.Email {
		t.Fatalf("sent = %+v, want one welcome email to %s", mailer.sent, c.Email)
	}
}
