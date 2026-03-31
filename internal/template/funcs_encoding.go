package template

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/hex"
	"io"

	"github.com/spf13/cast"
)

func (e *Engine) registerEncodingFuncs() {
	e.funcs["base64Encode"] = func(s interface{}) string {
		return base64.StdEncoding.EncodeToString([]byte(cast.ToString(s)))
	}

	e.funcs["base64Decode"] = func(s interface{}) (string, error) {
		b, err := base64.StdEncoding.DecodeString(cast.ToString(s))
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	e.funcs["hexEncode"] = func(s interface{}) string {
		return hex.EncodeToString([]byte(cast.ToString(s)))
	}

	e.funcs["hexDecode"] = func(s interface{}) (string, error) {
		b, err := hex.DecodeString(cast.ToString(s))
		if err != nil {
			return "", err
		}
		return string(b), nil
	}

	e.funcs["gzip"] = func(s interface{}) (string, error) {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		if _, err := w.Write([]byte(cast.ToString(s))); err != nil {
			return "", err
		}
		if err := w.Close(); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	e.funcs["gunzip"] = func(s interface{}) (string, error) {
		r, err := gzip.NewReader(bytes.NewReader([]byte(cast.ToString(s))))
		if err != nil {
			return "", err
		}
		defer r.Close()
		b, err := io.ReadAll(r)
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
}
