package sql

import (
	"database/sql"

	"github.com/dpovey/goobie/pkg/goobie"
	"github.com/lib/pq"
)

func isPostgres(db *sql.DB) bool {
	_, ok := db.Driver().(*pq.Driver)
	return ok
}
func DatabaseSpecificTransforms(db *sql.DB, fr goobie.Fragment) goobie.Fragment {
	if isPostgres(db) {
		return fr.ToPostgres()
	}
	return fr
}
