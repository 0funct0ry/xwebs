package template

import (
	"math"
	"math/rand"
	"time"

	"github.com/spf13/cast"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func (e *Engine) registerMathFuncs() {
	e.funcs["add"] = func(a, b interface{}) interface{} {
		return cast.ToFloat64(a) + cast.ToFloat64(b)
	}

	e.funcs["sub"] = func(a, b interface{}) interface{} {
		return cast.ToFloat64(a) - cast.ToFloat64(b)
	}

	e.funcs["mul"] = func(a, b interface{}) interface{} {
		return cast.ToFloat64(a) * cast.ToFloat64(b)
	}

	e.funcs["div"] = func(a, b interface{}) interface{} {
		divisor := cast.ToFloat64(b)
		if divisor == 0 {
			return 0
		}
		return cast.ToFloat64(a) / divisor
	}

	e.funcs["mod"] = func(a, b interface{}) interface{} {
		return cast.ToInt(a) % cast.ToInt(b)
	}

	e.funcs["max"] = func(a, b interface{}) interface{} {
		af, bf := cast.ToFloat64(a), cast.ToFloat64(b)
		if af > bf {
			return af
		}
		return bf
	}

	e.funcs["min"] = func(a, b interface{}) interface{} {
		af, bf := cast.ToFloat64(a), cast.ToFloat64(b)
		if af < bf {
			return af
		}
		return bf
	}

	e.funcs["round"] = func(a interface{}) float64 {
		return math.Round(cast.ToFloat64(a))
	}

	e.funcs["seq"] = func(start, end interface{}) []int {
		s, e := cast.ToInt(start), cast.ToInt(end)
		var res []int
		if s <= e {
			for i := s; i <= e; i++ {
				res = append(res, i)
			}
		} else {
			for i := s; i >= e; i-- {
				res = append(res, i)
			}
		}
		return res
	}

	e.funcs["toInt"] = func(v interface{}) int {
		return cast.ToInt(v)
	}

	e.funcs["toFloat"] = func(v interface{}) float64 {
		return cast.ToFloat64(v)
	}

	e.funcs["random"] = func(min, max interface{}) int {
		mi, ma := cast.ToInt(min), cast.ToInt(max)
		if mi >= ma {
			return mi
		}
		return r.Intn(ma-mi) + mi
	}
}
