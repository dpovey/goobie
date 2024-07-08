package sql

import (
	"context"
	"database/sql"
	"github.com/dpovey/goobie/pkg/stream"
)

// SQLRowPull creates a NextChunk from SQL rows
func SQLRowPull(rows *sql.Rows, chunkSize int) stream.NextChunk[*sql.Rows] {
	// Will just fetch one row at a time and rely on the underlying sql.Rows cursor
	// to handle batch fetching
	return stream.NewPull(func(ctx context.Context) ([]*sql.Rows, bool, error) {
		if !rows.Next() {
			return nil, false, rows.Err()
		}

		return []*sql.Rows{rows}, true, nil
	})
}

// NewSQLRowStream creates a Stream from SQL rows
func NewSQLRowStream(rows *sql.Rows, chunkSize int) stream.Stream[*sql.Rows] {
	return stream.New(SQLRowPull(rows, chunkSize))
}
