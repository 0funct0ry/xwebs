package template

import (
	"time"

	"github.com/spf13/cast"
)

var startTime = time.Now()

func (e *Engine) registerTimeFuncs() {
	e.funcs["now"] = func() time.Time {
		return time.Now()
	}

	e.funcs["nowUnix"] = func() int64 {
		return time.Now().Unix()
	}

	e.funcs["nowUnixMilli"] = func() int64 {
		return time.Now().UnixMilli()
	}

	e.funcs["nowUnixNano"] = func() int64 {
		return time.Now().UnixNano()
	}

	e.funcs["formatTime"] = func(format string, t interface{}) string {
		var tm time.Time
		switch val := t.(type) {
		case time.Time:
			tm = val
		case int64:
			tm = time.Unix(val, 0)
		case string:
			// Try parsing with common formats
			var err error
			tm, err = cast.ToTimeE(val)
			if err != nil {
				return ""
			}
		default:
			tm = cast.ToTime(val)
		}
		return tm.Format(format)
	}

	e.funcs["parseTime"] = func(format, value string) (time.Time, error) {
		return time.Parse(format, value)
	}

	e.funcs["duration"] = func(value interface{}) (time.Duration, error) {
		return time.ParseDuration(cast.ToString(value))
	}

	e.funcs["since"] = func(t interface{}) time.Duration {
		return time.Since(cast.ToTime(t))
	}

	e.funcs["until"] = func(t interface{}) time.Duration {
		return time.Until(cast.ToTime(t))
	}

	e.funcs["uptime"] = func() time.Duration {
		return time.Since(startTime)
	}
}
