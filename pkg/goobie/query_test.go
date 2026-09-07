package goobie

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/dpovey/goobie/pkg/goobie/read"
	"github.com/dpovey/goobie/pkg/stream"
)

// Mock RowScan function for testing
func mockRowScan(values ...any) func(...any) error {
	index := 0
	return func(dest ...any) error {
		if index >= len(values) {
			return errors.New("no more rows")
		}

		remainingValues := values[index:]
		if len(dest) > len(remainingValues) {
			return errors.New("not enough values to scan")
		}

		for i, v := range dest {
			reflect.ValueOf(v).Elem().Set(reflect.ValueOf(remainingValues[i]))
		}

		index += len(dest)
		return nil
	}
}

// Mock Runner for testing
type mockRunner struct {
	queryResults    map[string][][]any
	queryErrors     map[string]error
	queryScanErrors map[string]error
	queryRowErrors  map[string]error
	lastRows        *mockRows
}

type mockRows struct {
	rows    [][]any
	index   int
	closed  bool
	err     error
	scanErr error
}

func (r *mockRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *mockRows) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	if r.index == 0 || r.index > len(r.rows) {
		return errors.New("Scan called without current row")
	}
	row := r.rows[r.index-1]
	if len(dest) != len(row) {
		return errors.New("destination count does not match row")
	}
	for i := range dest {
		reflect.ValueOf(dest[i]).Elem().Set(reflect.ValueOf(row[i]))
	}
	return nil
}

func (r *mockRows) Err() error {
	return r.err
}

func (r *mockRows) Close() error {
	r.closed = true
	return nil
}

func (m *mockRunner) Query(fragment Fragment) (Rows, error) {
	sql := fragment.SQL
	if err, exists := m.queryErrors[sql]; exists {
		return nil, err
	}
	if result, exists := m.queryResults[sql]; exists {
		m.lastRows = &mockRows{
			rows:    result,
			err:     m.queryRowErrors[sql],
			scanErr: m.queryScanErrors[sql],
		}
		return m.lastRows, nil
	}
	return nil, errors.New("no mock result configured")
}

func (m *mockRunner) RunUpdate(update UpdateT) (int64, error) {
	return 0, errors.New("not implemented in mock")
}

func (m *mockRunner) RunUpdateMany(update UpdateT, params [][]any) (int64, error) {
	return 0, errors.New("not implemented in mock")
}

// Test struct for query tests
type TestPerson struct {
	ID   int64
	Name string
	Age  int
}

func TestQuery(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		args []any
	}{
		{"Simple select", "SELECT id, name, age FROM person", []any{}},
		{"With parameters", "SELECT id, name FROM person WHERE age > ?", []any{18}},
		{"Multiple parameters", "SELECT * FROM person WHERE name = ? AND age = ?", []any{"John", 25}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := Query[TestPerson](tt.sql, tt.args...)

			// Verify SQL and args are set correctly
			if query.SQL().SQL != tt.sql {
				t.Errorf("Expected SQL %q, got %q", tt.sql, query.SQL().SQL)
			}

			if !reflect.DeepEqual(query.SQL().Args, tt.args) {
				t.Errorf("Expected args %v, got %v", tt.args, query.SQL().Args)
			}

		})
	}
}

func TestQuery0(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		args []any
	}{
		{"Simple count", "SELECT COUNT(*) FROM person", []any{}},
		{"With parameter", "SELECT name FROM person WHERE id = ?", []any{1}},
		{"Multiple parameters", "SELECT age FROM person WHERE name = ? AND city = ?", []any{"John", "NYC"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := Query0[string](tt.sql, tt.args...)

			// Verify SQL and args are set correctly
			if query.SQL().SQL != tt.sql {
				t.Errorf("Expected SQL %q, got %q", tt.sql, query.SQL().SQL)
			}

			if !reflect.DeepEqual(query.SQL().Args, tt.args) {
				t.Errorf("Expected args %v, got %v", tt.args, query.SQL().Args)
			}

		})
	}
}

func TestQueryFr(t *testing.T) {
	fragment := Fr("SELECT * FROM person WHERE id = ?", 123)
	query := QueryFr[TestPerson](fragment)

	// Verify fragment is used correctly
	if query.SQL().SQL != fragment.SQL {
		t.Errorf("Expected SQL %q, got %q", fragment.SQL, query.SQL().SQL)
	}

	if !reflect.DeepEqual(query.SQL().Args, fragment.Args) {
		t.Errorf("Expected args %v, got %v", fragment.Args, query.SQL().Args)
	}

}

