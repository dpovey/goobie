package sql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/dpovey/goobie/pkg/goobie"
	"github.com/dpovey/goobie/pkg/goobie/read"
	"github.com/dpovey/goobie/pkg/stream"
	"github.com/lib/pq"
	"reflect"
)

func isPostgres(db *sql.DB) bool {
	_, ok := db.Driver().(*pq.Driver)
	return ok
}
func DatabaseSpecificTransforms(db *sql.DB, fr goobie.Fragment) goobie.Fragment {
	if isPostgres(db) {
		return fr.ToPostgres()
	}
	return fr
}

func MapResultToSlice(rows *sql.Rows, reader read.ReadAny, resultType reflect.Type) (any, error) {
	results := reflect.MakeSlice(reflect.SliceOf(resultType), 0, 0)
	for rows.Next() {
		r, err := reader.ReadAny(rows.Scan)
		if err != nil {
			return results, err
		}
		results = reflect.Append(results, reflect.ValueOf(r))
	}
	return results, rows.Err()
}

func MapResultToUnique(rows *sql.Rows, reader read.ReadAny) (any, error) {
	result, err := MapResultToOption(rows, reader)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("expected at least one result")
	}
	if rows.Next() {
		return nil, fmt.Errorf("multiple rows found")
	}
	return *result, nil
}

func MapResultToEmpty(rows *sql.Rows) error {
	if rows.Next() {
		return fmt.Errorf("expected no results")
	}
	return nil
}

func MapResultToOption(rows *sql.Rows, reader read.ReadAny) (*any, error) {
	if rows.Next() {
		row, err := reader.ReadAny(rows.Scan)
		if err != nil {
			return nil, err
		}
		return &row, nil
	} else {
		return nil, nil
	}
}

func MapResultToStream(rows *sql.Rows, reader read.ReadAny) stream.Stream[any] {
	return stream.FromNextChunkFunc[any](func(ctx context.Context) ([]any, bool, error) {
		chunk := make([]any, 0, 1024) // Pre-allocate for efficiency
		for rows.Next() {
			value, err := reader.ReadAny(rows.Scan)
			if err != nil {
				return nil, false, err
			}
			chunk = append(chunk, value)

			if len(chunk) >= 1024 {
				return chunk, true, nil
			}
		}
		if err := rows.Err(); err != nil {
			return nil, false, err
		}
		if len(chunk) > 0 {
			return chunk, true, nil
		}
		return nil, false, nil // Stream is finished
	})
}

func mapQuery(query goobie.QueryAny, strategy goobie.ResultStrategy, err error, rows *sql.Rows, resultType reflect.Type) (any, error) {
	if err != nil {
		return nil, err
	}
	switch strategy {
	case goobie.SLICE:
		slice, err := MapResultToSlice(rows, query.ReadAny(), resultType)
		if err != nil {
			return nil, err
		}
		return any(slice), nil
	case goobie.UNIQUE:
		unique, err := MapResultToUnique(rows, query.ReadAny())
		if err != nil {
			return nil, err
		}
		return any(unique), nil
	case goobie.EMPTY:
		err := MapResultToEmpty(rows)
		if err != nil {
			return nil, err
		}
		return nil, nil
	case goobie.STREAM:
		s := MapResultToStream(rows, query.ReadAny())
		return s, nil
	default:
		return nil, fmt.Errorf("unsupported result strategy")
	}
}
