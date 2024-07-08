package sql

import (
	"database/sql"
	"github.com/dpovey/goobie/pkg/goobie"
	"reflect"
)

type Transaction struct {
	db *sql.DB
	tx *sql.Tx
}

func NewSqlTransaction(db *sql.DB, tx *sql.Tx) *Transaction {
	return &Transaction{db, tx}
}

func (t *Transaction) RunQuery(query goobie.QueryAny, strategy goobie.ResultStrategy, resultType reflect.Type) (any, error) {
	fr := DatabaseSpecificTransforms(t.db, query.SQL())
	rows, err := t.tx.Query(fr.SQL, fr.Args...)
	return mapQuery(query, strategy, err, rows, resultType)
}

func (t *Transaction) RunUpdate(update goobie.UpdateT) (int64, error) {
	fr := DatabaseSpecificTransforms(t.db, update.SQL)
	result, err := t.tx.Exec(fr.SQL, fr.Args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (t *Transaction) Commit() error {
	return t.tx.Commit()
}

func (t *Transaction) Rollback() error {
	return t.tx.Rollback()
}
