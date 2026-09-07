package sql

import (
	stdsql "database/sql"
	"testing"

	"github.com/dpovey/goobie/pkg/goobie"
	_ "github.com/lib/pq"
)

var (
	_ goobie.Transactor  = (*Transactor)(nil)
	_ goobie.Transaction = (*Transaction)(nil)
)

func TestDatabaseSpecificTransformsPostgresPlaceholders(t *testing.T) {
	db, err := stdsql.Open("postgres", "")
	if err != nil {
		t.Fatalf("open postgres driver: %v", err)
	}
	defer db.Close()

	got := DatabaseSpecificTransforms(db, goobie.Fr(
		"SELECT * FROM person WHERE id = ? AND name = ?",
		7,
		"alice",
	))
	if got.SQL != "SELECT * FROM person WHERE id = $1 AND name = $2" {
		t.Fatalf("transformed SQL = %q", got.SQL)
	}
}
