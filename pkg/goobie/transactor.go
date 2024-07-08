package goobie

import "reflect"

// Runner the base interface for Transactor and Transaction.
type Runner interface {
	RunQuery(query QueryAny, strategy ResultStrategy, resultType reflect.Type) (any, error)
	RunUpdate(update UpdateT) (int64, error)
}

// Transactor interface
type Transactor interface {
	Runner
	Transact(func(Transaction) error) error
	Begin() (Transaction, error)
}

type Transaction interface {
	Runner
	Commit() error
	Rollback() error
}

type ResultStrategy int

const (
	EMPTY ResultStrategy = iota
	UNIQUE
	OPTION
	SLICE
	STREAM
)

type RunFn[A any] func(Runner) (A, error)

type Op[A any] struct {
	Run RunFn[A]
}

type UpdateOp = Op[int64]

func NewOp[A any](run RunFn[A]) Op[A] {
	return Op[A]{run}
}

func FlatMap[A any, B any](op Op[A], f func(A) Op[B]) Op[B] {
	return Op[B]{
		Run: func(runner Runner) (B, error) {
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
		return Op[A]{
			Run: func(runner Runner) (A, error) {
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
		return Op[C]{
			Run: func(runner Runner) (C, error) {
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
