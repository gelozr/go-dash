package db

import (
	"context"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/gelozr/go-dash/internal/config"
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

func TestMySQLDSNPreservesPassword(t *testing.T) {
	cfg := &config.Config{
		DBUser: "user",
		DBPass: "a b+c@d%e:/?",
		DBHost: "localhost",
		DBPort: 3306,
		DBName: "go_dash",
	}

	parsed, err := mysqldriver.ParseDSN(mysqlDSN(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Passwd != cfg.DBPass {
		t.Error("parsed password does not match the configured password")
	}
}

func TestMySQLDSNPreservesDatabaseName(t *testing.T) {
	cfg := &config.Config{
		DBUser: "user",
		DBHost: "localhost",
		DBPort: 3306,
		DBName: "go?dash%25",
	}

	parsed, err := mysqldriver.ParseDSN(mysqlDSN(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.DBName != cfg.DBName {
		t.Errorf("parsed database name = %q, want %q", parsed.DBName, cfg.DBName)
	}
}

func TestMySQLDSNIPv6Host(t *testing.T) {
	for _, host := range []string{"::1", "[::1]"} {
		t.Run(host, func(t *testing.T) {
			cfg := &config.Config{DBHost: host, DBPort: 3307}
			parsed, err := mysqldriver.ParseDSN(mysqlDSN(cfg))
			if err != nil {
				t.Fatal(err)
			}
			if want := "[::1]:3307"; parsed.Addr != want {
				t.Errorf("parsed address = %q, want %q", parsed.Addr, want)
			}
		})
	}
}
