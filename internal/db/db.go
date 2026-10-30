package db

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/gelozr/go-dash/internal/config"
	"github.com/gelozr/go-dash/internal/logger"
)

type ctxKey string

var dbTxKey = ctxKey("db_tx_key")

// DB is the only way stores reach the database, so the tx-from-context rule lives in one place.
type DB struct {
	gorm *gorm.DB
}

// Conn returns the transaction stored in ctx by Do, or the base handle, bound to ctx.
// Don't run concurrent queries on a ctx that holds a transaction: it's a single connection.
func (d *DB) Conn(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(dbTxKey).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return d.gorm.WithContext(ctx)
}

// Do runs fn in a transaction. Inside another Do it joins the outer transaction through a savepoint.
func (d *DB) Do(ctx context.Context, fn func(context.Context) error) error {
	return d.Conn(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, dbTxKey, tx))
	})
}

func (d *DB) Close() error {
	sqlDB, err := d.gorm.DB()
	if err != nil {
		return fmt.Errorf("get sql db: %w", err)
	}
	return sqlDB.Close()
}

func Open(cfg *config.Config, log logger.Logger) (*DB, error) {
	wd, _ := os.Getwd()

	logLevel := gormlogger.Warn
	l := &dbLogger{Logger: log, level: logLevel, basePath: wd}

	dns := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.DBUser,
		url.QueryEscape(cfg.DBPass),
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBName,
	)

	db, err := gorm.Open(mysql.Open(dns), &gorm.Config{
		TranslateError: true,
		Logger: gormlogger.New(l, gormlogger.Config{
			SlowThreshold:             time.Second,
			Colorful:                  false,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      true,
			LogLevel:                  logLevel,
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot initialize db connection: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("extract db: %v", err)
	}
	sqlDB.SetMaxOpenConns(10)

	return &DB{gorm: db}, nil
}

type TxManager interface {
	Do(context.Context, func(context.Context) error) error
}

type dbLogger struct {
	logger.Logger
	level    gormlogger.LogLevel
	basePath string
}

func (l dbLogger) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)

	msg = strings.ReplaceAll(msg, "\n", " ")
	msg = strings.ReplaceAll(msg, l.basePath+string(filepath.Separator), "")

	switch l.level {
	case gormlogger.Error:
		l.Error(msg)
	case gormlogger.Warn:
		l.Warn(msg)
	default:
		l.Info(msg)
	}
}
