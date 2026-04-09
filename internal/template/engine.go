package template

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cast"
)

// Engine is a wrapper around text/template that provides standard functions.
type Engine struct {
	funcs     template.FuncMap
	sandboxed bool
	session   map[string]interface{}
	sessionID string
}

// New creates a new template engine with the standard functions registered.
func New(sandboxed bool) *Engine {
	e := &Engine{
		funcs:     make(template.FuncMap),
		sandboxed: sandboxed,
		session:   make(map[string]interface{}),
		sessionID: "sess-" + cast.ToString(time.Now().UnixNano()),
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
	e.registerContextFuncs()
	e.registerConnFuncs()
	e.registerColorFuncs()
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

// registerContextFuncs adds session management functions to the engine's function map.
func (e *Engine) registerContextFuncs() {
	e.funcs["sessionSet"] = func(key string, value interface{}) string {
		e.session[key] = value
		return ""
	}
	e.funcs["sessionGet"] = func(key string) interface{} {
		return e.session[key]
	}
	e.funcs["sessionClear"] = func() string {
		e.session = make(map[string]interface{})
		return ""
	}
}

// Execute renders the template string with the provided data.
func (e *Engine) Execute(name, text string, data interface{}) (string, error) {
	if ctx, ok := data.(*TemplateContext); ok {
		// Merge engine session into context session
		if ctx.Session == nil {
			ctx.Session = make(map[string]interface{})
		}
		for k, v := range e.session {
			ctx.Session[k] = v
		}

		// Vars should contain both global handler variables and session variables
		if ctx.Vars == nil {
			ctx.Vars = make(map[string]interface{})
		}
		for k, v := range ctx.Session {
			ctx.Vars[k] = v
		}

		// Also populate environment variables if not already set
		if len(ctx.Env) == 0 && !e.sandboxed {
			ctx.Env = make(map[string]string)
			for _, env := range os.Environ() {
				parts := strings.SplitN(env, "=", 2)
				if len(parts) == 2 {
					ctx.Env[parts[0]] = parts[1]
				}
			}
		}
	}

	funcs := make(template.FuncMap)
	for k, v := range e.funcs {
		funcs[k] = v
	}

	if ctx, ok := data.(*TemplateContext); ok {
		// Connection-specific contextual functions
		funcs["connID"] = func() string {
			if ctx.ConnectionID != "" {
				return ctx.ConnectionID
			}
			if ctx.Conn != nil && ctx.Conn.URL != "" {
				return ctx.Conn.URL
			}
			return "🔌"
		}
		funcs["shortConnID"] = func() string {
			id := ctx.ConnectionID
			if id == "" && ctx.Conn != nil {
				id = ctx.Conn.URL
			}
			if id == "" {
				return "🔌"
			}
			if len(id) > 8 {
				return id[:8]
			}
			return id
		}
		funcs["sessionID"] = func() string { return ctx.SessionID }
		funcs["clientIP"] = func() string {
			if ctx.ClientIP != "" {
				return ctx.ClientIP
			}
			if ctx.Conn != nil {
				return ctx.Conn.ClientIP
			}
			return "❓"
		}
		funcs["remoteAddr"] = func() string {
			if ctx.RemoteAddr != "" {
				return ctx.RemoteAddr
			}
			if ctx.Conn != nil {
				return ctx.Conn.RemoteAddr
			}
			return "❓"
		}
		funcs["localAddr"] = func() string {
			if ctx.LocalAddr != "" {
				return ctx.LocalAddr
			}
			if ctx.Conn != nil {
				return ctx.Conn.LocalAddr
			}
			return "❓"
		}
		funcs["subprotocol"] = func() string {
			if ctx.Subprotocol != "" {
				return ctx.Subprotocol
			}
			if ctx.Conn != nil {
				return ctx.Conn.Subprotocol
			}
			return ""
		}
		funcs["connectedSince"] = func() time.Time {
			if !ctx.ConnectedSince.IsZero() {
				return ctx.ConnectedSince
			}
			if ctx.Conn != nil {
				return ctx.Conn.ConnectedAt
			}
			return time.Time{}
		}
		funcs["uptime"] = func() time.Duration {
			if ctx.Uptime > 0 {
				return ctx.Uptime
			}
			if ctx.Conn != nil {
				return ctx.Conn.Uptime
			}
			if !ctx.ConnectedSince.IsZero() {
				return time.Since(ctx.ConnectedSince)
			}
			return 0
		}
		funcs["messageCount"] = func() uint64 {
			if ctx.MessageCount > 0 {
				return ctx.MessageCount
			}
			if ctx.Conn != nil {
				return ctx.Conn.MessageCount
			}
			return 0
		}
	}

	tmpl, err := template.New(name).Funcs(funcs).Parse(text)
	if err != nil {
		return "", fmt.Errorf("parsing template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing template %s: %w", name, err)
	}

	return buf.String(), nil
}

// FuncNames returns a sorted list of registered template function names.
func (e *Engine) FuncNames() []string {
	names := make([]string, 0, len(e.funcs))
	for name := range e.funcs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetSessionID returns the session ID.
func (e *Engine) GetSessionID() string {
	return e.sessionID
}
