package stream

import (
	"context"
	"sync"
	"sync/atomic"
)

// Stream represents a sequence of elements that can be consumed asynchronously. The streams
// are pull based meaning that the consumer can request elements from the stream at its own pace.
// The implementation encapsulates a NextChunk function which returns items.
type Stream[A any] struct {
	// next is the underlying NextChunk that produces data for this Stream
	next NextChunk[A]
}

// NextChunk represents a function that, when run, produces a chunk of data, a boolean indicating
// if more data is available, and a possible error.
type NextChunk[A any] struct {
	// run is the core function of NextChunk. It takes a context and returns:
	// - A slice of type A, representing a chunk of data
	// - A boolean indicating if more data is available (true) or if the stream has ended (false)
	// - An error of type E, or nil if no error occurred
	run func(context.Context) ([]A, bool, error)
}

func New[A any](p NextChunk[A]) Stream[A] {
	return Stream[A]{next: p}
}
func Fail[A any](err error) Stream[A] {
	return New(NextChunk[A]{
		run: func(context.Context) ([]A, bool, error) {
			return nil, false, err
		},
	})
}

func FromSlice[A any](chunk []A) Stream[A] {
	var emitted = atomic.Bool{}
	return New(NextChunk[A]{
		run: func(context.Context) ([]A, bool, error) {
			if len(chunk) == 0 || emitted.Load() || !emitted.CompareAndSwap(false, true) {
				return nil, false, nil
			}
			return chunk, false, nil
		},
	})
}

func From[A any](item A) Stream[A] {
	return FromSlice([]A{item})
}

func FromNextChunkFunc[A any](f func(context.Context) ([]A, bool, error)) Stream[A] {
	return New(NextChunk[A]{run: f})
}

func Empty[A any]() Stream[A] {
	return New(NextChunk[A]{
		run: func(context.Context) ([]A, bool, error) {
			return nil, false, nil // nil value of E
		},
	})
}

func Succeed[A any](item A) Stream[A] {
	return From(item)
}

func NewPull[A any](f func(context.Context) ([]A, bool, error)) NextChunk[A] {
	return NextChunk[A]{run: f}
}

func (s Stream[A]) Next(ctx context.Context) ([]A, bool, error) {
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
		return s.next.run(ctx)
	}
}

func (s Stream[A]) Take(n int) Stream[A] {
	remaining := n
	return New(NextChunk[A]{
		run: func(ctx context.Context) ([]A, bool, error) {
			var result []A
			for remaining > 0 {
				chunk, hasNext, err := s.Next(ctx)
				if err != nil {
					return result, hasNext, err
				}
				if len(chunk) > remaining {
					result = append(result, chunk[:remaining]...)
					remaining = 0
				} else {
					result = append(result, chunk...)
					remaining -= len(chunk)
				}
				if !hasNext || len(result) > 0 {
					return result, false, nil
				}
			}
			return nil, false, nil
		},
	})
}

func (s Stream[A]) ForEach(ctx context.Context, f func(A)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		chunk, ok, err := s.Next(ctx)
		if !ok || err != nil {
			return
		}
		for _, a := range chunk {
			f(a)
		}
	}
}

func (s Stream[A]) Collect(ctx context.Context) ([]A, error) {
	result := make([]A, 0)
	for {
		chunk, hasNext, err := s.Next(ctx)
		if err != nil {
			return result, err
		}
		result = append(result, chunk...)
		if !hasNext {
			return result, nil
		}
	}
}

func Map[A any, B any](s Stream[A], f func(A) B) Stream[B] {
	return New(NextChunk[B]{
		run: func(ctx context.Context) ([]B, bool, error) {
			chunk, hasNext, err := s.Next(ctx)
			if err != nil {
				return nil, hasNext, err
			}
			result := make([]B, len(chunk))
			for i, a := range chunk {
				result[i] = f(a)
			}
			return result, hasNext, err
		},
	})
}

// UnsafeCast maps a stream of any to a stream of a specific type.
// This function is unsafe and can cause runtime panics.
func UnsafeCast[A any](s Stream[any]) Stream[A] {
	return Map(s, func(a any) A {
		return a.(A)
	})
}

func MapError[A any](s Stream[A], f func(error) error) Stream[A] {
	return New(NextChunk[A]{
		run: func(ctx context.Context) ([]A, bool, error) {
			chunk, ok, err := s.Next(ctx)
			if !ok || err != nil {
				return chunk, ok, f(err)
			}
			return chunk, ok, nil
		},
	})
}

