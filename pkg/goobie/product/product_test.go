package product

import (
	"testing"
)

// Sample structs for testing
type TestStruct struct {
	IntField    int
	StringField string
	FloatField  float64
}

type TestStructWithUnexported struct {
	ExportedField   string
	unexportedField int
}

func TestNewProduct(t *testing.T) {
	product := NewProduct[TestStruct]()

	if _, ok := product.(*Productizer[TestStruct]); !ok {
		t.Errorf("NewProduct did not return a *Productizer")
	}
}

func TestNewProductFailsOnNonStruct(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic, got nil")
		}
	}()

	NewProduct[int]()
}

func TestProductizer_Values(t *testing.T) {
	test := TestStruct{
		IntField:    1,
		StringField: "test",
		FloatField:  1.1,
	}
	product := NewProductFrom(&test)

	values := product.Values()
	if len(values) != 3 {
		t.Errorf("Expected 3 values, got %d", len(values))
	}

	if values[0] != 1 || values[1] != "test" || values[2] != 1.1 {
		t.Errorf("Values not as expected: %v", values)
	}
}

func TestProductizer_Mutate(t *testing.T) {
	product := NewProduct[TestStruct]()

	pointers := product.Mutate()
	if len(pointers) != 3 {
		t.Errorf("Expected 3 pointers, got %d", len(pointers))
	}

	// Check that we can mutate through these pointers
	*pointers[0].(*int) = 2
	*pointers[1].(*string) = "modified"
	*pointers[2].(*float64) = 2.2

	values := product.Values()
	if values[0] != 2 || values[1] != "modified" || values[2] != 2.2 {
		t.Errorf("Mutation did not work as expected: %v", values)
	}
}

func TestProductizer_Read(t *testing.T) {
	product := NewProduct[TestStruct]()

	mockScan := func(dest ...any) error {
		*dest[0].(*int) = 42
		*dest[1].(*string) = "scanned"
		*dest[2].(*float64) = 3.14
		return nil
	}

	result, err := product.(*Productizer[TestStruct]).Read(mockScan)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if result.IntField != 42 || result.StringField != "scanned" || result.FloatField != 3.14 {
		t.Errorf("Read did not populate struct as expected: %v", result)
	}

	v := product.Value()
	if v.IntField != 42 || v.StringField != "scanned" || v.FloatField != 3.14 {
		t.Errorf("Value did not return the struct as expected: %v", v)
	}
}

func TestProductizer_WithUnexportedFields(t *testing.T) {
	product := NewProduct[TestStructWithUnexported]()

	values := product.Values()
	if len(values) != 1 {
		t.Errorf("Expected 1 values, got %d", len(values))
	}

	pointers := product.Mutate()
	if len(pointers) != 1 {
		t.Errorf("Expected 1 pointer (for exported field), got %d", len(pointers))
	}
}

func TestProductizer_TypeChecking(t *testing.T) {
	var _ Product[TestStruct] = &Productizer[TestStruct]{}
	var _ Product[TestStructWithUnexported] = &Productizer[TestStructWithUnexported]{}
}
