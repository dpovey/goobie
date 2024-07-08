package sql

import (
	"database/sql"
	"github.com/dpovey/goobie/pkg/goobie"
	"reflect"
)

type Transactor struct {
	db *sql.DB
}

func NewSqlTransactor(db *sql.DB) *Transactor {
	return &Transactor{db: db}
}

func (t Transactor) Transact(fn func(transaction goobie.Transaction) error) error {
	tx, err := t.Begin()
	if err != nil {
		return err
	}

	// Defer a function to handle rollback
	defer func() {
		if p := recover(); p != nil {
			// A panic occurred, attempt to roll back
			_ = tx.Rollback()
			panic(p) // Re-throw panic after Rollback
		} else if err != nil {
			// If error is non-nil, rollback
			_ = tx.Rollback()
		} else {
			// Attempt to commit the transaction
			err = tx.Commit()
		}
	}()

	// Run the provided function
	err = fn(tx)

	// Return the error (if any)
	return err
}

func (t Transactor) Begin() (goobie.Transaction, error) {
	tx, err := t.db.Begin()
	if err != nil {
		return nil, err
	}
	return NewSqlTransaction(t.db, tx), nil
}

func (t Transactor) RunQuery(query goobie.QueryAny, strategy goobie.ResultStrategy, resultType reflect.Type) (any, error) {
	fr := DatabaseSpecificTransforms(t.db, query.SQL())
	rows, err := t.db.Query(fr.SQL, fr.Args...)
	return mapQuery(query, strategy, err, rows, resultType)
}
func (t Transactor) RunUpdate(update goobie.UpdateT) (int64, error) {
	fr := DatabaseSpecificTransforms(t.db, update.SQL)
	result, err := t.db.Exec(fr.SQL, fr.Args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