func FlatMap[A any, B any](s Stream[A], f func(A) Stream[B]) Stream[B] {
	var innerStream Stream[B]

	return New(NextChunk[B]{
		run: func(ctx context.Context) ([]B, bool, error) {
			for {
				// If we have an inner stream, try to get elements from it
				if innerStream.next.run != nil {
					chunk, hasNext, err := innerStream.Next(ctx)
					if err != nil {
						return nil, hasNext, err
					}
					if len(chunk) > 0 {
						return chunk, hasNext, nil
					}
					if !hasNext {
						// Inner stream is exhausted, clear it
						innerStream = Stream[B]{}
					}
				}

				// Get the next chunk from the outer stream
				chunkA, hasNext, err := s.Next(ctx)
				if err != nil {
					return nil, hasNext, err
				}
				if len(chunkA) == 0 && !hasNext {
					return nil, hasNext, nil
				}

				// Find the first non-empty inner stream
				for _, a := range chunkA {
					innerStream = f(a)
					chunk, hasNext, err := innerStream.Next(ctx)
					if err != nil {
						return nil, hasNext, err
					}
					if len(chunk) > 0 {
						return chunk, hasNext, nil
					}
					if !hasNext {
						// Inner stream is exhausted, clear it
						innerStream = Stream[B]{}
					}
				}
				// If we've processed all elements and found no results, continue to the next outer chunk
			}
		},
	})
}

func Filter[A any](s Stream[A], pred func(A) bool) Stream[A] {
	return New(NextChunk[A]{
		run: func(ctx context.Context) ([]A, bool, error) {
			for {
				chunk, hasNext, err := s.Next(ctx)
				if err != nil {
					return nil, hasNext, err
				}
				result := make([]A, 0, len(chunk))
				for _, a := range chunk {
					if pred(a) {
						result = append(result, a)
					}
				}
				return result, hasNext, err
			}
		},
	})
}

func ToMap[A any, K comparable, V any](s Stream[A], ctx context.Context, keyValue func(A) (K, V)) (map[K]V, error) {
	result := make(map[K]V)
	for {
		chunk, hasNext, err := s.Next(ctx)
		if err != nil {
			return result, err
		}
		for _, a := range chunk {
			key, value := keyValue(a)
			result[key] = value
		}
		if !hasNext {
			return result, nil
		}
	}
}
func Fold[A any, B any](s Stream[A], ctx context.Context, z B, f func(B, A) B) B {
	acc := z
	for {
		chunk, ok, err := s.Next(ctx)
		if !ok || err != nil {
			return acc
		}
		for _, a := range chunk {
			acc = f(acc, a)
		}
	}
}

func FoldChunksLeft[A any, B any, E error](s Stream[A], ctx context.Context, z B, f func(B, []A) B) B {
	acc := z
	for {
		chunk, ok, err := s.Next(ctx)
		if !ok || err != nil {
			return acc
		}
		acc = f(acc, chunk)
	}
}

func ZipWith[A any, B any, C any](s1 Stream[A], s2 Stream[B], f func(A, B) C) Stream[C] {
	return New(NextChunk[C]{
		run: func(ctx context.Context) ([]C, bool, error) {
			chunkA, okA, errA := s1.Next(ctx)
			if errA != nil {
				return nil, okA, errA
			}
			chunkB, okB, errB := s2.Next(ctx)
			if errB != nil {
				return nil, okB, errB
			}

			minLen := len(chunkA)
			if len(chunkB) < minLen {
				minLen = len(chunkB)
			}

			result := make([]C, minLen)
			for i := 0; i < minLen; i++ {
				result[i] = f(chunkA[i], chunkB[i])
			}
			return result, okA && okB, nil
		},
	})
}

func ParMap[A any, B any](s Stream[A], concurrency int, f func(A) (B, error)) Stream[B] {
	return New(NextChunk[B]{
		run: func(ctx context.Context) ([]B, bool, error) {
			chunk, hasNext, err := s.Next(ctx)
			if err != nil {
				return nil, hasNext, err
			}

			result := make([]B, len(chunk))
			var wg sync.WaitGroup
			errChan := make(chan error, len(chunk))

			semaphore := make(chan struct{}, concurrency)

			for i, a := range chunk {
				wg.Add(1)
				go func(i int, a A) {
					defer wg.Done()
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					b, err := f(a)
					if err != nil {
						errChan <- err
						return
					}
					result[i] = b
				}(i, a)
			}

			wg.Wait()
			close(errChan)

			if err, ok := <-errChan; ok {
				return nil, false, err // Type assertion, be cautious
			}

			return result, hasNext, nil
		},
	})
}

func Bracket[A any, R any](
	acquire func() (R, error),
	use func(R) Stream[A],
	release func(R) error,
) Stream[A] {
	return New(NextChunk[A]{
		run: func(ctx context.Context) ([]A, bool, error) {
			resource, err := acquire()
			if err != nil {
				return nil, false, err // Type assertion, be cautious
			}

			stream := use(resource)
			chunk, ok, streamErr := stream.Next(ctx)

			if releaseErr := release(resource); releaseErr != nil {
				if streamErr != nil {
					// If both stream and release produce errors, you might want to combine them somehow
					return nil, false, streamErr
				}
				return nil, false, releaseErr
			}

			return chunk, ok, streamErr
		},
	})
}
