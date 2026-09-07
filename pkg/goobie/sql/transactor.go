package sql

import (
	"database/sql"

	"github.com/dpovey/goobie/pkg/goobie"
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

func (t Transactor) Query(fragment goobie.Fragment) (goobie.Rows, error) {
	fr := DatabaseSpecificTransforms(t.db, fragment)
	return t.db.Query(fr.SQL, fr.Args...)
}

func (t Transactor) RunUpdate(update goobie.UpdateT) (int64, error) {
	fragment := goobie.Fragment{SQL: update.GetSQL(), Args: update.GetArgs()}
	fr := DatabaseSpecificTransforms(t.db, fragment)
	result, err := t.db.Exec(fr.SQL, fr.Args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (t *Transactor) RunUpdateMany(update goobie.UpdateT, params [][]any) (int64, error) {
	fragment := goobie.Fragment{SQL: update.GetSQL(), Args: update.GetArgs()}
	fr := DatabaseSpecificTransforms(t.db, fragment)

	// For batch operations, we prepare the SQL without initial args
	// The args come from the params parameter
	var r int64 = 0
	stmt, err := t.db.Prepare(fr.SQL)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	for _, p := range params {
		result, err := stmt.Exec(p...)
		if err != nil {
			return r, err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return r, err
		}
		r += n
	}
	return r, nil
}
