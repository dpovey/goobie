package goobie

import (
	"reflect"
	"testing"
)

func TestCountPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected int
	}{
		{"Empty string", "", 0},
		{"No placeholders", "SELECT * FROM users", 0},
		{"Single placeholder", "SELECT * FROM users WHERE id = ?", 1},
		{"Multiple placeholders", "INSERT INTO users (name, age) VALUES (?, ?)", 2},
		{"Placeholder in string", "SELECT * FROM users WHERE name = '?'", 0},
		{"Escaped quote", "SELECT * FROM users WHERE name = '\\''", 0},
		{"Complex query", "SELECT * FROM users WHERE name = ? AND age > ? AND city IN (?, ?, ?)", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countPlaceholders(tt.sql); got != tt.expected {
				t.Errorf("countPlaceholders() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSplitOnPlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		sql      string
		expected []string
	}{
		{"Empty string", "", []string{""}},
		{"No placeholders", "SELECT * FROM users", []string{"SELECT * FROM users"}},
		{"Single placeholder", "SELECT * FROM users WHERE id = ?", []string{"SELECT * FROM users WHERE id = ", ""}},
		{"Multiple placeholders", "INSERT INTO users (name, age) VALUES (?, ?)", []string{"INSERT INTO users (name, age) VALUES (", ", ", ")"}},
		{"Placeholder in string", "SELECT * FROM users WHERE name = '?'", []string{"SELECT * FROM users WHERE name = '?'"}},
		{"Escaped quote", "SELECT * FROM users WHERE name = '\\''", []string{"SELECT * FROM users WHERE name = '\\''"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitOnPlaceholders(tt.sql); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("splitOnPlaceholders() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFragmentToPostgres(t *testing.T) {
	tests := []struct {
		name     string
		fragment Fragment
		expected Fragment
	}{
		{
			"Empty fragment",
			Fragment{},
			Fragment{},
		},
		{
			"No placeholders",
			Fragment{SQL: "SELECT * FROM users"},
			Fragment{SQL: "SELECT * FROM users"},
		},
		{
			"Single placeholder",
			Fragment{SQL: "SELECT * FROM users WHERE id = ?", Args: []any{1}},
			Fragment{SQL: "SELECT * FROM users WHERE id = $1", Args: []any{1}},
		},
		{
			"Multiple placeholders",
			Fragment{SQL: "INSERT INTO users (name, age) VALUES (?, ?)", Args: []any{"John", 30}},
			Fragment{SQL: "INSERT INTO users (name, age) VALUES ($1, $2)", Args: []any{"John", 30}},
		},
		{
			"Repeated arguments",
			Fragment{SQL: "SELECT * FROM users WHERE name = ? OR name = ?", Args: []any{"John", "John"}},
			Fragment{SQL: "SELECT * FROM users WHERE name = $1 OR name = $1", Args: []any{"John"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fragment.ToPostgres(); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Fragment.ToPostgres() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestFrConst(t *testing.T) {
	tests := []struct {
		name        string
		sql         string
		shouldPanic bool
	}{
		{"Valid constant", "SELECT * FROM users", false},
		{"With placeholder", "SELECT * FROM users WHERE id = ?", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("FrConst() did not panic")
					}
				}()
			}
			got := FrConst(tt.sql)
			if !tt.shouldPanic && got.SQL != tt.sql {
				t.Errorf("FrConst() = %v, want %v", got.SQL, tt.sql)
			}
		})
	}
}

func TestFr0Unchecked(t *testing.T) {
	sql := "SELECT * FROM users"
	got := Fr0Unchecked(sql)
	if got.SQL != sql || got.Args != nil {
		t.Errorf("Fr0Unchecked() = %v, want {SQL: %v, Args: nil}", got, sql)
	}
}

func TestSQL(t *testing.T) {
	sql := "SELECT * FROM users WHERE id = ?"
	arg := 1
	got := SQL(sql, arg)
	if got.SQL != sql || !reflect.DeepEqual(got.Args, []any{arg}) {
		t.Errorf("SQL() = %v, want {SQL: %v, Args: %v}", got, sql, []any{arg})
	}
}

func TestFr(t *testing.T) {
	sql := "SELECT * FROM users WHERE id = ? AND name = ?"
	args := []any{1, "John"}
	got := Fr(sql, args...)
	if got.SQL != sql || !reflect.DeepEqual(got.Args, []any{1, "John"}) {
		t.Errorf("Fr() = %v, want {SQL: %v, Args: %v}", got, sql, []any{1, "John"})
	}
}

func TestFragmentConcat(t *testing.T) {
	f1 := Fragment{SQL: "SELECT * FROM users", Args: []any{1}}
	f2 := Fragment{SQL: "WHERE id = ?", Args: []any{2}}
	expected := Fragment{SQL: "SELECT * FROM users\nWHERE id = ?", Args: []any{1, 2}}
	got := f1.Concat(f2)
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Fragment.Concat() = %v, want %v", got, expected)
	}
}

func TestCombine(t *testing.T) {
	tests := []struct {
		name      string
		fragments []Fragment
		expected  Fragment
	}{
		{
			"Empty slice",
			[]Fragment{},
			Fragment{},
		},
		{
			"Single fragment",
			[]Fragment{{SQL: "SELECT * FROM users", Args: []any{1}}},
			Fragment{SQL: "SELECT * FROM users", Args: []any{1}},
		},
		{
			"Multiple fragments",
			[]Fragment{
				{SQL: "SELECT * FROM users", Args: []any{1}},
				{SQL: "WHERE id = ?", Args: []any{2}},
				{SQL: "AND name = ?", Args: []any{"John"}},
			},
			Fragment{SQL: "SELECT * FROM users\nWHERE id = ?\nAND name = ?", Args: []any{1, 2, "John"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Combine(tt.fragments); !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Combine() = %v, want %v", got, tt.expected)
			}
		})
	}
}
