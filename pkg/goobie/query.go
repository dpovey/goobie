package goobie

import (
	"fmt"
	"github.com/dpovey/goobie/pkg/goobie/product"
	"github.com/dpovey/goobie/pkg/goobie/read"
	"github.com/dpovey/goobie/pkg/stream"
	"reflect"
)

// QueryAny is needed to work around limitations in the go type system.
type QueryAny interface {
	ReadAny() read.ReadAny
	SQL() Fragment
}

type QueryT[A any] struct {
	Read read.Read[A]
	sql  Fragment
}

func (q QueryT[A]) ReadAny() read.ReadAny {
	return read.ToReadAny(q.Read)
}

func (q QueryT[A]) SQL() Fragment {
	return q.sql
}

func QueryFrAndRead[A any](fragment Fragment, reader read.Read[A]) QueryT[A] {
	return QueryT[A]{
		Read: reader,
		sql:  fragment,
	}
}

// Query creates a query that returns values of type A using struct-based scanning.
// Use this for struct types where you want to scan multiple columns into struct fields.
// For primitive types (string, int, etc.), use Query0 instead for better performance.
//
// Example:
//   Query[Person]("SELECT id, name, age FROM person")
func Query[A any](sql string, args ...any) QueryT[A] {
	return QueryFr[A](Fr(sql, args...))
}

// Query0 creates a query that returns values of type A using primitive scanning.
// Use this for primitive types (string, int, float64, bool, etc.) or when scanning
// a single column. This avoids reflection overhead for simple types.
//
// Example:
//   Query0[string]("SELECT name FROM person")
//   Query0[int64]("SELECT COUNT(*) FROM person")
func Query0[A any](sql string, args ...any) QueryT[A] {
	return QueryFrAndRead[A](Fr(sql, args...), read.Get[A]{})
}

func Query0Fr[A any](fragment Fragment) QueryT[A] {
	return QueryFrAndRead[A](fragment, read.Get[A]{})
}

func QueryFr[A any](fragment Fragment) QueryT[A] {
	return QueryFrAndRead[A](fragment, product.NewProduct[A]())
}

// Convenience functions for common primitive types
// These provide a more explicit API for frequently used types

// QueryString is a convenience function for string queries
func QueryString(sql string, args ...any) QueryT[string] {
	return Query0[string](sql, args...)
}

// QueryInt is a convenience function for int queries  
func QueryInt(sql string, args ...any) QueryT[int] {
	return Query0[int](sql, args...)
}

// QueryInt64 is a convenience function for int64 queries
func QueryInt64(sql string, args ...any) QueryT[int64] {
	return Query0[int64](sql, args...)
}

// QueryFloat64 is a convenience function for float64 queries
func QueryFloat64(sql string, args ...any) QueryT[float64] {
	return Query0[float64](sql, args...)
}

// QueryBool is a convenience function for bool queries
func QueryBool(sql string, args ...any) QueryT[bool] {
	return Query0[bool](sql, args...)
}

func (q QueryT[A]) ToSlice() Op[[]A] {
	var a A
	resultType := reflect.TypeOf(a)
	run := func(t Runner) ([]A, error) {
		result, err := t.RunQuery(q, SLICE, resultType)
		if err != nil {
			return nil, err
		}
		value := result.(reflect.Value)
		if value.Kind() != reflect.Slice {
			return nil, fmt.Errorf("Expected slice, got %v", value.Kind())
		}
		return value.Interface().([]A), nil
	}
	return RunOp[[]A]{run}
}

func (q QueryT[A]) Unique() Op[A] {
	var a A
	run := func(t Runner) (A, error) {
		result, err := t.RunQuery(q, UNIQUE, reflect.TypeOf(a))
		if err != nil {
			return a, err
		}
		return result.(A), nil
	}
	return RunOp[A]{run}
}

func (q QueryT[A]) Option() Op[*A] {
	var a A
	run := func(t Runner) (*A, error) {
		result, err := t.RunQuery(q, OPTION, reflect.TypeOf(a))
		if err != nil {
			return nil, err
		}
		if result == nil {
			return nil, nil
		}
		value := result.(A)
		return &value, nil
	}
	return RunOp[*A]{run}
}

func (q QueryT[A]) None() Op[any] {
	run := func(t Runner) (any, error) {
		result, err := t.RunQuery(q, EMPTY, reflect.TypeOf(q.Read))
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return RunOp[any]{run}
}

func (q QueryT[A]) Stream() Op[stream.Stream[A]] {
	var a A
	run := func(t Runner) (stream.Stream[A], error) {
		result, err := t.RunQuery(q, STREAM, reflect.TypeOf(a))
		if err != nil {
			return stream.Fail[A](err), err
		}
		anyStream := result.(stream.Stream[any])
		return stream.UnsafeCast[A](anyStream), nil
	}
	return RunOp[stream.Stream[A]]{run}
}

//func (q QueryT[A]) Check() Op[bool] {
//	run := func(t Runner) (bool, error) {
//		_, err := t.RunCheck(any(q).(QueryT[any]), CHECK)
//	}
//}
