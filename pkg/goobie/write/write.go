package write

import (
	"github.com/dpovey/goobie/pkg/goobie/product"
	"reflect"
)

type Write[T any] interface {
	Write(T) ([]any, error)
}

type Put[T any] struct {
	value T
}

func (p Put[T]) Write(value T) ([]any, error) {
	return []any{value}, nil
}

type WriterF[T any] func(T) ([]any, error)

func Writer[T any]() WriterF[T] {
	var zero T
	var writer WriterF[T]
	switch reflect.TypeOf(zero).Kind() {
	case reflect.Struct:
		writer = func(t T) ([]any, error) {
			return product.NewProductFrom(&t).Values(), nil
		}
	default:
		writer = func(t T) ([]any, error) {
			return Put[T]{t}.Write(t)
		}
	}
	return writer
}
