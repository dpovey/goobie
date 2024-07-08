package read

import (
	"time"
)

// Scan function that takes a slice of pointers to values to scan and
// fills them from a row of data
type RowScan = func(...any) error

// Read is an interface for reading and parsing data of type T into a slice of
// pointers representing the values. For example given the following struct;
//
//	type Person struct {
//		Name string
//		Age  int
//	}
//
// An implementation of Read would look like:
//
//	func (p *Person) Values(scan RowScan) (*Person, error) {
//		err := scan(&p.Name, &p.Age)
//		if err != nil {
//			return nil, err
//		}
//		if p.Age < 0 {
//			return nil, fmt.Errorf("Age must be positive")
//		}
//		return p
//	}
type Read[T any] interface {
	Read(s RowScan) (T, error)
}

// ReadAny is needed to work around Go's type limitations when you need to use say a slice
// of Readers (c.f. Combine).  Get instances and Reader instances will automatically implement
// this. If you use a custom Read instance you will need to implement this yourself.
type ReadAny interface {
	ReadAny(s RowScan) (any, error)
}

func ToReadAny[T any](reader Read[T]) ReadAny {
	if r, ok := reader.(ReadAny); ok {
		return r
	}
	return NewReader(reader.Read)
}

// Map applies a transformation function to the output of a Read[A],
// returning a new Read[B] while preserving the arity.
func Map[A any, B any](reader Read[A], f func(A) B) Read[B] {
	return Reader[B]{
		read: func(s RowScan) (B, error) {
			a, err := reader.Read(s)
			if err != nil {
				var zero B
				return zero, err
			}
			return f(a), nil
		},
	}
}

// Reader is a helper type that implements Read
type Reader[T any] struct {
	read func(s RowScan) (T, error)
}

func NewReader[T any](read func(s RowScan) (T, error)) Reader[T] {
	return Reader[T]{read: read}
}

func (r Reader[T]) Read(s RowScan) (T, error) {
	return r.read(s)
}

func (r Reader[T]) ReadAny(s RowScan) (any, error) {
	result, err := r.Read(s)
	return any(result), err
}

func Combine(readers ...ReadAny) Read[[]any] {
	return Reader[[]any]{
		read: func(s RowScan) ([]any, error) {
			result := make([]any, len(readers))
			var err error

			for i, reader := range readers {
				result[i], err = reader.ReadAny(s)
				if err != nil {
					return nil, err
				}
			}

			return result, nil
		},
	}
}

// Get provides a common implementation for reading a single item. You can use by
type Get[T any] struct {
}

func (Get[T]) Read(s RowScan) (T, error) {
	var result T
	err := s(&result)
	return result, err
}

func (g Get[T]) ReadAny(s RowScan) (any, error) {
	return g.Read(s)
}

var (
	IntRead   = Get[int]{}
	Int8Read  = Get[int8]{}
	Int16Read = Get[int16]{}
	Int32Read = Get[int32]{}
	Int64Read = Get[int64]{}

	UintRead = Get[uint]{}

	Uint8Read = Get[uint8]{}

	Uint16Read = Get[uint16]{}

	Uint32Read = Get[uint32]{}

	Uint64Read = Get[uint64]{}

	Float32Read = Get[float32]{}

	Float64Read = Get[float64]{}

	BoolRead = Get[bool]{}

	StringRead = Get[string]{}

	TimeRead = Get[time.Time]{}

	// TODO others
)
