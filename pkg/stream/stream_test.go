package stream

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	s := New(NextChunk[int]{
		run: func(context.Context) ([]int, bool, error) {
			return []int{1, 2, 3}, true, nil
		},
	})

	chunk, ok, err := s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("Expected ok to be true")
	}
	if !reflect.DeepEqual(chunk, []int{1, 2, 3}) {
		t.Fatalf("Expected [1, 2, 3], got %v", chunk)
	}
}

func TestFail(t *testing.T) {
	expectedErr := errors.New("test error")
	s := Fail[int](expectedErr)

	_, ok, err := s.Next(context.Background())
	if err != expectedErr {
		t.Fatalf("Expected error %v, got %v", expectedErr, err)
	}
	if ok {
		t.Fatal("Expected ok to be false")
	}
}

func TestFromSlice(t *testing.T) {
	s := FromSlice([]int{1, 2, 3})

	chunk, hasNext, err := s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hasNext {
		t.Fatal("Expected hasNext to be false")
	}
	if !reflect.DeepEqual(chunk, []int{1, 2, 3}) {
		t.Fatalf("Expected [1, 2, 3], got %v", chunk)
	}
}

func TestFrom(t *testing.T) {
	s := From(42)

	chunk, hasNext, err := s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hasNext {
		t.Fatal("Expected hasNextok to be true")
	}
	if !reflect.DeepEqual(chunk, []int{42}) {
		t.Fatalf("Expected [42], got %v", chunk)
	}
}

func TestEmpty(t *testing.T) {
	s := Empty[int]()

	chunk, hasNext, err := s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hasNext {
		t.Fatal("Expected ok to be false")
	}
	if chunk != nil {
		t.Fatalf("Expected nil chunk, got %v", chunk)
	}
}

func TestSucceed(t *testing.T) {
	s := Succeed(42)

	chunk, hasNext, err := s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hasNext {
		t.Fatal("Expected ok to be false")
	}
	if !reflect.DeepEqual(chunk, []int{42}) {
		t.Fatalf("Expected [42], got %v", chunk)
	}

	// Second call should return empty
	chunk, hasNext, err = s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hasNext {
		t.Fatal("Expected ok to be false on second call")
	}
	if chunk != nil {
		t.Fatalf("Expected nil chunk on second call, got %v", chunk)
	}
}

func TestTake(t *testing.T) {
	s := FromSlice([]int{1, 2, 3, 4, 5})
	s = s.Take(3)

	chunk, hasNext, err := s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hasNext {
		t.Fatal("Expected ok to be false")
	}
	if !reflect.DeepEqual(chunk, []int{1, 2, 3}) {
		t.Fatalf("Expected [1, 2, 3], got %v", chunk)
	}

	// Second call should return empty
	chunk, hasNext, err = s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hasNext {
		t.Fatal("Expected ok to be false on second call")
	}
	if chunk != nil {
		t.Fatalf("Expected nil chunk on second call, got %v", chunk)
	}
}

func TestMap(t *testing.T) {
	s := FromSlice([]int{1, 2, 3})
	s = Map(s, func(x int) int { return x * 2 })

	chunk, hasNext, err := s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hasNext {
		t.Fatal("Expected ok to be false")
	}
	if !reflect.DeepEqual(chunk, []int{2, 4, 6}) {
		t.Fatalf("Expected [2, 4, 6], got %v", chunk)
	}
}

func TestFilter(t *testing.T) {
	s := FromSlice([]int{1, 2, 3, 4, 5})
	s = Filter(s, func(x int) bool { return x%2 == 0 })

	chunk, hasNext, err := s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hasNext {
		t.Fatal("Expected ok to be false")
	}
	if !reflect.DeepEqual(chunk, []int{2, 4}) {
		t.Fatalf("Expected [2, 4], got %v", chunk)
	}
}

func ignoreTestFlatMap(t *testing.T) {
	s := FromSlice([]int{1, 2})
	s = FlatMap(s, func(x int) Stream[int] { return FromSlice([]int{x, x * 10}) })

	chunk, hasNext, err := s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hasNext {
		t.Fatal("Expected ok to be false")
	}
	if !reflect.DeepEqual(chunk, []int{1, 10}) {
		t.Fatalf("Expected [1, 10], got %v", chunk)
	}

	chunk, hasNext, err = s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hasNext {
		t.Fatal("Expected ok to be true")
	}
	if !reflect.DeepEqual(chunk, []int{2, 20}) {
		t.Fatalf("Expected [2, 20], got %v", chunk)
	}
}

func TestCollect(t *testing.T) {
	s := FromSlice([]int{1, 2, 3, 4, 5})
	result, err := s.Collect(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !reflect.DeepEqual(result, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("Expected [1, 2, 3, 4, 5], got %v", result)
	}
}

func TestZipWith(t *testing.T) {
	s1 := FromSlice([]int{1, 2, 3})
	s2 := FromSlice([]string{"a", "b", "c"})
	s := ZipWith(s1, s2, func(x int, y string) string { return fmt.Sprintf("%d%s", x, y) })

	chunk, hasNext, err := s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hasNext {
		t.Fatal("Expected ok to be false")
	}
	if !reflect.DeepEqual(chunk, []string{"1a", "2b", "3c"}) {
		t.Fatalf("Expected [1a 2b 3c], got %v", chunk)
	}
}

func TestParMap(t *testing.T) {
	s := FromSlice([]int{1, 2, 3, 4, 5})
	s = ParMap(s, 2, func(x int) (int, error) { return x * 2, nil })

	chunk, hasNext, err := s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if hasNext {
		t.Fatal("Expected hasNext to be false")
	}
	if !reflect.DeepEqual(chunk, []int{2, 4, 6, 8, 10}) {
		t.Fatalf("Expected [2, 4, 6, 8, 10], got %v", chunk)
	}
}

func TestBracket(t *testing.T) {
	acquired := false
	released := false
	s := Bracket(
		func() (int, error) { acquired = true; return 42, nil },
		func(x int) Stream[int] { return FromSlice([]int{x, x * 2}) },
		func(x int) error { released = true; return nil },
	)

	chunk, ok, err := s.Next(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ok {
		t.Fatal("Expected ok to be false")
	}
	if !reflect.DeepEqual(chunk, []int{42, 84}) {
		t.Fatalf("Expected [42, 84], got %v", chunk)
	}
	if !acquired {
		t.Fatal("Resource was not acquired")
	}
	if !released {
		t.Fatal("Resource was not released")
	}
}
