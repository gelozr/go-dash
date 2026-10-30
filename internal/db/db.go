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

func Open(cfg *config.Config, log logger.Logger) (*gorm.DB, error) {
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
	// sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	// sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	return db, nil
}

type TxManager interface {
	Do(context.Context, func(context.Context) error) error
}

type GormTxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) *GormTxManager {
	return &GormTxManager{db: db}
}

func (g *GormTxManager) Do(ctx context.Context, fn func(context.Context) error) error {
	return g.db.WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			ctx = context.WithValue(ctx, dbTxKey, tx)
			return fn(ctx)
		})
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
