package bootstrap

import (
	"github.com/google/wire"

	"github.com/gelozr/go-dash/internal/app"
	"github.com/gelozr/go-dash/internal/auth"
	"github.com/gelozr/go-dash/internal/config"
	"github.com/gelozr/go-dash/internal/customer"
	"github.com/gelozr/go-dash/internal/dashboard"
	"github.com/gelozr/go-dash/internal/db"
	"github.com/gelozr/go-dash/internal/event"
	"github.com/gelozr/go-dash/internal/event/registry"
	"github.com/gelozr/go-dash/internal/hashing"
	"github.com/gelozr/go-dash/internal/http"
	"github.com/gelozr/go-dash/internal/http/validation"
	"github.com/gelozr/go-dash/internal/http/validation/gp"
	"github.com/gelozr/go-dash/internal/invoice"
	"github.com/gelozr/go-dash/internal/logger"
	"github.com/gelozr/go-dash/internal/logger/slog"
	"github.com/gelozr/go-dash/internal/mail"
	"github.com/gelozr/go-dash/internal/user"
)

var AppServiceProviders = wire.NewSet(
	// CONFIG
	config.Load,

	// LOGGER
	slog.New,
	wire.Bind(new(logger.Logger), new(*slog.Logger)),

	// DB
	db.Open,
	wire.Bind(new(db.TxManager), new(*db.DB)),

	// EVENT
	event.NewBroker,
	wire.Bind(new(event.Publisher), new(*event.Broker)),
	registry.RegisterAll,

	// VALIDATOR
	gp.New,
	wire.Bind(new(validation.Validator), new(*gp.Validator)),

	// HASHING
	hashing.NewManager,

	// MAIL
	mail.NewManager,
	wire.Bind(new(mail.Mailer), new(mail.Manager)),

	// AUTH
	auth.NewGuard,

	// STORE & SERVICES
	customer.NewStore,
	wire.Bind(new(customer.Store), new(*customer.GormStore)),
	customer.NewService,

	user.NewStore,
	wire.Bind(new(user.Store), new(*user.GormStore)),
	user.NewService,

	dashboard.NewStore,
	wire.Bind(new(dashboard.Store), new(*dashboard.GormStore)),
	dashboard.NewService,

	invoice.NewStore,
	wire.Bind(new(invoice.Store), new(*invoice.GormStore)),
	invoice.NewService,

	// USE CASES
	app.NewCreateInvoice,
)

var HTTPProviders = wire.NewSet(
	// HANDLERS
	http.NewAuthHandler,
	http.NewDashboardHandler,
	http.NewUserHandler,
	http.NewCustomerHandler,
	http.NewInvoiceHandler,

	// ENGINE
	http.NewFiberServer,

	// ROUTES
	http.SetupFiberRoutes,
)

func AppProvider(
	cfg *config.Config,
	database *db.DB,
	logger logger.Logger,
	fiberServer *http.FiberServer,
	_ registry.RegisterInitializer,
	_ http.RouteInitializer,
) *App {
	return NewApp(cfg, database, logger, fiberServer)
}
