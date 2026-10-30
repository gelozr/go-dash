package db

import (
	"context"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// openLazy returns a handle that never connects, so Conn can be tested without MySQL.
func openLazy(t *testing.T) *gorm.DB {
	t.Helper()
	g, err := gorm.Open(mysql.New(mysql.Config{DSN: "u:p@tcp(127.0.0.1:1)/x", SkipInitializeWithVersion: true}),
		&gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func TestConn(t *testing.T) {
	base, tx := openLazy(t), openLazy(t)
	d := &DB{gorm: base}

	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "req")

	got := d.Conn(ctx)
	if got.Statement.ConnPool != base.ConnPool || got.Statement.Context != ctx {
		t.Error("without a transaction, Conn should return the base handle bound to ctx")
	}

	txCtx := context.WithValue(ctx, dbTxKey, tx)
	got = d.Conn(txCtx)
	if got.Statement.ConnPool != tx.ConnPool || got.Statement.Context != txCtx {
		t.Error("with a transaction in ctx, Conn should return it bound to ctx")
	}
}
