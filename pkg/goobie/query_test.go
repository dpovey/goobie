package goobie

import (
	"errors"
	"reflect"
	"testing"
	"github.com/dpovey/goobie/pkg/goobie/read"
	"github.com/dpovey/goobie/pkg/goobie/product"
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
	queryResults map[string]any
	queryErrors  map[string]error
}

func (m *mockRunner) RunQuery(query QueryAny, strategy ResultStrategy, resultType reflect.Type) (any, error) {
	sql := query.SQL().SQL
	if err, exists := m.queryErrors[sql]; exists {
		return nil, err
	}
	if result, exists := m.queryResults[sql]; exists {
		return result, nil
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
			
			// Verify it uses product reader (reflection-based)
			if _, ok := query.Read.(*product.Productizer[TestPerson]); !ok {
				t.Errorf("Query[T] should use Productizer for struct types")
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
			
			// Verify it uses Get reader (primitive scanning)
			if _, ok := query.Read.(read.Get[string]); !ok {
				t.Errorf("Query0[T] should use Get[T] for primitive types")
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
	
	// Verify it uses product reader
	if _, ok := query.Read.(*product.Productizer[TestPerson]); !ok {
		t.Errorf("QueryFr[T] should use Productizer for struct types")
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
	
	// Verify it uses Get reader
	if _, ok := query.Read.(read.Get[string]); !ok {
		t.Errorf("Query0Fr[T] should use Get[T] for primitive types")
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
	
	// Verify it returns an Op
	if op == nil {
		t.Error("ToSlice() should return a non-nil Op")
	}
	
	// Test with mock runner
	runner := &mockRunner{
		queryResults: map[string]any{
			"SELECT name FROM person": []string{"John", "Jane", "Bob"},
		},
	}
	
	result, err := op.Run(runner)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Check if result can be converted to []string
	resultAny := any(result)
	if names, ok := resultAny.([]string); ok {
		if len(names) != 3 {
			t.Errorf("Expected 3 names, got %d", len(names))
		}
	} else {
		t.Errorf("Expected []string, got %T", result)
	}
}

func TestQueryT_Unique(t *testing.T) {
	query := Query0[string]("SELECT name FROM person WHERE id = 1")
	op := query.Unique()
	
	// Verify it returns an Op
	if op == nil {
		t.Error("Unique() should return a non-nil Op")
	}
	
	// Test with mock runner
	runner := &mockRunner{
		queryResults: map[string]any{
			"SELECT name FROM person WHERE id = 1": "John",
		},
	}
	
	result, err := op.Run(runner)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Check if result can be converted to string
	resultAny := any(result)
	if name, ok := resultAny.(string); !ok {
		t.Errorf("Expected string, got %T", result)
	} else if name != "John" {
		t.Errorf("Expected 'John', got %q", name)
	}
}

func TestQueryT_None(t *testing.T) {
	query := Query0[string]("DELETE FROM person WHERE id = 1")
	op := query.None()
	
	// Verify it returns an Op
	if op == nil {
		t.Error("None() should return a non-nil Op")
	}
	
	// Test with mock runner
	runner := &mockRunner{
		queryResults: map[string]any{
			"DELETE FROM person WHERE id = 1": nil,
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
	
	// Verify it returns an Op
	if op == nil {
		t.Error("Stream() should return a non-nil Op")
	}
	
	// Note: Full stream testing would require more complex mocking
	// This just verifies the method exists and returns the right type
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
