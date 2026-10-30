package http

import (
	"github.com/gelozr/go-dash/internal/auth"
)

type RouteInitializer struct{}

func SetupFiberRoutes(
	s *FiberServer,
	guard *auth.Guard,
	authH *AuthHandler,
	dashH *DashboardHandler,
	userH *UserHandler,
	custH *CustomerHandler,
	invH *InvoiceHandler,
) RouteInitializer {

	r := s.app.Group("/api")

	// fiber v3 runs a route's handlers in argument order, so middleware must come before the handler

	// auth routes
	ag := r.Group("/auth", loggerKeyMiddleware("http.auth"), rateLimiter(5))
	{
		ag.Post("/login", authH.Login)
		ag.Post("/refresh", authH.Refresh)
	}

	// dashboard routes
	dg := r.Group("/dash", loggerKeyMiddleware("http.dashboard"), AuthMiddleware(guard))
	{
		dg.Get("/overview", dashH.GetOverview)
		dg.Get("/revenues", dashH.GetMonthlyRevenues)
	}

	// user routes
	r.Get("/users/email/:email", loggerKeyMiddleware("http.user"), AuthMiddleware(guard), userH.GetByEmail)

	// customer routes
	cg := r.Group("/customers", loggerKeyMiddleware("http.customer"), AuthMiddleware(guard))
	{
		cg.Get("/", custH.List)
		cg.Get("/filtered", custH.SearchWithInvoiceInfo)
		cg.Get("/:id", custH.Get)
		cg.Post("/", rateLimiter(30), custH.Create)
	}

	// invoice routes
	ig := r.Group("/invoices", loggerKeyMiddleware("http.invoice"), AuthMiddleware(guard))
	{
		ig.Get("/latest", invH.GetLatest)
		ig.Get("/filtered", invH.Search)

		ig.Get("/:id", invH.Get)
		ig.Post("/", rateLimiter(30), invH.Create)
		ig.Patch("/:id", rateLimiter(30), invH.Update)
		ig.Delete("/:id", rateLimiter(30), invH.Delete)
	}

	return RouteInitializer{}
}
