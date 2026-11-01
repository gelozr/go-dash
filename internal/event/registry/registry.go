package registry

import (
	"context"
	"fmt"

	himoevent "github.com/gelozr/himo/event"

	"github.com/gelozr/go-dash/internal/customer"
	"github.com/gelozr/go-dash/internal/mail"
)

// Subscriptions holds the registered buses so shutdown can wait for their async handlers.
type Subscriptions struct {
	buses []interface{ Wait() }
}

func RegisterAll(broker *himoevent.Broker, custSvc *customer.Service, mailer mail.Mailer) *Subscriptions {
	custCreated := himoevent.NewBus[customer.Created]()
	custCreated.SubscribeAsync(SendWelcomeEmail(custSvc, mailer))
	broker.RegisterBus(custCreated)

	return &Subscriptions{buses: []interface{ Wait() }{custCreated}}
}

// Wait blocks until every async handler started so far has returned, or ctx is done.
func (s *Subscriptions) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		for _, b := range s.buses {
			b.Wait()
		}
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for event handlers: %w", ctx.Err())
	}
}

func SendWelcomeEmail(custSvc *customer.Service, mailer mail.Mailer) himoevent.Handler[customer.Created] {
	return func(ctx context.Context, e customer.Created) error {
		cust, err := custSvc.GetByID(ctx, e.ID)
		if err != nil {
			return fmt.Errorf("SendWelcomeEmail: get customer by id: %w", err)
		}

		m := &mail.Message{
			To: []mail.Address{{
				Name:    cust.Name,
				Address: cust.Email,
			}},
			Subject: "Welcome subject",
			HTML:    "<p>Welcome message</p>",
		}

		if err = mailer.Send(ctx, m); err != nil {
			return fmt.Errorf("SendWelcomeEmail: send welcome email: %w", err)
		}

		return nil
	}
}
