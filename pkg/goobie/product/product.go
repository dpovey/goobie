package product

import (
	"github.com/dpovey/goobie/pkg/goobie/read"
	"reflect"
)

// Product is an interface for a "Product" type that consists of
// a number of fields.  Instances of Product can support a
type Product[A any] interface {
	read.Read[A]
	Value() A
	Values() []any
	Mutate() []any
}

func NewProduct[A any]() Product[A] {
	return NewProductFrom(new(A))
}

func NewProductFrom[A any](instance *A) Product[A] {
	v := reflect.ValueOf(instance)
	if reflect.TypeOf(*instance).Kind() != reflect.Struct {
		panic("NewProduct requires a struct type")
	}
	return &Productizer[A]{value: v, instance: instance}
}

type Productizer[A any] struct {
	value    reflect.Value
	instance *A
}

func (product *Productizer[A]) Value() A {
	return *product.instance
}

func (product *Productizer[A]) Read(scan read.RowScan) (A, error) {
	err := scan(product.Mutate()...)
	if err != nil {
		var zero A
		return zero, err
	}
	return *product.instance, nil
}

func (product *Productizer[A]) Values() []any {
	v := product.value.Elem()
	var values []any
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).IsExported() {
			values = append(values, v.Field(i).Interface())
		}
	}
	return values
}

func (product *Productizer[A]) Mutate() []any {
	v := product.value.Elem()
	var pointers []any
	var nrFields = v.NumField()
	for i := 0; i < nrFields; i++ {
		field := v.Field(i)
		if v.Type().Field(i).IsExported() && field.CanAddr() && field.CanSet() {
			ptr := field.Addr().Interface()
			pointers = append(pointers, ptr)
		}
	}
	return pointers
}
