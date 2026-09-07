package read

import (
	"errors"
	"testing"
)

func TestRow1Reader(t *testing.T) {
	got, err := (Row1[int]{}).Reader().Read(mockRowScan(42))
	if err != nil {
		t.Fatalf("Row1.Reader().Read() error = %v", err)
	}
	if got.V1 != 42 {
		t.Fatalf("Row1.Reader().Read() = %v, want 42", got)
	}
}

func TestRow2Map(t *testing.T) {
	reader := (Row2[int, string]{}).Map(func(id int, name string) string {
		return name
	})
	got, err := reader.Read(mockRowScan(7, "alice"))
	if err != nil {
		t.Fatalf("Row2.Map().Read() error = %v", err)
	}
	if got != "alice" {
		t.Fatalf("Row2.Map().Read() = %q, want alice", got)
	}
}

func TestRow4Map(t *testing.T) {
	type country struct {
		code       string
		name       string
		population int
		gnp        *float64
	}

	gnp := 12.5
	reader := (Row4[string, string, int, *float64]{}).Map(
		func(code string, name string, population int, gnp *float64) country {
			return country{code: code, name: name, population: population, gnp: gnp}
		},
	)
	got, err := reader.Read(mockRowScan("AUS", "Australia", 27_000_000, &gnp))
	if err != nil {
		t.Fatalf("Row4.Map().Read() error = %v", err)
	}
	if got.code != "AUS" || got.name != "Australia" || got.population != 27_000_000 || got.gnp != &gnp {
		t.Fatalf("Row4.Map().Read() = %+v", got)
	}
}

func TestRow22Map(t *testing.T) {
	reader := (Row22[
		int, int, int, int, int, int, int, int, int, int, int,
		int, int, int, int, int, int, int, int, int, int, int,
	]{}).Map(func(
		v1, v2, v3, v4, v5, v6, v7, v8, v9, v10, v11,
		v12, v13, v14, v15, v16, v17, v18, v19, v20, v21, v22 int,
	) int {
		return v1 + v2 + v3 + v4 + v5 + v6 + v7 + v8 + v9 + v10 + v11 +
			v12 + v13 + v14 + v15 + v16 + v17 + v18 + v19 + v20 + v21 + v22
	})

	got, err := reader.Read(mockRowScan(
		1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
		12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22,
	))
	if err != nil {
		t.Fatalf("Row22.Map().Read() error = %v", err)
	}
	if got != 253 {
		t.Fatalf("Row22.Map().Read() = %d, want 253", got)
	}
}

func TestRowReaderReturnsScanError(t *testing.T) {
	expectedErr := errors.New("scan failed")
	_, err := (Row2[int, string]{}).Reader().Read(func(...any) error {
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("Row2.Reader().Read() error = %v, want %v", err, expectedErr)
	}
}
