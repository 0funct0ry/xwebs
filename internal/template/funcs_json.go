package template

import (
	"encoding/json"
	"strings"

	"github.com/itchyny/gojq"
	"github.com/spf13/cast"
)

func (e *Engine) registerJSONFuncs() {
	e.funcs["toJSON"] = func(v interface{}) (string, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	e.funcs["prettyJSON"] = func(v interface{}) (string, error) {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	e.funcs["compactJSON"] = func(v interface{}) (string, error) {
		var data interface{}
		err := json.Unmarshal([]byte(cast.ToString(v)), &data)
		if err != nil {
			return "", err
		}
		b, err := json.Marshal(data)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	e.funcs["fromJSON"] = func(s interface{}) (interface{}, error) {
		var v interface{}
		err := json.Unmarshal([]byte(cast.ToString(s)), &v)
		if err != nil {
			return nil, err
		}
		return v, nil
	}

	e.funcs["isJSON"] = func(s interface{}) bool {
		var v interface{}
		return json.Unmarshal([]byte(cast.ToString(s)), &v) == nil
	}

	e.funcs["jq"] = func(query string, v interface{}) (interface{}, error) {
		q, err := gojq.Parse(query)
		if err != nil {
			return nil, err
		}
		var input interface{}
		switch val := v.(type) {
		case string:
			if err := json.Unmarshal([]byte(val), &input); err != nil {
				input = val // Fallback if not JSON string
			}
		default:
			input = v
		}

		iter := q.Run(input)
		var results []interface{}
		for {
			v, ok := iter.Next()
			if !ok {
				break
			}
			if err, ok := v.(error); ok {
				return nil, err
			}
			results = append(results, v)
		}

		if len(results) == 0 {
			return nil, nil
		}
		if len(results) == 1 {
			return results[0], nil
		}
		return results, nil
	}

	e.funcs["mergeJSON"] = func(v1, v2 interface{}) (interface{}, error) {
		var d1, d2 map[string]interface{}
		
		// Helper to unmarshal if string
		toData := func(v interface{}) (map[string]interface{}, error) {
			var d map[string]interface{}
			switch val := v.(type) {
			case string:
				if err := json.Unmarshal([]byte(val), &d); err != nil {
					return nil, err
				}
			case map[string]interface{}:
				d = val
			default:
				// Try marshalling and unmarshalling
				b, err := json.Marshal(v)
				if err != nil {
					return nil, err
				}
				if err := json.Unmarshal(b, &d); err != nil {
					return nil, err
				}
			}
			return d, nil
		}

		d1, err := toData(v1)
		if err != nil {
			return nil, err
		}
		d2, err = toData(v2)
		if err != nil {
			return nil, err
		}

		// Simple shallow merge
		res := make(map[string]interface{})
		for k, v := range d1 {
			res[k] = v
		}
		for k, v := range d2 {
			res[k] = v
		}
		return res, nil
	}

	e.funcs["setJSON"] = func(key string, val, v interface{}) (interface{}, error) {
		var data map[string]interface{}
		switch d := v.(type) {
		case string:
			if err := json.Unmarshal([]byte(d), &data); err != nil {
				return nil, err
			}
		case map[string]interface{}:
			data = d
		default:
			b, _ := json.Marshal(v)
			_ = json.Unmarshal(b, &data)
		}
		if data == nil {
			data = make(map[string]interface{})
		}
		data[key] = val
		return data, nil
	}

	e.funcs["deleteJSON"] = func(key string, v interface{}) (interface{}, error) {
		var data map[string]interface{}
		switch d := v.(type) {
		case string:
			if err := json.Unmarshal([]byte(d), &data); err != nil {
				return nil, err
			}
		case map[string]interface{}:
			data = d
		default:
			b, _ := json.Marshal(v)
			_ = json.Unmarshal(b, &data)
		}
		if data != nil {
			delete(data, key)
		}
		return data, nil
	}

	e.funcs["jsonPath"] = func(path string, v interface{}) (interface{}, error) {
		// Use jq as an engine for jsonPath since it's more powerful and compliant
		// Map simple dot notation to jq
		if !strings.HasPrefix(path, ".") {
			path = "." + path
		}
		return e.funcs["jq"].(func(string, interface{}) (interface{}, error))(path, v)
	}
}
