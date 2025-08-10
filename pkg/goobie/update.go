package goobie

import (
	"github.com/dpovey/goobie/pkg/goobie/write"
)

// UpdateT is the interface that the Runner expects
type UpdateT interface {
	GetSQL() string
	GetArgs() []any
}

type UpdateF struct {
	SQL Fragment
}

func (u UpdateF) GetSQL() string {
	return u.SQL.SQL
}

func (u UpdateF) GetArgs() []any {
	return u.SQL.Args
}

func (u UpdateF) Run(runner Runner) (int64, error) {
	return runner.RunUpdate(u)
}

type Writer[A any] func(A) ([]any, error)

type UpdateWithWrite[A any] struct {
	SQL    string
	Writer write.WriterF[A]
	Values []A
}

func (u UpdateWithWrite[A]) GetSQL() string {
	return u.SQL
}

func (u UpdateWithWrite[A]) GetArgs() []any {
	if len(u.Values) == 0 {
		return []any{}
	}
	
	// Use the writer to convert the first value to args
	args, err := u.Writer(u.Values[0])
	if err != nil {
		return []any{} // Return empty on error - could be improved
	}
	return args
}

func NewUpdateT[A any](sql string, values ...A) UpdateWithWrite[A] {
	return UpdateWithWrite[A]{SQL: sql, Writer: write.Writer[A](), Values: values}
}

type UpdateOp[A any] struct {
	run RunFn[A]
}

func NewUpdateOp[A any](run RunFn[A]) UpdateOp[A] {
	return UpdateOp[A]{run}
}

func (u UpdateOp[A]) Run(runner Runner) (A, error) {
	return u.run(runner)
}

func Update(sql string, args ...any) UpdateF {
	return UpdateFr(Fr(sql, args...))
}

func Update0(sql string) UpdateF {
	return UpdateFr(Fr(sql))
}

func UpdateAll(ops ...UpdateF) UpdateOp[int64] {
	var run RunFn[int64] = func(runner Runner) (int64, error) {
		var total int64
		for _, op := range ops {
			n, err := runner.RunUpdate(op)
			if err != nil {
				return 0, err
			}
			total += n
		}
		return total, nil
	}
	return UpdateOp[int64]{run}
}

func UpdateFr(fragment Fragment) UpdateF {
	return UpdateF{SQL: fragment}
}

func (u UpdateWithWrite[A]) Run(runner Runner) (int64, error) {
	return runner.RunUpdate(u)
}

func (u UpdateWithWrite[A]) Update() UpdateOp[int64] {
	return NewUpdateOp(func(runner Runner) (int64, error) {
		return u.Run(runner)
	})
}

func (u UpdateWithWrite[A]) UpdateMany(params [][]any) UpdateOp[int64] {
	return NewUpdateOp(func(runner Runner) (int64, error) {
		return runner.RunUpdateMany(u, params)
	})
}
