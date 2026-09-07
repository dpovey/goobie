package goobie

type RunFn[A any] func(Runner) (A, error)

// Op represents a computation that requires a Runner and produces an A.
type Op[A any] struct {
	run RunFn[A]
}

// RunOp is retained as an alias for compatibility.
type RunOp[A any] = Op[A]

func (op Op[A]) Run(runner Runner) (A, error) {
	return op.run(runner)
}

func NewOp[A any](run RunFn[A]) Op[A] {
	return Op[A]{run}
}

func ErrOp[A any](err error) Op[A] {
	var zero A
	return Op[A]{
		run: func(Runner) (A, error) {
			return zero, err
		},
	}
}

// Map transforms the successful result of op.
func (op Op[A]) Map[B any](f func(A) B) Op[B] {
	return Op[B]{
		run: func(runner Runner) (B, error) {
			result, err := op.Run(runner)
			if err != nil {
				var zero B
				return zero, err
			}
			return f(result), nil
		},
	}
}

// FlatMap sequences op with an operation selected from its successful result.
func (op Op[A]) FlatMap[B any](f func(A) Op[B]) Op[B] {
	return Op[B]{
		run: func(runner Runner) (B, error) {
			result, err := op.Run(runner)
			if err != nil {
				var zero B
				return zero, err
			}
			return f(result).Run(runner)
		},
	}
}

// Map2 combines successful results from op and other.
func (op Op[A]) Map2[B any, C any](other Op[B], f func(A, B) C) Op[C] {
	return Op[C]{
		run: func(runner Runner) (C, error) {
			resultA, err := op.Run(runner)
			if err != nil {
				var zero C
				return zero, err
			}
			resultB, err := other.Run(runner)
			if err != nil {
				var zero C
				return zero, err
			}
			return f(resultA, resultB), nil
		},
	}
}

func MapN[A any](ops ...Op[any]) func(f func(...any) A) Op[A] {
	return func(f func(...any) A) Op[A] {
		return Op[A]{
			run: func(runner Runner) (A, error) {
				var results []any
				for _, op := range ops {
					result, err := op.Run(runner)
					if err != nil {
						var zero A
						return zero, err
					}
					results = append(results, result)
				}
				return f(results...), nil
			},
		}
	}
}
