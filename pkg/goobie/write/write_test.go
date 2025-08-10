package write

import (
	"reflect"
	"testing"
)

// Test struct for write testing
type TestPerson struct {
	Name string
	Age  int
	City string
}

func TestPut_Write(t *testing.T) {
	person := TestPerson{
		Name: "John",
		Age:  25,
		City: "NYC",
	}
	
	put := Put[TestPerson]{}
	values, err := put.Write(person)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Put[T].Write returns []any{value}, so should be single element
	expected := []any{person}
	if !reflect.DeepEqual(values, expected) {
		t.Errorf("Expected %v, got %v", expected, values)
	}
}

func TestPut_WritePointer(t *testing.T) {
	person := &TestPerson{
		Name: "Jane",
		Age:  30,
		City: "LA",
	}
	
	put := Put[*TestPerson]{}
	values, err := put.Write(person)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Should handle pointer to struct
	expected := []any{person}
	if !reflect.DeepEqual(values, expected) {
		t.Errorf("Expected %v, got %v", expected, values)
	}
}

func TestPut_WriteNilPointer(t *testing.T) {
	var person *TestPerson = nil
	
	put := Put[*TestPerson]{}
	values, err := put.Write(person)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Should handle nil pointer gracefully
	if values == nil {
		t.Error("Expected non-nil slice for nil pointer")
	}
	
	expected := []any{person}
	if !reflect.DeepEqual(values, expected) {
		t.Errorf("Expected %v, got %v", expected, values)
	}
}

func TestPut_WritePrimitive(t *testing.T) {
	// Test with primitive types
	putInt := Put[int]{}
	values, err := putInt.Write(42)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	expected := []any{42}
	if !reflect.DeepEqual(values, expected) {
		t.Errorf("Expected %v, got %v", expected, values)
	}
	
	putString := Put[string]{}
	values, err = putString.Write("hello")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	expected = []any{"hello"}
	if !reflect.DeepEqual(values, expected) {
		t.Errorf("Expected %v, got %v", expected, values)
	}
}

func TestPut_WriteEmptyStruct(t *testing.T) {
	type EmptyStruct struct{}
	
	empty := EmptyStruct{}
	put := Put[EmptyStruct]{}
	values, err := put.Write(empty)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Put returns the struct itself as single element
	expected := []any{empty}
	if !reflect.DeepEqual(values, expected) {
		t.Errorf("Expected %v, got %v", expected, values)
	}
}

func TestPut_WriteNestedStruct(t *testing.T) {
	type Address struct {
		Street string
		City   string
	}
	
	type PersonWithAddress struct {
		Name    string
		Age     int
		Address Address
	}
	
	person := PersonWithAddress{
		Name: "John",
		Age:  25,
		Address: Address{
			Street: "123 Main St",
			City:   "NYC",
		},
	}
	
	put := Put[PersonWithAddress]{}
	values, err := put.Write(person)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Put returns the struct itself as single element
	expected := []any{person}
	if !reflect.DeepEqual(values, expected) {
		t.Errorf("Expected %v, got %v", expected, values)
	}
}

func TestWriter(t *testing.T) {
	person := TestPerson{
		Name: "Alice",
		Age:  28,
		City: "Boston",
	}
	
	writer := Writer[TestPerson]()
	values, err := writer(person)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// Writer for struct should use product system and flatten fields
	if len(values) == 0 {
		t.Error("Expected non-empty values for struct")
	}
	
	// Should contain the struct field values
	containsName := false
	containsAge := false
	containsCity := false
	
	for _, v := range values {
		switch v {
		case "Alice":
			containsName = true
		case 28:
			containsAge = true
		case "Boston":
			containsCity = true
		}
	}
	
	if !containsName || !containsAge || !containsCity {
		t.Errorf("Missing expected field values in: %v", values)
	}
}

func TestWriterF(t *testing.T) {
	// Test WriterF function creation
	writerFunc := Writer[TestPerson]()
	
	if writerFunc == nil {
		t.Error("Writer should return non-nil function")
	}
	
	person := TestPerson{
		Name: "Bob",
		Age:  35,
		City: "Chicago",
	}
	
	values, err := writerFunc(person)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(values) == 0 {
		t.Error("Expected non-empty values")
	}
}

func TestWrite_Interface(t *testing.T) {
	// Test that Put implements Write interface
	var writer Write[TestPerson]
	
	put := Put[TestPerson]{}
	writer = put
	
	if writer == nil {
		t.Error("Put should implement Write interface")
	}
	
	person := TestPerson{Name: "Test", Age: 20, City: "Test City"}
	values, err := writer.Write(person)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	if len(values) != 1 {
		t.Errorf("Expected 1 value, got %d", len(values))
	}
}

func TestWrite_TypeSafety(t *testing.T) {
	// Test that Write is type-safe at compile time
	put := Put[TestPerson]{}
	
	person := TestPerson{Name: "John", Age: 25, City: "NYC"}
	values, err := put.Write(person)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// This should compile and work correctly
	if len(values) == 0 {
		t.Error("Expected non-empty values")
	}
	
	// The following should not compile (type safety):
	// put.Write("string") // Would be compile error
	// put.Write(123)      // Would be compile error
}

func TestWrite_MultipleValues(t *testing.T) {
	// Test writing multiple values of the same type
	people := []TestPerson{
		{Name: "John", Age: 25, City: "NYC"},
		{Name: "Jane", Age: 30, City: "LA"},
		{Name: "Bob", Age: 35, City: "Chicago"},
	}
	
	put := Put[TestPerson]{}
	
	var allValues [][]any
	for _, person := range people {
		values, err := put.Write(person)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		allValues = append(allValues, values)
	}
	
	if len(allValues) != 3 {
		t.Errorf("Expected 3 sets of values, got %d", len(allValues))
	}
	
	// Each person should produce 1 value
	for i, values := range allValues {
		if len(values) != 1 {
			t.Errorf("Person %d should produce 1 value, got %d", i, len(values))
		}
	}
	
	// Check specific values
	if allValues[0][0] != people[0] || allValues[1][0] != people[1] || allValues[2][0] != people[2] {
		t.Error("Values not extracted correctly")
	}
}
