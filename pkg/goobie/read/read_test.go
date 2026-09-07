package read

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

// Mock RowScan function for testing
func mockRowScan(values ...any) RowScan {
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

func TestGet(t *testing.T) {
	gotInt, err := Get[int]().Read(mockRowScan(42))
	if err != nil || gotInt != 42 {
		t.Fatalf("Get[int]().Read() = %v, %v; want 42, nil", gotInt, err)
	}

	gotString, err := Get[string]().Read(mockRowScan("hello"))
	if err != nil || gotString != "hello" {
		t.Fatalf("Get[string]().Read() = %v, %v; want hello, nil", gotString, err)
	}

	expectedErr := errors.New("scan error")
	_, err = Get[int]().Read(func(...any) error { return expectedErr })
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Get[int]().Read() error = %v, want %v", err, expectedErr)
	}
}

func TestMap(t *testing.T) {
	intReader := Get[int]()
	formatInt := func(i int) string { return fmt.Sprintf("value-%d", i) }
	mappedReader := intReader.Map(formatInt)

	got, err := mappedReader.Read(mockRowScan(21))
	if err != nil {
		t.Errorf("Map() error = %v", err)
		return
	}
	if got != "value-21" {
		t.Errorf("Map() = %v, want %v", got, "value-21")
	}
}

type TestStruct struct {
	Int    int
	String string
	Nested NestedStruct
}

type NestedStruct struct {
	Bool bool
	Time time.Time
}

func TestPredefinedReaders(t *testing.T) {
	now := time.Now()
	assertRead(t, "IntRead", IntRead, 42)
	assertRead(t, "Int8Read", Int8Read, int8(8))
	assertRead(t, "Int16Read", Int16Read, int16(16))
	assertRead(t, "Int32Read", Int32Read, int32(32))
	assertRead(t, "Int64Read", Int64Read, int64(64))
	assertRead(t, "UintRead", UintRead, uint(42))
	assertRead(t, "Uint8Read", Uint8Read, uint8(8))
	assertRead(t, "Uint16Read", Uint16Read, uint16(16))
	assertRead(t, "Uint32Read", Uint32Read, uint32(32))
	assertRead(t, "Uint64Read", Uint64Read, uint64(64))
	assertRead(t, "Float32Read", Float32Read, float32(3.14))
	assertRead(t, "Float64Read", Float64Read, 3.14)
	assertRead(t, "BoolRead", BoolRead, true)
	assertRead(t, "StringRead", StringRead, "hello")
	assertRead(t, "TimeRead", TimeRead, now)
}

func assertRead[T any](t *testing.T, name string, reader Read[T], want T) {
	t.Helper()
	got, err := reader.Read(mockRowScan(want))
	if err != nil {
		t.Fatalf("%s.Read() error = %v", name, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s.Read() = %v, want %v", name, got, want)
	}
}
