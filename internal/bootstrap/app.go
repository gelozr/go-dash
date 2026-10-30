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

type App struct {
	cfg      *config.Config
	dbCloser io.Closer
	logger   logger.Logger
	server   Server
}

func NewApp(
	cfg *config.Config,
	dbCloser io.Closer,
	logger logger.Logger,
	server Server,
) *App {
	return &App{
		cfg:      cfg,
		logger:   logger,
		dbCloser: dbCloser,
		server:   server,
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

		// shutdown server
		if err := a.server.Shutdown(shutCtx); err != nil {
			return fmt.Errorf("graceful shutdown failed, forcing close: %w", err)
		}
		return nil
	}
}

func (a *App) Logger() logger.Logger {
	return a.logger
}

func (a *App) Close() error {
	var errs []error

	if logCloser, ok := a.logger.(io.Closer); ok {
		errs = append(errs, logCloser.Close())
	}

	errs = append(errs, a.dbCloser.Close())

	return errors.Join(errs...)
}
