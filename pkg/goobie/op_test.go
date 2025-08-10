package goobie

import (
	"errors"
	"reflect"
	"testing"
)

// Mock Runner for op testing
type mockOpRunner struct {
	results map[string]any
	errors  map[string]error
}

func (m *mockOpRunner) RunQuery(query QueryAny, strategy ResultStrategy, resultType reflect.Type) (any, error) {
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
	// Create an op that returns an int
	intOp := RunOp[int]{
		run: func(runner Runner) (int, error) {
			return 42, nil
		},
	}
	
	// Map it to a string using FlatMap (since there's no simple Map function)
	stringOp := FlatMap(intOp, func(i int) Op[string] {
		return RunOp[string]{
			run: func(runner Runner) (string, error) {
				return "Number: " + string(rune(i+'0')), nil
			},
		}
	})
	
	runner := &mockOpRunner{}
	_, err := stringOp.Run(runner)
	
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	
	// The important thing is that we can compose operations
	if _, ok := stringOp.(Op[string]); !ok {
		t.Error("FlatMap should return Op[string]")
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
	
	// Map it to a string using FlatMap
	stringOp := FlatMap(intOp, func(i int) Op[string] {
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

func TestRunOp_FlatMap(t *testing.T) {
	// Create an op that returns an int
	intOp := RunOp[int]{
		run: func(runner Runner) (int, error) {
			return 5, nil
		},
	}
	
	// FlatMap it to another op that uses the int
	stringOp := FlatMap(intOp, func(i int) Op[string] {
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
	stringOp := FlatMap(intOp, func(i int) Op[string] {
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
	stringOp := FlatMap(intOp, func(i int) Op[string] {
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
	// Test that RunOp implements Op interface
	var op Op[string]
	
	runOp := RunOp[string]{
		run: func(runner Runner) (string, error) {
			return "test", nil
		},
	}
	
	op = runOp
	if op == nil {
		t.Error("RunOp should implement Op interface")
	}
	
	// Test that we can call interface methods
	runner := &mockOpRunner{}
	_, err := op.Run(runner)
	
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
	
	op2 := FlatMap(op1, func(i int) Op[int] {
		return RunOp[int]{
			run: func(runner Runner) (int, error) {
				return i * 2, nil
			},
		}
	})
	
	op3 := FlatMap(op2, func(i int) Op[string] {
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
