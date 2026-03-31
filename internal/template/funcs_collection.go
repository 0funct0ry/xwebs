package template

import (
	"fmt"
	"sort"

	"github.com/spf13/cast"
)

func (e *Engine) registerCollectionFuncs() {
	e.funcs["default"] = func(d, v interface{}) interface{} {
		if v == nil || cast.ToString(v) == "" {
			return d
		}
		return v
	}

	e.funcs["coalesce"] = func(args ...interface{}) interface{} {
		for _, arg := range args {
			if arg != nil && cast.ToString(arg) != "" {
				return arg
			}
		}
		return nil
	}

	e.funcs["ternary"] = func(cond bool, t, f interface{}) interface{} {
		if cond {
			return t
		}
		return f
	}

	e.funcs["dict"] = func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("dict requires even number of arguments")
		}
		dict := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				key = cast.ToString(values[i])
			}
			dict[key] = values[i+1]
		}
		return dict, nil
	}

	e.funcs["list"] = func(v ...interface{}) []interface{} {
		return v
	}

	e.funcs["keys"] = func(v interface{}) ([]string, error) {
		m, err := cast.ToStringMapE(v)
		if err != nil {
			return nil, err
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return keys, nil
	}

	e.funcs["values"] = func(v interface{}) ([]interface{}, error) {
		m, err := cast.ToStringMapE(v)
		if err != nil {
			return nil, err
		}
		values := make([]interface{}, 0, len(m))
		for _, val := range m {
			values = append(values, val)
		}
		return values, nil
	}

	e.funcs["pick"] = func(keys []string, v interface{}) (map[string]interface{}, error) {
		m, err := cast.ToStringMapE(v)
		if err != nil {
			return nil, err
		}
		res := make(map[string]interface{})
		for _, k := range keys {
			if val, ok := m[k]; ok {
				res[k] = val
			}
		}
		return res, nil
	}

	e.funcs["omit"] = func(keys []string, v interface{}) (map[string]interface{}, error) {
		m, err := cast.ToStringMapE(v)
		if err != nil {
			return nil, err
		}
		res := make(map[string]interface{})
		omitSet := make(map[string]struct{})
		for _, k := range keys {
			omitSet[k] = struct{}{}
		}
		for k, val := range m {
			if _, ok := omitSet[k]; !ok {
				res[k] = val
			}
		}
		return res, nil
	}

	e.funcs["chunk"] = func(size int, v interface{}) [][]interface{} {
		items := cast.ToSlice(v)
		if size <= 0 || len(items) == 0 {
			return nil
		}
		var chunks [][]interface{}
		for i := 0; i < len(items); i += size {
			end := i + size
			if end > len(items) {
				end = len(items)
			}
			chunks = append(chunks, items[i:end])
		}
		return chunks
	}

	e.funcs["uniq"] = func(v interface{}) []interface{} {
		items := cast.ToSlice(v)
		if len(items) == 0 {
			return items
		}
		seen := make(map[interface{}]struct{})
		var res []interface{}
		for _, item := range items {
			if _, ok := seen[item]; !ok {
				seen[item] = struct{}{}
				res = append(res, item)
			}
		}
		return res
	}

	e.funcs["sortAlpha"] = func(v interface{}) []string {
		items := cast.ToStringSlice(v)
		sort.Strings(items)
		return items
	}

	e.funcs["reverse"] = func(v interface{}) []interface{} {
		items := cast.ToSlice(v)
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
		return items
	}

	e.funcs["first"] = func(v interface{}) interface{} {
		items := cast.ToSlice(v)
		if len(items) == 0 {
			return nil
		}
		return items[0]
	}

	e.funcs["last"] = func(v interface{}) interface{} {
		items := cast.ToSlice(v)
		if len(items) == 0 {
			return nil
		}
		return items[len(items)-1]
	}

	e.funcs["rest"] = func(v interface{}) []interface{} {
		items := cast.ToSlice(v)
		if len(items) <= 1 {
			return nil
		}
		return items[1:]
	}

	e.funcs["pluck"] = func(key string, v interface{}) []interface{} {
		items := cast.ToSlice(v)
		var res []interface{}
		for _, item := range items {
			m, err := cast.ToStringMapE(item)
			if err == nil {
				if val, ok := m[key]; ok {
					res = append(res, val)
				}
			}
		}
		return res
	}
}
