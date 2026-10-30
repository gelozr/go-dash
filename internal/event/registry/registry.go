package registry

import (
	"context"
	"fmt"

	"github.com/gelozr/go-dash/internal/customer"
	"github.com/gelozr/go-dash/internal/event"
	"github.com/gelozr/go-dash/internal/logger"
	"github.com/gelozr/go-dash/internal/mail"
)

type RegisterInitializer struct{}

func RegisterAll(
	broker *event.Broker,
	custSvc *customer.Service,
	mailer mail.Mailer,
	logger logger.Logger,
) RegisterInitializer {
	log := logger.With("component", "event/registry")

	custCreatedBus := event.NewBus[customer.Created]()
	{
		_ = custCreatedBus.SetAsyncHandler(asyncHandler[customer.Created](log))

		custCreatedBus.SubscribeAsync(SendWelcomeEmail(custSvc, mailer))

		broker.RegisterBus(custCreatedBus)
	}

	return RegisterInitializer{}
}

func asyncHandler[T any](log logger.Logger) func(event.Handler[T]) event.Handler[T] {
	return func(h event.Handler[T]) event.Handler[T] {
		return func(ctx context.Context, e T) error {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.ErrorContext(ctx, fmt.Sprintf("panic: %s", r))
					}
				}()
				if err := h(ctx, e); err != nil {
					log.ErrorContext(ctx, fmt.Sprintf("async handler error: %s", err.Error()))
				}
			}()
			return nil
		}
	}
}

func SendWelcomeEmail(custSvc *customer.Service, mailer mail.Mailer) event.Handler[customer.Created] {
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
