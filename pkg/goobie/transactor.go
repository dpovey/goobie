package goobie

import "reflect"

// Runner the base interface for Transactor and Transaction.
type Runner interface {
	RunQuery(query QueryAny, strategy ResultStrategy, resultType reflect.Type) (any, error)
	RunUpdate(update UpdateT) (int64, error)
	RunUpdateMany(update UpdateT, params [][]any) (int64, error)
}

// Transactor interface
type Transactor interface {
	Runner
	Transact(func(Transaction) error) error
	Begin() (Transaction, error)
}

type Transaction interface {
	Runner
	Commit() error
	Rollback() error
}

type ResultStrategy int

const (
	EMPTY ResultStrategy = iota
	UNIQUE
	OPTION
	SLICE
	STREAM
)
