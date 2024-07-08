package goobie

type UpdateT struct {
	SQL Fragment
}

func Update(sql string, args ...any) UpdateT {
	return UpdateF(Fr(sql, args...))
}

func UpdateAll(ops ...UpdateT) UpdateOp {
	var run RunFn[int64] = func(runner Runner) (int64, error) {
		var total int64
		for _, op := range ops {
			n, err := op.Run(runner)
			if err != nil {
				return total, err
			}
			total += n
		}
		return total, nil
	}
	return NewOp[int64](run)
}

func UpdateF(fragment Fragment) UpdateT {
	return UpdateT{SQL: fragment}
}

func (u UpdateT) Run(runner Runner) (int64, error) {
	return u.Update().Run(runner)
}

func (u UpdateT) Update() UpdateOp {
	var run RunFn[int64] = func(runner Runner) (int64, error) {
		return runner.RunUpdate(u)
	}
	return Op[int64]{run}
}

// TODO batch operation? Need to understand underlying support.
