package product

import (
	"github.com/dpovey/goobie/pkg/goobie/read"
	"reflect"
	"sync"
	"unsafe"
)

var (
	typeCacheMutex sync.RWMutex
	typeCache      = make(map[reflect.Type]*typeInfo)
	typeCacheOnce  = make(map[reflect.Type]*sync.Once)
)

type typeInfo struct {
	fields     []reflect.StructField
	ptrOffsets []uintptr
}

// Product is an interface for a "Product" type that consists of
// a number of fields.  Instances of Product can support a
type Product[A any] interface {
	Read(read.RowScan) (A, error)
	Value() A
	Values() []any
	Mutate() []any
}

type Productizer[A any] struct {
	value    reflect.Value
	instance *A
	typeInfo *typeInfo
}

func NewProduct[A any]() Product[A] {
	return NewProductFrom(new(A))
}

func NewProductFrom[A any](instance *A) Product[A] {
	v := reflect.ValueOf(instance)
	t := v.Elem().Type()
	if t.Kind() != reflect.Struct {
		panic("NewProduct requires a struct type")
	}

	ti := getOrCreateTypeInfo(t)

	return &Productizer[A]{value: v, instance: instance, typeInfo: ti}
}

func getOrCreateTypeInfo2(t reflect.Type) *typeInfo {
	once, _ := typeCacheOnce[t]
	if once == nil {
		once = &sync.Once{}
		typeCacheOnce[t] = once
	}
	once.Do(func() {
		typeCache[t] = makeTypeInfo(t)
	})
	return typeCache[t]
}

func getOrCreateTypeInfo(t reflect.Type) *typeInfo {
	typeCacheMutex.RLock()
	ti, ok := typeCache[t]
	typeCacheMutex.RUnlock()

	if ok {
		return ti
	}

	typeCacheMutex.Lock()
	defer typeCacheMutex.Unlock()

	// Double-check in case another goroutine created it while we were waiting
	if ti, ok = typeCache[t]; ok {
		return ti
	}

	ti = makeTypeInfo(t)

	typeCache[t] = ti
	return ti
}

func makeTypeInfo(t reflect.Type) *typeInfo {
	n := t.NumField()
	ti := &typeInfo{
		fields:     make([]reflect.StructField, 0, n),
		ptrOffsets: make([]uintptr, 0, n),
	}
	for i := 0; i < n; i++ {
		field := t.Field(i)
		if field.IsExported() {
			ti.fields = append(ti.fields, field)
			ti.ptrOffsets = append(ti.ptrOffsets, field.Offset)
		}
	}
	return ti
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

func (product *Productizer[A]) Write() ([]any, error) {
	return product.Values(), nil
}

func (product *Productizer[A]) Values() []any {
	baseAddr := unsafe.Pointer(product.instance)
	values := make([]any, len(product.typeInfo.ptrOffsets))
	for i, offset := range product.typeInfo.ptrOffsets {
		fieldAddr := unsafe.Add(baseAddr, offset)
		values[i] = reflect.NewAt(product.typeInfo.fields[i].Type, fieldAddr).Elem().Interface()
	}
	return values
}

func (product *Productizer[A]) Mutate() []any {
	baseAddr := unsafe.Pointer(product.instance)
	ptrs := make([]any, len(product.typeInfo.ptrOffsets))
	for i, offset := range product.typeInfo.ptrOffsets {
		fieldAddr := unsafe.Add(baseAddr, offset)
		// Convert unsafe.Pointer back to the appropriate pointer type
		ptrs[i] = reflect.NewAt(product.typeInfo.fields[i].Type, fieldAddr).Interface()
	}
	return ptrs
}

// ValuesSlow is a naive version of Values. It is about twice as slow
// as Values.  It is included for comparison purposes.
func (product *Productizer[A]) ValuesSlow() []any {
	v := product.value.Elem()
	var values []any
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).IsExported() {
			values = append(values, v.Field(i).Interface())
		}
	}
	return values
}

// MutateSlow is a naive version of Mutate. It is about four times as slow
// as Mutate.  It is included for comparison purposes.
func (product *Productizer[A]) MutateSlow() []any {
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
