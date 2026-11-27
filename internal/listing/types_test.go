package listing

import (
	"math"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestPageScopeReturnsNoRowsWhenOffsetOverflows(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{
		DSN:                       "u:p@tcp(127.0.0.1:1)/x",
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}

	query := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Table("invoices").Scopes(NewPage(math.MaxInt, 100).Scope()).Find(&[]struct{ ID int }{})
	})
	if want := "SELECT * FROM `invoices` LIMIT 0"; query != want {
		t.Errorf("query = %q, want %q", query, want)
	}
}

func TestNewResultHasNext(t *testing.T) {
	for _, tc := range []struct {
		name  string
		page  Page
		total int64
		want  bool
	}{
		{"empty", NewPage(1, 10), 0, false},
		{"last page", NewPage(2, 10), 20, false},
		{"one more item", NewPage(2, 10), 21, true},
		{"overflow", NewPage(math.MaxInt, 100), 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := NewResult([]int{}, tc.page, tc.total).HasNext; got != tc.want {
				t.Errorf("HasNext = %t, want %t", got, tc.want)
			}
		})
	}
}
