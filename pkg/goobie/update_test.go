package goobie

import (
	"errors"
	"reflect"
	"testing"
)

// Mock Runner for update testing
type mockUpdateRunner struct {
	updateResults map[string]int64
	updateErrors  map[string]error
	batchResults  map[string]int64
	batchErrors   map[string]error
}

func (m *mockUpdateRunner) RunQuery(query QueryAny, strategy ResultStrategy, resultType reflect.Type) (any, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockUpdateRunner) RunUpdate(update UpdateT) (int64, error) {
	sql := update.GetSQL()
	if err, exists := m.updateErrors[sql]; exists {
		return 0, err
	}
	if result, exists := m.updateResults[sql]; exists {
		return result, nil
	}
	return 0, errors.New("no mock result configured")
}

func (m *mockUpdateRunner) RunUpdateMany(update UpdateT, params [][]any) (int64, error) {
	sql := update.GetSQL()
	if err, exists := m.batchErrors[sql]; exists {
		return 0, err
	}
	if result, exists := m.batchResults[sql]; exists {
		return result, nil
	}
	return 0, errors.New("no mock batch result configured")
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		args []any
	}{
		{"Simple insert", "INSERT INTO person (name) VALUES (?)", []any{"John"}},
		{"Update with where", "UPDATE person SET age = ? WHERE id = ?", []any{25, 1}},
		{"Delete", "DELETE FROM person WHERE id = ?", []any{1}},
		{"No parameters", "TRUNCATE TABLE person", []any{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update := Update(tt.sql, tt.args...)
			
			// Verify SQL and args are set correctly
			if update.GetSQL() != tt.sql {
				t.Errorf("Expected SQL %q, got %q", tt.sql, update.GetSQL())
			}
			
			if !reflect.DeepEqual(update.GetArgs(), tt.args) {
				t.Errorf("Expected args %v, got %v", tt.args, update.GetArgs())
			}
			
			// Verify it's an UpdateF
			if update.GetSQL() == "" {
				t.Errorf("Update() should return valid UpdateF with SQL")
			}
		})
	}
}

func TestUpdateF_Run(t *testing.T) {
	update := Update("INSERT INTO person (name) VALUES (?)", "John")
	
	// Test successful execution
	runner := &mockUpdateRunner{
		updateResults: map[string]int64{
			"INSERT INTO person (name) VALUES (?)": 1,
		},
	}
	
	result, err := update.Run(runner)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if result != 1 {
		t.Errorf("Expected 1 row affected, got %d", result)
	}
}

func TestUpdateF_RunError(t *testing.T) {
	update := Update("INSERT INTO nonexistent (name) VALUES (?)", "John")
	
	// Test error handling
	runner := &mockUpdateRunner{
		updateErrors: map[string]error{
			"INSERT INTO nonexistent (name) VALUES (?)": errors.New("table does not exist"),
		},
	}
	
	result, err := update.Run(runner)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	
	if result != 0 {
		t.Errorf("Expected 0 rows affected on error, got %d", result)
	}
	
	if err.Error() != "table does not exist" {
		t.Errorf("Expected specific error message, got %q", err.Error())
	}
}

func TestUpdateFr(t *testing.T) {
	fragment := Fr("UPDATE person SET age = ? WHERE id = ?", 30, 1)
	update := UpdateFr(fragment)
	
	// Verify fragment is used correctly
	if update.GetSQL() != fragment.SQL {
		t.Errorf("Expected SQL %q, got %q", fragment.SQL, update.GetSQL())
	}
	
	if !reflect.DeepEqual(update.GetArgs(), fragment.Args) {
		t.Errorf("Expected args %v, got %v", fragment.Args, update.GetArgs())
	}
	
	// Verify it's an UpdateF
	if update.GetSQL() == "" {
		t.Errorf("UpdateFr() should return valid UpdateF with SQL")
	}
}

func TestUpdateAll(t *testing.T) {
	updates := []UpdateF{
		Update("INSERT INTO person (name) VALUES (?)", "John"),
		Update("INSERT INTO person (name) VALUES (?)", "Jane"),
		Update("INSERT INTO person (name) VALUES (?)", "Bob"),
	}
	
	batchUpdate := UpdateAll(updates...)
	
	_, err := batchUpdate.Run(&mockUpdateRunner{})
	// We expect an error since no mock results are configured, but it should not panic
	if err == nil {
		t.Error("Expected error from empty mock runner")
	}
}

func TestUpdateOp_Run(t *testing.T) {
	updates := []UpdateF{
		Update("INSERT INTO person (name) VALUES (?)", "John"),
		Update("INSERT INTO person (name) VALUES (?)", "Jane"),
	}
	
	batchUpdate := UpdateAll(updates...)
	
	// Mock runner that expects individual update calls
	runner := &mockUpdateRunner{
		updateResults: map[string]int64{
			"INSERT INTO person (name) VALUES (?)": 1,
		},
	}
	
	result, err := batchUpdate.Run(runner)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Should return total rows affected (2 updates * 1 row each = 2)
	if result != 2 {
		t.Errorf("Expected 2 total rows affected, got %d", result)
	}
}

