package goobie

import (
	"errors"
	"testing"
)

// Mock Runner for op testing
type mockOpRunner struct {
	results map[string]any
	errors  map[string]error
}

func (m *mockOpRunner) Query(Fragment) (Rows, error) {
	return nil, errors.New("not implemented in mock")
}

func (m *mockOpRunner) RunUpdate(update UpdateT) (int64, error) {
	return 0, errors.New("not implemented in mock")
}

func (m *mockOpRunner) RunUpdateMany(update UpdateT, params [][]any) (int64, error) {
	return 0, errors.New("not implemented in mock")
}

func TestRunOp_Run(t *testing.T) {
	// Test successful operation
	op := RunOp[string]{
		run: func(runner Runner) (string, error) {
			return "success", nil
		},
	}

	runner := &mockOpRunner{}
	_, err := op.Run(runner)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRunOp_RunError(t *testing.T) {
	// Test error handling
	expectedErr := errors.New("operation failed")
	op := RunOp[string]{
		run: func(runner Runner) (string, error) {
			return "", expectedErr
		},
	}

	runner := &mockOpRunner{}
	_, err := op.Run(runner)

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestRunOp_Map(t *testing.T) {
	intOp := RunOp[int]{
		run: func(runner Runner) (int, error) {
			return 42, nil
		},
	}

	stringOp := intOp.Map(func(i int) string {
		return "Number: " + string(rune(i+'0'))
	})

	runner := &mockOpRunner{}
	result, err := stringOp.Run(runner)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != "Number: Z" {
		t.Errorf("Map() = %q, want %q", result, "Number: Z")
	}
}

func TestRunOp_MapError(t *testing.T) {
	// Create an op that returns an error
	expectedErr := errors.New("original error")
	intOp := RunOp[int]{
		run: func(runner Runner) (int, error) {
			return 0, expectedErr
		},
	}

	stringOp := intOp.Map(func(i int) string {
		return "Should not be called"
	})

	runner := &mockOpRunner{}
	_, err := stringOp.Run(runner)

	if err != expectedErr {
		t.Errorf("Expected original error %v, got %v", expectedErr, err)
	}
}

func TestRunOp_FlatMap(t *testing.T) {
	// Create an op that returns an int
	intOp := RunOp[int]{
		run: func(runner Runner) (int, error) {
			return 5, nil
		},
	}

	// FlatMap it to another op that uses the int
	stringOp := intOp.FlatMap(func(i int) Op[string] {
		return RunOp[string]{
			run: func(runner Runner) (string, error) {
				if i > 0 {
					return "positive", nil
				}
				return "non-positive", nil
			},
		}
	})

	runner := &mockOpRunner{}
	_, err := stringOp.Run(runner)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestRunOp_FlatMapError(t *testing.T) {
	// Create an op that returns an error
	expectedErr := errors.New("original error")
	intOp := RunOp[int]{
		run: func(runner Runner) (int, error) {
			return 0, expectedErr
		},
	}

	// FlatMap it - the function should not be called
	stringOp := intOp.FlatMap(func(i int) Op[string] {
		t.Error("FlatMap function should not be called on error")
		return RunOp[string]{
			run: func(runner Runner) (string, error) {
				return "Should not be called", nil
			},
		}
	})

	runner := &mockOpRunner{}
	_, err := stringOp.Run(runner)

	if err != expectedErr {
		t.Errorf("Expected original error %v, got %v", expectedErr, err)
	}
}

func TestRunOp_FlatMapInnerError(t *testing.T) {
	// Create an op that succeeds
	intOp := RunOp[int]{
		run: func(runner Runner) (int, error) {
			return 5, nil
		},
	}

	// FlatMap it to an op that fails
	expectedErr := errors.New("inner error")
	stringOp := intOp.FlatMap(func(i int) Op[string] {
		return RunOp[string]{
			run: func(runner Runner) (string, error) {
				return "", expectedErr
			},
		}
	})

	runner := &mockOpRunner{}
	_, err := stringOp.Run(runner)

	if err != expectedErr {
		t.Errorf("Expected inner error %v, got %v", expectedErr, err)
	}
}

func TestOp_Interface(t *testing.T) {
	runOp := RunOp[string]{
		run: func(runner Runner) (string, error) {
			return "test", nil
		},
	}

	runner := &mockOpRunner{}
	_, err := runOp.Run(runner)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestOp_Composition(t *testing.T) {
	// Test composing multiple operations
	op1 := RunOp[int]{
		run: func(runner Runner) (int, error) {
			return 10, nil
		},
	}

	op2 := op1.FlatMap(func(i int) Op[int] {
		return RunOp[int]{
			run: func(runner Runner) (int, error) {
				return i * 2, nil
			},
		}
	})

	op3 := op2.FlatMap(func(i int) Op[string] {
		return RunOp[string]{
			run: func(runner Runner) (string, error) {
				if i == 20 {
					return "twenty", nil
				}
				return "other", nil
			},
		}
	})

	runner := &mockOpRunner{}
	_, err := op3.Run(runner)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
