package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gelozr/go-dash/internal/config"
	"github.com/gelozr/go-dash/internal/logger"
)

type Server interface {
	Serve() error
	Shutdown(context.Context) error
}

// Waiter is anything shutdown must wait for after the server stops, e.g. async event handlers.
type Waiter interface {
	Wait(context.Context) error
}

type App struct {
	cfg      *config.Config
	dbCloser io.Closer
	logger   logger.Logger
	server   Server
	events   Waiter
}

func NewApp(
	cfg *config.Config,
	dbCloser io.Closer,
	logger logger.Logger,
	server Server,
	events Waiter,
) *App {
	return &App{
		cfg:      cfg,
		logger:   logger,
		dbCloser: dbCloser,
		server:   server,
		events:   events,
	}
}

func (a *App) Run() error {
	// handle signals for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srvErr := make(chan error, 1)

	// run server in a goroutine
	go func() {
		if err := a.server.Serve(); err != nil {
			srvErr <- fmt.Errorf("start server: %w", err)
		}
	}()

	select {
	case err := <-srvErr:
		return err
	case <-ctx.Done():
		a.logger.With("component", "server").Info("shutdown signal received")

		// give other goroutines time to finish
		shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// server first, so no request can publish an event after we start waiting on handlers
		shutdownErr := a.server.Shutdown(shutCtx)
		if shutdownErr != nil {
			shutdownErr = fmt.Errorf("graceful shutdown failed, forcing close: %w", shutdownErr)
		}
		return errors.Join(shutdownErr, a.events.Wait(shutCtx))
	}
}

func (a *App) Logger() logger.Logger {
	return a.logger
}

func (a *App) Close() error {
	var logErr error
	if logCloser, ok := a.logger.(io.Closer); ok {
		logErr = logCloser.Close()
	}
	return errors.Join(logErr, a.dbCloser.Close())
}