func TestUpdateOp_RunError(t *testing.T) {
	updates := []UpdateF{
		Update("INSERT INTO person (name) VALUES (?)", "John"),
		Update("INSERT INTO nonexistent (name) VALUES (?)", "Jane"),
	}
	
	batchUpdate := UpdateAll(updates...)
	
	// Mock runner with error on second update
	runner := &mockUpdateRunner{
		updateResults: map[string]int64{
			"INSERT INTO person (name) VALUES (?)": 1,
		},
		updateErrors: map[string]error{
			"INSERT INTO nonexistent (name) VALUES (?)": errors.New("table does not exist"),
		},
	}
	
	result, err := batchUpdate.Run(runner)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	
	// Should return partial result (1 successful update)
	if result != 1 {
		t.Errorf("Expected 1 row affected before error, got %d", result)
	}
}

// Test struct for generic update testing
type TestPersonInfo struct {
	Name string
	Age  int
}

func TestNewUpdateT(t *testing.T) {
	sql := "INSERT INTO person (name, age) VALUES (?, ?)"
	values := []TestPersonInfo{
		{Name: "John", Age: 25},
		{Name: "Jane", Age: 30},
	}
	
	update := NewUpdateT[TestPersonInfo](sql, values...)
	
	// Verify SQL is set correctly
	if update.GetSQL() != sql {
		t.Errorf("Expected SQL %q, got %q", sql, update.GetSQL())
	}
	
	// Verify it's an UpdateWithWrite
	if update.GetSQL() == "" {
		t.Errorf("NewUpdateT() should return valid UpdateWithWrite with SQL")
	}
}

func TestUpdateWithWrite_GetArgs(t *testing.T) {
	sql := "INSERT INTO person (name, age) VALUES (?, ?)"
	values := []TestPersonInfo{
		{Name: "John", Age: 25},
		{Name: "Jane", Age: 30},
	}
	
	update := NewUpdateT[TestPersonInfo](sql, values...)
	args := update.GetArgs()
	
	// Should flatten struct values into args
	// Each TestPersonInfo should contribute 2 args (name, age)
	expectedArgCount := len(values) * 2
	if len(args) != expectedArgCount {
		t.Errorf("Expected %d args, got %d", expectedArgCount, len(args))
	}
	
	// Check first struct's values
	if args[0] != "John" || args[1] != 25 {
		t.Errorf("Expected first struct args [John, 25], got [%v, %v]", args[0], args[1])
	}
	
	// Check second struct's values
	if args[2] != "Jane" || args[3] != 30 {
		t.Errorf("Expected second struct args [Jane, 30], got [%v, %v]", args[2], args[3])
	}
}

func TestUpdateWithWrite_Run(t *testing.T) {
	sql := "INSERT INTO person (name, age) VALUES (?, ?)"
	values := []TestPersonInfo{
		{Name: "John", Age: 25},
	}
	
	update := NewUpdateT[TestPersonInfo](sql, values...)
	
	// Test successful execution
	runner := &mockUpdateRunner{
		updateResults: map[string]int64{
			sql: 1,
		},
	}
	
	result, err := update.Run(runner)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if result != 1 {
		t.Errorf("Expected 1 row affected, got %d", result)
	}
}

func TestUpdateT_Interface(t *testing.T) {
	// Test that both UpdateF and UpdateWithWrite implement UpdateT interface
	var updateT UpdateT
	
	// UpdateF should implement UpdateT
	updateF := Update("INSERT INTO person (name) VALUES (?)", "John")
	updateT = updateF
	if updateT == nil {
		t.Error("UpdateF should implement UpdateT interface")
	}
	
	// UpdateWithWrite should implement UpdateT
	updateWithWrite := NewUpdateT[TestPersonInfo]("INSERT INTO person (name, age) VALUES (?, ?)", TestPersonInfo{Name: "John", Age: 25})
	updateT = updateWithWrite
	if updateT == nil {
		t.Error("UpdateWithWrite should implement UpdateT interface")
	}
}

func TestUpdateT_GetMethods(t *testing.T) {
	// Test UpdateF GetSQL/GetArgs
	updateF := Update("INSERT INTO person (name) VALUES (?)", "John")
	if updateF.GetSQL() != "INSERT INTO person (name) VALUES (?)" {
		t.Errorf("UpdateF.GetSQL() failed")
	}
	if !reflect.DeepEqual(updateF.GetArgs(), []any{"John"}) {
		t.Errorf("UpdateF.GetArgs() failed")
	}
	
	// Test UpdateWithWrite GetSQL/GetArgs
	updateWithWrite := NewUpdateT[TestPersonInfo]("INSERT INTO person (name, age) VALUES (?, ?)", TestPersonInfo{Name: "John", Age: 25})
	if updateWithWrite.GetSQL() != "INSERT INTO person (name, age) VALUES (?, ?)" {
		t.Errorf("UpdateWithWrite.GetSQL() failed")
	}
	args := updateWithWrite.GetArgs()
	if len(args) != 2 || args[0] != "John" || args[1] != 25 {
		t.Errorf("UpdateWithWrite.GetArgs() failed, got %v", args)
	}
}
