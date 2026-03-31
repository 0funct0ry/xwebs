package template

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/spf13/cast"
)

// Engine is a wrapper around text/template that provides standard functions.
type Engine struct {
	funcs     template.FuncMap
	sandboxed bool
}

// New creates a new template engine with the standard functions registered.
func New(sandboxed bool) *Engine {
	e := &Engine{
		funcs:     make(template.FuncMap),
		sandboxed: sandboxed,
	}
	e.registerStringFuncs()
	e.registerJSONFuncs()
	e.registerEncodingFuncs()
	e.registerCryptoFuncs()
	e.registerTimeFuncs()
	e.registerMathFuncs()
	e.registerSystemFuncs()
	e.registerIDFuncs()
	e.registerCollectionFuncs()
	return e
}

// registerStringFuncs adds string manipulation functions to the engine's function map.
func (e *Engine) registerStringFuncs() {
	e.funcs["upper"] = func(s interface{}) string {
		return strings.ToUpper(cast.ToString(s))
	}
	e.funcs["lower"] = func(s interface{}) string {
		return strings.ToLower(cast.ToString(s))
	}
	e.funcs["trim"] = func(s interface{}) string {
		return strings.TrimSpace(cast.ToString(s))
	}
	e.funcs["replace"] = func(old, new, s interface{}) string {
		return strings.ReplaceAll(cast.ToString(s), cast.ToString(old), cast.ToString(new))
	}
	e.funcs["split"] = func(sep, s interface{}) []string {
		return strings.Split(cast.ToString(s), cast.ToString(sep))
	}
	e.funcs["join"] = func(sep, items interface{}) string {
		return strings.Join(cast.ToStringSlice(items), cast.ToString(sep))
	}
	e.funcs["contains"] = func(substr, s interface{}) bool {
		return strings.Contains(cast.ToString(s), cast.ToString(substr))
	}
	e.funcs["regexMatch"] = func(pattern, s interface{}) (bool, error) {
		return regexp.MatchString(cast.ToString(pattern), cast.ToString(s))
	}
	e.funcs["regexFind"] = func(pattern, s interface{}) (string, error) {
		re, err := regexp.Compile(cast.ToString(pattern))
		if err != nil {
			return "", err
		}
		return re.FindString(cast.ToString(s)), nil
	}
	e.funcs["regexReplace"] = func(pattern, repl, s interface{}) (string, error) {
		re, err := regexp.Compile(cast.ToString(pattern))
		if err != nil {
			return "", err
		}
		return re.ReplaceAllString(cast.ToString(s), cast.ToString(repl)), nil
	}
	e.funcs["shellEscape"] = func(s interface{}) string {
		str := cast.ToString(s)
		if str == "" {
			return "''"
		}
		const safe = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_./"
		isSafe := true
		for _, r := range str {
			if !strings.ContainsRune(safe, r) {
				isSafe = false
				break
			}
		}
		if isSafe {
			return str
		}
		return "'" + strings.ReplaceAll(str, "'", "'\\''") + "'"
	}
	e.funcs["urlEncode"] = func(s interface{}) string {
		return url.QueryEscape(cast.ToString(s))
	}
	e.funcs["quote"] = func(s interface{}) string {
		return strconv.Quote(cast.ToString(s))
	}
	e.funcs["truncate"] = func(length, s interface{}) string {
		str := cast.ToString(s)
		l := cast.ToInt(length)
		if l <= 0 {
			return ""
		}
		r := []rune(str)
		if len(r) <= l {
			return str
		}
		return string(r[:l]) + "..."
	}
	e.funcs["padLeft"] = func(length, s interface{}) string {
		str := cast.ToString(s)
		l := cast.ToInt(length)
		r := []rune(str)
		if len(r) >= l {
			return str
		}
		return strings.Repeat(" ", l-len(r)) + str
	}
	e.funcs["padRight"] = func(length, s interface{}) string {
		str := cast.ToString(s)
		l := cast.ToInt(length)
		r := []rune(str)
		if len(r) >= l {
			return str
		}
		return str + strings.Repeat(" ", l-len(r))
	}
	e.funcs["indent"] = func(count, s interface{}) string {
		str := cast.ToString(s)
		n := cast.ToInt(count)
		if n <= 0 {
			return str
		}
		pad := strings.Repeat(" ", n)
		lines := strings.Split(str, "\n")
		for i, line := range lines {
			if line != "" {
				lines[i] = pad + line
			}
		}
		return strings.Join(lines, "\n")
	}
}

// Execute renders the template string with the provided data.
func (e *Engine) Execute(name, text string, data interface{}) (string, error) {
	tmpl, err := template.New(name).Funcs(e.funcs).Parse(text)
	if err != nil {
		return "", fmt.Errorf("parsing template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template %s: %w", name, err)
	}

	return buf.String(), nil
}
