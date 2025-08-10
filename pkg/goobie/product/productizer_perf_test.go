package product

import (
	"reflect"
	"testing"
	"time"
)

// PerfTestStruct is a sample struct for testing
type PerfTestStruct struct {
	Int    int
	String string
	Float  float64
	Bool   bool
}

func BenchmarkProductizer(b *testing.B) {
	// Create a slice of test structs
	PerfTestStructs := make([]*PerfTestStruct, 1000)
	for i := range PerfTestStructs {
		PerfTestStructs[i] = &PerfTestStruct{
			Int:    i,
			String: "test",
			Float:  float64(i) * 1.5,
			Bool:   i%2 == 0,
		}
	}

	// Create Productizers
	productizers := make([]*Productizer[PerfTestStruct], len(PerfTestStructs))
	for i, ts := range PerfTestStructs {
		productizers[i] = NewProductFrom(ts).(*Productizer[PerfTestStruct])
	}

	// Benchmark Values
	b.Run("Values", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, p := range productizers {
				_ = p.Values()
			}
		}
	})

	// Benchmark ValuesSlow
	b.Run("ValuesSlow", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, p := range productizers {
				_ = p.ValuesSlow()
			}
		}
	})

	// Benchmark Mutate
	b.Run("Mutate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, p := range productizers {
				_ = p.Mutate()
			}
		}
	})

	// Benchmark MutateSlow
	b.Run("MutateSlow", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, p := range productizers {
				_ = p.MutateSlow()
			}
		}
	})
}

// TestProductizerCorrectness ensures that fast and slow methods return the same results
func TestProductizerCorrectness(t *testing.T) {
	perfTestStruct := &PerfTestStruct{
		Int:    42,
		String: "test",
		Float:  3.14,
		Bool:   true,
	}

	productizer := NewProductFrom(perfTestStruct).(*Productizer[PerfTestStruct])

	// Test Values
	fastValues := productizer.Values()
	slowValues := productizer.ValuesSlow()
	if !reflect.DeepEqual(fastValues, slowValues) {
		t.Errorf("Values mismatch: fast %v, slow %v", fastValues, slowValues)
	}

	// Test Mutate
	fastMutate := productizer.Mutate()
	slowMutate := productizer.MutateSlow()
	if !reflect.DeepEqual(fastMutate, slowMutate) {
		t.Errorf("Mutate mismatch: fast %v, slow %v", fastMutate, slowMutate)
	}
}

// Helper function to time execution
func timeExecution(f func()) time.Duration {
	start := time.Now()
	f()
	return time.Since(start)
}

// TestProductizerPerformance_ValuesMutate compares execution times
// for values and mutate
func TestProductizerPerformance(t *testing.T) {
	const iterations = 1000000

	perfTestStruct := &PerfTestStruct{
		Int:    42,
		String: "test",
		Float:  3.14,
		Bool:   true,
	}

	productizer := NewProductFrom(perfTestStruct).(*Productizer[PerfTestStruct])

	valuesFastTime := timeExecution(func() {
		for i := 0; i < iterations; i++ {
			productizer.Values()
		}
	})

	valuesSlowTime := timeExecution(func() {
		for i := 0; i < iterations; i++ {
			productizer.ValuesSlow()
		}
	})

	mutateFastTime := timeExecution(func() {
		for i := 0; i < iterations; i++ {
			productizer.Mutate()
		}
	})

	mutateSlowTime := timeExecution(func() {
		for i := 0; i < iterations; i++ {
			productizer.MutateSlow()
		}
	})

	t.Logf("Values Fast: %v", valuesFastTime)
	t.Logf("Values Slow: %v", valuesSlowTime)
	t.Logf("Mutate Fast: %v", mutateFastTime)
	t.Logf("Mutate Slow: %v", mutateSlowTime)

	if valuesFastTime >= valuesSlowTime {
		t.Errorf("Values: Fast method (%v) not faster than slow method (%v)", valuesFastTime, valuesSlowTime)
	}

	if mutateFastTime >= mutateSlowTime {
		t.Errorf("Mutate: Fast method (%v) not faster than slow method (%v)", mutateFastTime, mutateSlowTime)
	}
}

func TestProductizerPerformance_Creation(t *testing.T) {
	const iterations = 1000000

	perfTestStruct := &PerfTestStruct{
		Int:    42,
		String: "test",
		Float:  3.14,
		Bool:   true,
	}

	creationTime := timeExecution(func() {
		for i := 0; i < iterations; i++ {
			NewProductFrom(perfTestStruct).Mutate()
		}
	})
	t.Logf("Creation: %v. Op/s: %v", creationTime, iterations/creationTime.Seconds())
}
