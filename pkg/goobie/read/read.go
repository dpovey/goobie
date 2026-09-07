package read

import (
	"time"
)

// Scan function that takes a slice of pointers to values to scan and
// fills them from a row of data
type RowScan = func(...any) error

// Read decodes one database row into a value of type T.
type Read[T any] struct {
	read func(RowScan) (T, error)
}

// Reader is retained as an alias for compatibility.
type Reader[T any] = Read[T]

func NewReader[T any](read func(RowScan) (T, error)) Read[T] {
	return Read[T]{read: read}
}

func (r Read[T]) Read(scan RowScan) (T, error) {
	return r.read(scan)
}

// Map applies a transformation to the value read by r.
func (r Read[T]) Map[U any](f func(T) U) Read[U] {
	return Read[U]{
		read: func(s RowScan) (U, error) {
			value, err := r.Read(s)
			if err != nil {
				var zero U
				return zero, err
			}
			return f(value), nil
		},
	}
}

// Get returns a reader for a single column.
func Get[T any]() Read[T] {
	return NewReader(func(scan RowScan) (T, error) {
		var result T
		err := scan(&result)
		return result, err
	})
}

var (
	IntRead   = Get[int]()
	Int8Read  = Get[int8]()
	Int16Read = Get[int16]()
	Int32Read = Get[int32]()
	Int64Read = Get[int64]()

	UintRead = Get[uint]()

	Uint8Read = Get[uint8]()

	Uint16Read = Get[uint16]()

	Uint32Read = Get[uint32]()

	Uint64Read = Get[uint64]()

	Float32Read = Get[float32]()

	Float64Read = Get[float64]()

	BoolRead = Get[bool]()

	StringRead = Get[string]()

	TimeRead = Get[time.Time]()

	// TODO others
)
