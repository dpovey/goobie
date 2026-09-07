package goobie

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/dpovey/goobie/pkg/goobie/product"
	"github.com/dpovey/goobie/pkg/goobie/read"
	"github.com/dpovey/goobie/pkg/stream"
)

type QueryT[A any] struct {
	Read read.Read[A]
	sql  Fragment
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
//
//	Query[Person]("SELECT id, name, age FROM person")
func Query[A any](sql string, args ...any) QueryT[A] {
	return QueryFr[A](Fr(sql, args...))
}

// Query0 creates a query that returns values of type A using primitive scanning.
// Use this for primitive types (string, int, float64, bool, etc.) or when scanning
// a single column. This avoids reflection overhead for simple types.
//
// Example:
//
//	Query0[string]("SELECT name FROM person")
//	Query0[int64]("SELECT COUNT(*) FROM person")
func Query0[A any](sql string, args ...any) QueryT[A] {
	return QueryFrAndRead[A](Fr(sql, args...), read.Get[A]())
}

func Query0Fr[A any](fragment Fragment) QueryT[A] {
	return QueryFrAndRead[A](fragment, read.Get[A]())
}

func QueryFr[A any](fragment Fragment) QueryT[A] {
	return QueryFrAndRead[A](fragment, read.NewReader(func(scan read.RowScan) (A, error) {
		return product.NewProduct[A]().Read(scan)
	}))
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
	run := func(t Runner) (result []A, err error) {
		rows, err := t.Query(q.sql)
		if err != nil {
			return nil, err
		}
		defer func() {
			err = errors.Join(err, rows.Close())
		}()

		result = make([]A, 0)
		for rows.Next() {
			value, err := q.Read.Read(rows.Scan)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return result, nil
	}
	return NewOp(run)
}

func (q QueryT[A]) Unique() Op[A] {
	run := func(t Runner) (result A, err error) {
		var zero A
		rows, err := t.Query(q.sql)
		if err != nil {
			return zero, err
		}
		defer func() {
			err = errors.Join(err, rows.Close())
		}()

		if !rows.Next() {
			if err := rows.Err(); err != nil {
				return zero, err
			}
			return zero, fmt.Errorf("expected at least one result")
		}
		result, err = q.Read.Read(rows.Scan)
		if err != nil {
			return zero, err
		}
		if rows.Next() {
			return zero, fmt.Errorf("multiple rows found")
		}
		if err := rows.Err(); err != nil {
			return zero, err
		}
		return result, nil
	}
	return NewOp(run)
}

func (q QueryT[A]) Option() Op[*A] {
	run := func(t Runner) (result *A, err error) {
		rows, err := t.Query(q.sql)
		if err != nil {
			return nil, err
		}
		defer func() {
			err = errors.Join(err, rows.Close())
		}()

		if !rows.Next() {
			if err := rows.Err(); err != nil {
				return nil, err
			}
			return nil, nil
		}
		value, err := q.Read.Read(rows.Scan)
		if err != nil {
			return nil, err
		}
		if rows.Next() {
			return nil, fmt.Errorf("multiple rows found")
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return &value, nil
	}
	return NewOp(run)
}

func (q QueryT[A]) None() Op[any] {
	run := func(t Runner) (result any, err error) {
		rows, err := t.Query(q.sql)
		if err != nil {
			return nil, err
		}
		defer func() {
			err = errors.Join(err, rows.Close())
		}()

		if rows.Next() {
			return nil, fmt.Errorf("expected no results")
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	return NewOp(run)
}

func (q QueryT[A]) Stream() Op[stream.Stream[A]] {
	run := func(t Runner) (stream.Stream[A], error) {
		rows, err := t.Query(q.sql)
		if err != nil {
			return stream.Fail[A](err), err
		}
		closeRows := sync.OnceValue(rows.Close)
		next := func(context.Context) ([]A, bool, error) {
			chunk := make([]A, 0, 1024)
			for rows.Next() {
				value, err := q.Read.Read(rows.Scan)
				if err != nil {
					return nil, false, errors.Join(err, closeRows())
				}
				chunk = append(chunk, value)
				if len(chunk) == cap(chunk) {
					return chunk, true, nil
				}
			}
			if err := rows.Err(); err != nil {
				return nil, false, errors.Join(err, closeRows())
			}
			return chunk, false, closeRows()
		}
		return stream.FromNextChunkFuncWithClose(next, closeRows), nil
	}
	return NewOp(run)
}
