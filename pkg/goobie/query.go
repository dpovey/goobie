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

func Query[A any](sql string, args ...any) QueryT[A] {
	return QueryFr[A](Fr(sql, args...))
}

// Query0 is a convenience function for queries that return a single value.
func Query0[A any](sql string, args ...any) QueryT[A] {
	return QueryFrAndRead[A](Fr(sql, args...), read.Get[A]{})
}

func Query0Fr[A any](fragment Fragment) QueryT[A] {
	return QueryFrAndRead[A](fragment, read.Get[A]{})
}

func QueryFr[A any](fragment Fragment) QueryT[A] {
	var a A
	// Check if A is a primitive type or a struct
	switch any(a).(type) {
	case interface{}:
		return QueryFrStruct[A](fragment)
	default:
		return Query0Fr[A](fragment)
	}
}

func QueryStruct[T interface{}](sql string, params ...any) QueryT[T] {
	return QueryFrStruct[T](Fr(sql, params...))
}
func QueryFrStruct[T interface{}](fragment Fragment) QueryT[T] {
	return QueryFrAndRead[T](fragment, product.NewProduct[T]())
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
	return Op[[]A]{run}
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
	return Op[A]{run}
}

func (q QueryT[A]) Option() Op[*A] {
	var a *A
	run := func(t Runner) (*A, error) {
		result, err := t.RunQuery(q, OPTION, reflect.TypeOf(a))
		if err != nil {
			return nil, err
		}
		return result.(*A), nil
	}
	return Op[*A]{run}
}

var a any

func (q QueryT[A]) None() Op[any] {
	run := func(t Runner) (any, error) {
		result, err := t.RunQuery(q, EMPTY, reflect.TypeOf(a))
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	return Op[any]{run}
}

func (q QueryT[A]) Stream() Op[stream.Stream[A]] {
	var a A
	var resultType = reflect.TypeOf(a)
	run := func(t Runner) (stream.Stream[A], error) {
		result, err := t.RunQuery(q, STREAM, resultType)
		if err != nil {
			// Shouldn't happen, but in case
			return stream.Fail[A](err), err
		}
		return stream.UnsafeCast[A](result.(stream.Stream[any])), nil
	}
	return Op[stream.Stream[A]]{run}
}

//func (q QueryT[A]) Check() Op[bool] {
//	run := func(t Runner) (bool, error) {
//		_, err := t.RunCheck(any(q).(QueryT[any]), CHECK)
//	}
//}