func TestQuery0Fr(t *testing.T) {
	fragment := Fr("SELECT name FROM person WHERE id = ?", 123)
	query := Query0Fr[string](fragment)

	// Verify fragment is used correctly
	if query.SQL().SQL != fragment.SQL {
		t.Errorf("Expected SQL %q, got %q", fragment.SQL, query.SQL().SQL)
	}

	if !reflect.DeepEqual(query.SQL().Args, fragment.Args) {
		t.Errorf("Expected args %v, got %v", fragment.Args, query.SQL().Args)
	}

}

// Test convenience functions
func TestQueryConvenienceFunctions(t *testing.T) {
	sql := "SELECT name FROM person"
	args := []any{123}

	tests := []struct {
		name     string
		querySql string
		expected string
	}{
		{"QueryString", QueryString(sql, args...).SQL().SQL, sql},
		{"QueryInt", QueryInt(sql, args...).SQL().SQL, sql},
		{"QueryInt64", QueryInt64(sql, args...).SQL().SQL, sql},
		{"QueryFloat64", QueryFloat64(sql, args...).SQL().SQL, sql},
		{"QueryBool", QueryBool(sql, args...).SQL().SQL, sql},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.querySql != tt.expected {
				t.Errorf("Expected SQL %q, got %q", tt.expected, tt.querySql)
			}
		})
	}
}

func TestQueryT_ToSlice(t *testing.T) {
	query := Query0[string]("SELECT name FROM person")
	op := query.ToSlice()
	var _ Op[[]string] = op

	// Test with mock runner
	runner := &mockRunner{
		queryResults: map[string][][]any{
			"SELECT name FROM person": {{"John"}, {"Jane"}, {"Bob"}},
		},
	}

	result, err := op.Run(runner)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, []string{"John", "Jane", "Bob"}) {
		t.Errorf("Expected names, got %v", result)
	}
}

func TestQueryT_ToSliceWithTypedProduct(t *testing.T) {
	type person struct {
		id   int
		name string
	}
	reader := (read.Row2[int, string]{}).Map(func(id int, name string) person {
		return person{id: id, name: name}
	})
	query := QueryFrAndRead(Fr("SELECT id, name FROM person"), reader)
	runner := &mockRunner{
		queryResults: map[string][][]any{
			"SELECT id, name FROM person": {{1, "John"}, {2, "Jane"}},
		},
	}

	got, err := query.ToSlice().Run(runner)
	if err != nil {
		t.Fatalf("ToSlice().Run() error = %v", err)
	}
	want := []person{{id: 1, name: "John"}, {id: 2, name: "Jane"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ToSlice().Run() = %v, want %v", got, want)
	}
}

func TestQueryT_Unique(t *testing.T) {
	query := Query0[string]("SELECT name FROM person WHERE id = 1")
	op := query.Unique()
	var _ Op[string] = op

	// Test with mock runner
	runner := &mockRunner{
		queryResults: map[string][][]any{
			"SELECT name FROM person WHERE id = 1": {{"John"}},
		},
	}

	result, err := op.Run(runner)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != "John" {
		t.Errorf("Expected 'John', got %q", result)
	}
}

func TestQueryT_None(t *testing.T) {
	query := Query0[string]("DELETE FROM person WHERE id = 1")
	op := query.None()
	var _ Op[any] = op

	// Test with mock runner
	runner := &mockRunner{
		queryResults: map[string][][]any{
			"DELETE FROM person WHERE id = 1": {},
		},
	}

	result, err := op.Run(runner)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result, got %v", result)
	}
}

func TestQueryT_Stream(t *testing.T) {
	query := Query0[string]("SELECT name FROM person")
	op := query.Stream()
	var _ Op[stream.Stream[string]] = op

	runner := &mockRunner{
		queryResults: map[string][][]any{
			"SELECT name FROM person": {{"John"}, {"Jane"}},
		},
	}
	result, err := op.Run(runner)
	if err != nil {
		t.Fatalf("Stream().Run() error = %v", err)
	}
	names, err := result.Collect(context.Background())
	if err != nil {
		t.Fatalf("Stream().Collect() error = %v", err)
	}
	if !reflect.DeepEqual(names, []string{"John", "Jane"}) {
		t.Fatalf("Stream().Collect() = %v", names)
	}
	if !runner.lastRows.closed {
		t.Fatal("Stream().Collect() did not close rows")
	}
}

func TestQueryT_StreamTakeClosesRows(t *testing.T) {
	rows := make([][]any, 1025)
	for i := range rows {
		rows[i] = []any{i + 1}
	}
	runner := &mockRunner{
		queryResults: map[string][][]any{
			"SELECT id FROM person": rows,
		},
	}
	result, err := Query0[int]("SELECT id FROM person").Stream().Run(runner)
	if err != nil {
		t.Fatalf("Stream().Run() error = %v", err)
	}
	values, err := result.Take(1).Collect(context.Background())
	if err != nil {
		t.Fatalf("Take().Collect() error = %v", err)
	}
	if !reflect.DeepEqual(values, []int{1}) {
		t.Fatalf("Take().Collect() = %v, want [1]", values)
	}
	if !runner.lastRows.closed {
		t.Fatal("Take().Collect() did not close rows")
	}
}

