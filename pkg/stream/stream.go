package stream

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

// Stream represents a sequence of elements that can be consumed asynchronously. The streams
// are pull based meaning that the consumer can request elements from the stream at its own pace.
// The implementation encapsulates a NextChunk function which returns items.
type Stream[A any] struct {
	// next is the underlying NextChunk that produces data for this Stream
	next   NextChunk[A]
	closer *closeState
}

type closeState struct {
	once sync.Once
	fn   func() error
	err  error
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

func newDerived[A any](p NextChunk[A], closer *closeState) Stream[A] {
	return Stream[A]{next: p, closer: closer}
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

// FromNextChunkFuncWithClose creates a stream that releases a resource when closed.
func FromNextChunkFuncWithClose[A any](
	f func(context.Context) ([]A, bool, error),
	closeFn func() error,
) Stream[A] {
	return Stream[A]{
		next:   NextChunk[A]{run: f},
		closer: &closeState{fn: closeFn},
	}
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
		_ = s.Close()
		return nil, false, ctx.Err()
	default:
		return s.next.run(ctx)
	}
}

// Close releases resources owned by the stream. It is safe to call more than once.
func (s Stream[A]) Close() error {
	if s.closer == nil {
		return nil
	}
	s.closer.once.Do(func() {
		s.closer.err = s.closer.fn()
	})
	return s.closer.err
}

func (s Stream[A]) Take(n int) Stream[A] {
	remaining := n
	return newDerived(NextChunk[A]{
		run: func(ctx context.Context) ([]A, bool, error) {
			if remaining <= 0 {
				return nil, false, s.Close()
			}
			chunk, hasNext, err := s.Next(ctx)
			if len(chunk) > remaining {
				chunk = chunk[:remaining]
			}
			remaining -= len(chunk)
			if err != nil {
				return chunk, false, errors.Join(err, s.Close())
			}
			if remaining == 0 || !hasNext {
				return chunk, false, s.Close()
			}
			return chunk, true, nil
		},
	}, s.closer)
}

func (s Stream[A]) ForEach(ctx context.Context, f func(A)) {
	defer s.Close()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		chunk, ok, err := s.Next(ctx)
		for _, a := range chunk {
			f(a)
		}
		if !ok || err != nil {
			return
		}
	}
}

func (s Stream[A]) Collect(ctx context.Context) ([]A, error) {
	defer s.Close()

	result := make([]A, 0)
	for {
		chunk, hasNext, err := s.Next(ctx)
		result = append(result, chunk...)
		if err != nil {
			return result, err
		}
		if !hasNext {
			return result, nil
		}
	}
}

func (s Stream[A]) Map[B any](f func(A) B) Stream[B] {
	return newDerived(NextChunk[B]{
		run: func(ctx context.Context) ([]B, bool, error) {
			chunk, hasNext, err := s.Next(ctx)
			result := make([]B, len(chunk))
			for i, a := range chunk {
				result[i] = f(a)
			}
			return result, hasNext, err
		},
	}, s.closer)
}

// UnsafeCast maps a stream of any to a stream of a specific type.
// This method is unsafe and can cause runtime panics.
func (s Stream[A]) UnsafeCast[B any]() Stream[B] {
	return s.Map(func(a A) B {
		return any(a).(B)
	})
}

func (s Stream[A]) MapError(f func(error) error) Stream[A] {
	return newDerived(NextChunk[A]{
		run: func(ctx context.Context) ([]A, bool, error) {
			chunk, ok, err := s.Next(ctx)
			if !ok || err != nil {
				return chunk, ok, f(err)
			}
			return chunk, ok, nil
		},
	}, s.closer)
}

func (s Stream[A]) FlatMap[B any](f func(A) Stream[B]) Stream[B] {
	var innerStream Stream[B]
	var pending []A
	var outerDone bool
	var outerErr error
	closer := &closeState{fn: func() error {
		err := s.Close()
		if innerErr := innerStream.Close(); err == nil {
			err = innerErr
		}
		return err
	}}

	return newDerived(NextChunk[B]{
		run: func(ctx context.Context) ([]B, bool, error) {
			for {
				if innerStream.next.run != nil {
					chunk, hasNext, err := innerStream.Next(ctx)
					if !hasNext {
						err = errors.Join(err, innerStream.Close())
						innerStream = Stream[B]{}
					}
					if len(chunk) > 0 {
						more := hasNext || len(pending) > 0 || !outerDone
						if !more {
							err = errors.Join(err, outerErr)
							outerErr = nil
						}
						return chunk, more, err
					}
					if err != nil {
						return nil, false, errors.Join(err, outerErr)
					}
				}

				if len(pending) > 0 {
					next := pending[0]
					pending = pending[1:]
					innerStream = f(next)
					continue
				}

				if outerDone {
					err := outerErr
					outerErr = nil
					return nil, false, err
				}

				chunk, hasNext, err := s.Next(ctx)
				pending = chunk
				outerDone = !hasNext || err != nil
				outerErr = err
			}
		},
	}, closer)
}

func (s Stream[A]) Filter(pred func(A) bool) Stream[A] {
	return newDerived(NextChunk[A]{
		run: func(ctx context.Context) ([]A, bool, error) {
			for {
				chunk, hasNext, err := s.Next(ctx)
				result := make([]A, 0, len(chunk))
				for _, a := range chunk {
					if pred(a) {
						result = append(result, a)
					}
				}
				return result, hasNext, err
			}
		},
	}, s.closer)
}

func (s Stream[A]) ToMap[K comparable, V any](ctx context.Context, keyValue func(A) (K, V)) (map[K]V, error) {
	defer s.Close()

	result := make(map[K]V)
	for {
		chunk, hasNext, err := s.Next(ctx)
		for _, a := range chunk {
			key, value := keyValue(a)
			result[key] = value
		}
		if err != nil {
			return result, err
		}
		if !hasNext {
			return result, nil
		}
	}
}
func (s Stream[A]) Fold[B any](ctx context.Context, z B, f func(B, A) B) B {
	defer s.Close()

	acc := z
	for {
		chunk, ok, err := s.Next(ctx)
		for _, a := range chunk {
			acc = f(acc, a)
		}
		if !ok || err != nil {
			return acc
		}
	}
}

func (s Stream[A]) FoldChunksLeft[B any](ctx context.Context, z B, f func(B, []A) B) B {
	defer s.Close()

	acc := z
	for {
		chunk, ok, err := s.Next(ctx)
		acc = f(acc, chunk)
		if !ok || err != nil {
			return acc
		}
	}
}

func (s Stream[A]) ZipWith[B any, C any](other Stream[B], f func(A, B) C) Stream[C] {
	closer := &closeState{fn: func() error {
		err := s.Close()
		if otherErr := other.Close(); err == nil {
			err = otherErr
		}
		return err
	}}
	return newDerived(NextChunk[C]{
		run: func(ctx context.Context) ([]C, bool, error) {
			chunkA, okA, errA := s.Next(ctx)
			if errA != nil {
				return nil, okA, errA
			}
			chunkB, okB, errB := other.Next(ctx)
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
	}, closer)
}

func (s Stream[A]) ParMap[B any](concurrency int, f func(A) (B, error)) Stream[B] {
	return newDerived(NextChunk[B]{
		run: func(ctx context.Context) ([]B, bool, error) {
			chunk, hasNext, err := s.Next(ctx)

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

			return result, hasNext, err
		},
	}, s.closer)
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
