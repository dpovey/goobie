package main

import (
	"bytes"
	"os"
	"testing"
)

func TestGeneratedRowsAreCurrent(t *testing.T) {
	want, err := generate(maxArity)
	if err != nil {
		t.Fatalf("generate rows: %v", err)
	}
	got, err := os.ReadFile("../../pkg/goobie/read/row_generated.go")
	if err != nil {
		t.Fatalf("read generated rows: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("row_generated.go is stale; run go generate ./pkg/goobie/read")
	}
}
