package slog

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gelozr/go-dash/internal/config"
	"github.com/gelozr/go-dash/internal/logger"
)

type Logger struct {
	logger *slog.Logger
	closer io.Closer
}

var _ logger.Logger = (*Logger)(nil)

func New(cfg *config.Config) (*Logger, error) {
	var (
		w      io.Writer = os.Stdout
		closer io.Closer
	)

	out, err := setOutput(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to set output: %w", err)
	}
	if out != nil {
		w, closer = out, out
	}

	var h slog.Handler

	level := slog.LevelInfo
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	if cfg.LogFormat == "json" {
		h = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level, ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				t := a.Value.Time()
				a.Value = slog.StringValue(t.Format(time.DateTime))
			}
			return a
		}})
	} else {
		h = slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	}

	h = &ctxHandler{base: h}

	sloglogger := slog.New(h)
	// libraries that log through slog's default (e.g. himo's async event handlers) end up in the app log
	slog.SetDefault(sloglogger)

	return &Logger{logger: sloglogger, closer: closer}, nil
}

func (l Logger) With(args ...any) logger.Logger {
	return Logger{logger: l.logger.With(args...)}
}

func (l Logger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l Logger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l Logger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l Logger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l Logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.logger.DebugContext(ctx, msg, args...)
}

func (l Logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.logger.InfoContext(ctx, msg, args...)
}

func (l Logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.logger.WarnContext(ctx, msg, args...)
}

func (l Logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.logger.ErrorContext(ctx, msg, args...)
}

func (l Logger) Close() error {
	if l.closer != nil {
		return l.closer.Close()
	}
	return nil
}

func setOutput(cfg *config.Config) (io.WriteCloser, error) {
	var w io.WriteCloser

	if strings.EqualFold(cfg.LogOutput, "file") {
		if err := os.MkdirAll(filepath.Dir(cfg.LogPath), 0o755); err != nil {
			return nil, fmt.Errorf("create output dir: %w", err)
		}
		f, err := os.OpenFile(cfg.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, fmt.Errorf("open output file: %w", err)
		}
		w = f
	}

	return w, nil
}
