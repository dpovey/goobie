package goobie

// Rows is the minimal row cursor required by typed query decoders.
type Rows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close() error
}

// Runner the base interface for Transactor and Transaction.
type Runner interface {
	Query(Fragment) (Rows, error)
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
