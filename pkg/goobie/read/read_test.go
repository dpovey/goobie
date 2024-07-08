package read

import (
	"errors"
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
	tests := []struct {
		name     string
		get      ReadAny
		scanFunc RowScan
		want     any
		wantErr  bool
	}{
		{
			name:     "Get Int",
			get:      Get[int]{},
			scanFunc: mockRowScan(42),
			want:     42,
			wantErr:  false,
		},
		{
			name:     "Get String",
			get:      Get[string]{},
			scanFunc: mockRowScan("hello"),
			want:     "hello",
			wantErr:  false,
		},
		{
			name: "Get Error",
			get:  Get[int]{},
			scanFunc: func(...any) error {
				return errors.New("scan error")
			},
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.get.ReadAny(tt.scanFunc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get.Values() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get.Values() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMap(t *testing.T) {
	intReader := Get[int]{}
	doubleInt := func(i int) int { return i * 2 }
	mappedReader := Map(intReader, doubleInt)

	got, err := mappedReader.Read(mockRowScan(21))
	if err != nil {
		t.Errorf("Map() error = %v", err)
		return
	}
	if got != 42 {
		t.Errorf("Map() = %v, want %v", got, 42)
	}
}

func TestCombine(t *testing.T) {
	intReader := Get[int]{}
	stringReader := Get[string]{}
	combinedReader := Combine(intReader, stringReader)

	got, err := combinedReader.Read(mockRowScan(42, "hello"))
	if err != nil {
		t.Errorf("Combine() error = %v", err)
		return
	}
	want := []any{42, "hello"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Combine() = %v, want %v", got, want)
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
	tests := []struct {
		name   string
		reader ReadAny
		input  any
		want   any
	}{
		{"IntRead", IntRead, 42, 42},
		{"Int8Read", Int8Read, int8(8), int8(8)},
		{"Int16Read", Int16Read, int16(16), int16(16)},
		{"Int32Read", Int32Read, int32(32), int32(32)},
		{"Int64Read", Int64Read, int64(64), int64(64)},
		{"UintRead", UintRead, uint(42), uint(42)},
		{"Uint8Read", Uint8Read, uint8(8), uint8(8)},
		{"Uint16Read", Uint16Read, uint16(16), uint16(16)},
		{"Uint32Read", Uint32Read, uint32(32), uint32(32)},
		{"Uint64Read", Uint64Read, uint64(64), uint64(64)},
		{"Float32Read", Float32Read, float32(3.14), float32(3.14)},
		{"Float64Read", Float64Read, 3.14, 3.14},
		{"BoolRead", BoolRead, true, true},
		{"StringRead", StringRead, "hello", "hello"},
		{"TimeRead", TimeRead, now, now},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.reader.ReadAny(mockRowScan(tt.input))
			if err != nil {
				t.Errorf("%s.Values() error = %v", tt.name, err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("%s.Values() = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