func TestQueryT_Option(t *testing.T) {
	const sql = "SELECT name FROM person WHERE id = 1"
	t.Run("present", func(t *testing.T) {
		runner := &mockRunner{queryResults: map[string][][]any{sql: {{"John"}}}}
		result, err := Query0[string](sql).Option().Run(runner)
		if err != nil {
			t.Fatalf("Option().Run() error = %v", err)
		}
		if result == nil || *result != "John" {
			t.Fatalf("Option().Run() = %v, want John", result)
		}
		if !runner.lastRows.closed {
			t.Fatal("Option().Run() did not close rows")
		}
	})
	t.Run("empty", func(t *testing.T) {
		runner := &mockRunner{queryResults: map[string][][]any{sql: {}}}
		result, err := Query0[string](sql).Option().Run(runner)
		if err != nil {
			t.Fatalf("Option().Run() error = %v", err)
		}
		if result != nil {
			t.Fatalf("Option().Run() = %v, want nil", result)
		}
	})
	t.Run("multiple", func(t *testing.T) {
		runner := &mockRunner{queryResults: map[string][][]any{sql: {{"John"}, {"Jane"}}}}
		if _, err := Query0[string](sql).Option().Run(runner); err == nil {
			t.Fatal("Option().Run() error = nil")
		}
	})
}

func TestQueryT_UniqueRejectsWrongCardinality(t *testing.T) {
	const sql = "SELECT name FROM person"
	for _, tc := range []struct {
		name string
		rows [][]any
	}{
		{name: "empty", rows: nil},
		{name: "multiple", rows: [][]any{{"John"}, {"Jane"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := &mockRunner{queryResults: map[string][][]any{sql: tc.rows}}
			if _, err := Query0[string](sql).Unique().Run(runner); err == nil {
				t.Fatal("Unique().Run() error = nil")
			}
		})
	}
}

func TestQueryT_PropagatesScanAndRowsErrors(t *testing.T) {
	const sql = "SELECT name FROM person"
	scanErr := errors.New("scan failed")
	runner := &mockRunner{
		queryResults:    map[string][][]any{sql: {{"John"}}},
		queryScanErrors: map[string]error{sql: scanErr},
	}
	if _, err := Query0[string](sql).ToSlice().Run(runner); !errors.Is(err, scanErr) {
		t.Fatalf("ToSlice().Run() error = %v, want %v", err, scanErr)
	}

	rowsErr := errors.New("rows failed")
	runner = &mockRunner{
		queryResults:   map[string][][]any{sql: {}},
		queryRowErrors: map[string]error{sql: rowsErr},
	}
	if _, err := Query0[string](sql).ToSlice().Run(runner); !errors.Is(err, rowsErr) {
		t.Fatalf("ToSlice().Run() error = %v, want %v", err, rowsErr)
	}
}

func TestQueryT_NoneRejectsRows(t *testing.T) {
	const sql = "DELETE FROM person RETURNING name"
	runner := &mockRunner{
		queryResults: map[string][][]any{sql: {{"John"}}},
	}
	if _, err := Query0[string](sql).None().Run(runner); err == nil {
		t.Fatal("None().Run() error = nil")
	}
}

func TestQueryT_StreamScanErrorClosesRows(t *testing.T) {
	const sql = "SELECT name FROM person"
	scanErr := errors.New("scan failed")
	runner := &mockRunner{
		queryResults:    map[string][][]any{sql: {{"John"}}},
		queryScanErrors: map[string]error{sql: scanErr},
	}
	result, err := Query0[string](sql).Stream().Run(runner)
	if err != nil {
		t.Fatalf("Stream().Run() error = %v", err)
	}
	if _, err := result.Collect(context.Background()); !errors.Is(err, scanErr) {
		t.Fatalf("Stream().Collect() error = %v, want %v", err, scanErr)
	}
	if !runner.lastRows.closed {
		t.Fatal("Stream scan error did not close rows")
	}
}

func TestQueryT_ErrorHandling(t *testing.T) {
	query := Query0[string]("SELECT name FROM nonexistent_table")
	op := query.ToSlice()

	// Test with mock runner that returns error
	runner := &mockRunner{
		queryErrors: map[string]error{
			"SELECT name FROM nonexistent_table": errors.New("table does not exist"),
		},
	}

	result, err := op.Run(runner)
	if err == nil {
		t.Error("Expected error, got nil")
	}

	if result != nil {
		t.Errorf("Expected nil result on error, got %v", result)
	}

	if err.Error() != "table does not exist" {
		t.Errorf("Expected specific error message, got %q", err.Error())
	}
}
