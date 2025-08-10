package goobie

type RunFn[A any] func(Runner) (A, error)

type RunOp[A any] struct {
	run RunFn[A]
}

func (q RunOp[A]) Run(runner Runner) (A, error) {
	return q.run(runner)
}

type Op[A any] interface {
	Run(Runner) (A, error)
}

func NewOp[A any](run RunFn[A]) Op[A] {
	return RunOp[A]{run}
}

func ErrOp[A any](err error) Op[A] {
	var zero A
	return RunOp[A]{
		run: func(Runner) (A, error) {
			return zero, err
		},
	}
}

func FlatMap[A any, B any](op Op[A], f func(A) Op[B]) Op[B] {
	return RunOp[B]{
		run: func(runner Runner) (B, error) {
			result, err := op.Run(runner)
			if err != nil {
				return *new(B), err
			}
			return f(result).Run(runner)
		},
	}
}

func MapN[A any](op ...Op[any]) func(f func(...any) A) Op[A] {
	return func(f func(...any) A) Op[A] {
		return RunOp[A]{
			run: func(runner Runner) (A, error) {
				var results []any
				for _, o := range op {
					result, err := o.Run(runner)
					if err != nil {
						return *new(A), err
					}
					results = append(results, result)
				}
				return f(results...), nil
			},
		}
	}
}
func Map2[A any, B any, C any](opA Op[A], opB Op[B]) func(f func(A, B) C) Op[C] {
	return func(f func(A, B) C) Op[C] {
		return RunOp[C]{
			run: func(runner Runner) (C, error) {
				resultA, err := opA.Run(runner)
				if err != nil {
					return *new(C), err
				}
				resultB, err := opB.Run(runner)
				if err != nil {
					return *new(C), err
				}
				return f(resultA, resultB), nil
			},
		}
	}
}
