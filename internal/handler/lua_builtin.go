package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/yuin/gopher-lua"
)

func init() {
	MustRegister(&LuaBuiltin{})
}

// LuaBuiltin runs an embedded Lua script for custom logic and transformations.
type LuaBuiltin struct{}

func (b *LuaBuiltin) Name() string        { return "lua" }
func (b *LuaBuiltin) Description() string { return "Run an embedded Lua script." }
func (b *LuaBuiltin) Scope() BuiltinScope { return Shared }

func (b *LuaBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Run an embedded Lua script for custom logic and transformations.",
		Fields: []BuiltinField{
			{Name: "script", Type: "string", Required: false, Description: "Inline Lua script content."},
			{Name: "file", Type: "string", Required: false, Description: "Path to a Lua file (supports templates)."},
			{Name: "timeout", Type: "duration", Default: "5s", Required: false, Description: "Execution timeout."},
		},
		TemplateVars: map[string]string{
			"message":       "Incoming message body (string)",
			"message_type":  "text | binary",
			"connection_id": "Unique ID of the sender",
			"vars":          "Table of session variables",
			"state":         "Persistent mutable table for the handler",
		},
		YAMLReplExample: "builtin: lua\nscript: |\n  if string.find(message, 'secret') then\n    return false -- drop message\n  end\n  return 'REDACTED: ' .. message",
		REPLAddExample:  ":handler add -m '*' --builtin lua --script 'return message:upper()'",
	}
}

func (b *LuaBuiltin) Validate(a Action) error {
	if a.Script == "" && a.File == "" {
		return fmt.Errorf("builtin lua requires either 'script:' or 'file:'")
	}
	if a.Script != "" && a.File != "" {
		return fmt.Errorf("builtin lua cannot have both 'script:' and 'file:'")
	}
	return nil
}

func (b *LuaBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Resolve script content
	var script string

	if a.Script != "" {
		script = a.Script
	} else {
		filePath, err := d.templateEngine.Execute("lua-file", a.File, tmplCtx)
		if err != nil {
			return fmt.Errorf("builtin lua: resolving file path template: %w", err)
		}
		filePath = strings.TrimSpace(filePath)
		resolvedPath := filePath
		if !filepath.IsAbs(filePath) && a.BaseDir != "" {
			resolvedPath = filepath.Join(a.BaseDir, filePath)
		}

		content, err := os.ReadFile(resolvedPath)
		if err != nil {
			return fmt.Errorf("builtin lua: reading file %s: %w", resolvedPath, err)
		}
		script = string(content)
	}

	// 2. Get/Initialize VM
	pool := d.registry.getLuaPool(a.HandlerName, a.MaxMemory)
	L := pool.Get()
	defer pool.Put(L)

	// Update context/timeout
	if a.Timeout != "" {
		dur, err := time.ParseDuration(a.Timeout)
		if err == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, dur)
			defer cancel()
		}
	} else {
		// Default 5s timeout if not specified
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
	}
	L.SetContext(ctx)

	// 3. Set up globals for this call
	b.setupGlobals(L, d, a, tmplCtx)

	// 4. Run script
	if err := L.DoString(script); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			tmplCtx.LuaError = "timeout"
			return fmt.Errorf("lua execution timeout")
		}
		tmplCtx.LuaError = err.Error()
		return fmt.Errorf("lua error: %w", err)
	}

	// 5. Handle return values
	ret := L.Get(-1)
	L.Pop(1)

	switch val := ret.(type) {
	case lua.LString:
		// Return string -> sent as response
		return d.conn.Write(&ws.Message{
			Type: ws.TextMessage,
			Data: []byte(string(val)),
			Metadata: ws.MessageMetadata{
				Direction: "sent",
				Timestamp: time.Now(),
			},
		})
	case *lua.LNilType:
		// Return nil -> fall through to respond:
		return nil
	case lua.LBool:
		if !bool(val) {
			// Return false explicitly -> drop message
			return ErrDrop
		}
		// Return true -> fall through
		return nil
	default:
		// Other types -> treated as nil (fall through)
		return nil
	}
}

func (b *LuaBuiltin) setupGlobals(L *lua.LState, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) {
	// message, message_len, message_type
	L.SetGlobal("message", lua.LString(tmplCtx.Message))
	L.SetGlobal("message_len", lua.LNumber(tmplCtx.MessageLen))
	L.SetGlobal("message_type", lua.LString(tmplCtx.MessageType))
	L.SetGlobal("connection_id", lua.LString(tmplCtx.ConnectionID))
	L.SetGlobal("remote_addr", lua.LString(tmplCtx.RemoteAddr))

	// vars (read-only copy)
	varsTable := L.NewTable()
	for k, v := range tmplCtx.Vars {
		varsTable.RawSetString(k, b.toLValue(L, v))
	}
	L.SetGlobal("vars", varsTable)

	// env (read-only copy)
	envTable := L.NewTable()
	for k, v := range tmplCtx.Env {
		envTable.RawSetString(k, lua.LString(v))
	}
	L.SetGlobal("env", envTable)

	// kv("key")
	L.SetGlobal("kv", L.NewFunction(func(L *lua.LState) int {
		if d.kvManager == nil {
			L.RaiseError("kv store not available (server mode only)")
			return 0
		}
		key := L.CheckString(1)
		val, ok := d.kvManager.GetKV(key)
		if !ok {
			L.Push(lua.LNil)
			return 1
		}
		L.Push(b.toLValue(L, val))
		return 1
	}))

	// json.encode / json.decode
	jsonMod := L.NewTable()
	L.SetField(jsonMod, "encode", L.NewFunction(func(L *lua.LState) int {
		v := L.CheckAny(1)
		data, err := json.Marshal(b.fromLValue(v))
		if err != nil {
			L.RaiseError("json.encode error: %v", err)
			return 0
		}
		L.Push(lua.LString(string(data)))
		return 1
	}))
	L.SetField(jsonMod, "decode", L.NewFunction(func(L *lua.LState) int {
		s := L.CheckString(1)
		var v interface{}
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			L.RaiseError("json.decode error: %v", err)
			return 0
		}
		L.Push(b.toLValue(L, v))
		return 1
	}))
	L.SetGlobal("json", jsonMod)

	// re.match(pattern, str), re.find(pattern, str)
	reMod := L.NewTable()
	L.SetField(reMod, "match", L.NewFunction(func(L *lua.LState) int {
		pattern := L.CheckString(1)
		str := L.CheckString(2)
		re, err := regexp.Compile(pattern)
		if err != nil {
			L.RaiseError("re.match pattern error: %v", err)
			return 0
		}
		L.Push(lua.LBool(re.MatchString(str)))
		return 1
	}))
	L.SetField(reMod, "find", L.NewFunction(func(L *lua.LState) int {
		pattern := L.CheckString(1)
		str := L.CheckString(2)
		re, err := regexp.Compile(pattern)
		if err != nil {
			L.RaiseError("re.find pattern error: %v", err)
			return 0
		}
		matches := re.FindStringSubmatch(str)
		if matches == nil {
			L.Push(lua.LNil)
			return 1
		}
		tbl := L.NewTable()
		for _, m := range matches {
			tbl.Append(lua.LString(m))
		}
		L.Push(tbl)
		return 1
	}))
	L.SetGlobal("re", reMod)

	// state (persistent mutable table)
	L.SetGlobal("state", d.registry.getLuaState(a.HandlerName, L))
}

func (b *LuaBuiltin) toLValue(L *lua.LState, v interface{}) lua.LValue {
	switch val := v.(type) {
	case string:
		return lua.LString(val)
	case float64:
		return lua.LNumber(val)
	case int:
		return lua.LNumber(val)
	case bool:
		return lua.LBool(val)
	case map[string]interface{}:
		tbl := L.NewTable()
		for k, v := range val {
			tbl.RawSetString(k, b.toLValue(L, v))
		}
		return tbl
	case []interface{}:
		tbl := L.NewTable()
		for _, v := range val {
			tbl.Append(b.toLValue(L, v))
		}
		return tbl
	case nil:
		return lua.LNil
	default:
		return lua.LString(fmt.Sprintf("%v", val))
	}
}

func (b *LuaBuiltin) fromLValue(v lua.LValue) interface{} {
	switch val := v.(type) {
	case lua.LString:
		return string(val)
	case lua.LNumber:
		return float64(val)
	case lua.LBool:
		return bool(val)
	case *lua.LTable:
		// Detect if it's an array or map
		isArr := true
		maxIdx := 0
		val.ForEach(func(k, v lua.LValue) {
			if n, ok := k.(lua.LNumber); ok && float64(n) == float64(int(n)) && int(n) > 0 {
				if int(n) > maxIdx {
					maxIdx = int(n)
				}
			} else {
				isArr = false
			}
		})
		if isArr && maxIdx > 0 {
			arr := make([]interface{}, maxIdx)
			for i := 1; i <= maxIdx; i++ {
				arr[i-1] = b.fromLValue(val.RawGet(lua.LNumber(i)))
			}
			return arr
		}
		m := make(map[string]interface{})
		val.ForEach(func(k, v lua.LValue) {
			m[k.String()] = b.fromLValue(v)
		})
		return m
	case *lua.LNilType:
		return nil
	default:
		return v.String()
	}
}

// LuaPool manages a pool of Lua VMs.
type LuaPool struct {
	pool      sync.Pool
	maxMemory int
}

func NewLuaPool(maxMemory int) *LuaPool {
	return &LuaPool{
		maxMemory: maxMemory,
		pool: sync.Pool{
			New: func() interface{} {
				L := lua.NewState(lua.Options{
					SkipOpenLibs: true,
				})
				// Load allowed standard libraries
				for _, lib := range []struct {
					name string
					fn   lua.LGFunction
				}{
					{"base", lua.OpenBase},
					{"table", lua.OpenTable},
					{"string", lua.OpenString},
					{"math", lua.OpenMath},
					{"debug", lua.OpenDebug},
					{"coroutine", lua.OpenCoroutine},
				} {
					if err := L.CallByParam(lua.P{
						Fn:      L.NewFunction(lib.fn),
						NRet:    0,
						Protect: true,
					}, lua.LString(lib.name)); err != nil {
						panic(err)
					}
				}

				// Sandboxing: remove dangerous functions from base library
				L.SetGlobal("loadfile", lua.LNil)
				L.SetGlobal("dofile", lua.LNil)
				L.SetGlobal("load", lua.LNil)
				L.SetGlobal("module", lua.LNil)
				L.SetGlobal("require", lua.LNil)
				L.SetGlobal("package", lua.LNil)

				// gopher-lua doesn't support memory limits directly in this version
				// or it's done via a custom allocator. Skipping for now.

				return L
			},
		},
	}
}

func (p *LuaPool) Get() *lua.LState {
	return p.pool.Get().(*lua.LState)
}

func (p *LuaPool) Put(L *lua.LState) {
	p.pool.Put(L)
}

// Helper methods for Registry to manage Lua resources
func (r *Registry) getLuaPool(handlerName string, maxMemory int) *LuaPool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if pool, ok := r.luaPools[handlerName]; ok {
		return pool.(*LuaPool)
	}

	pool := NewLuaPool(maxMemory)
	r.luaPools[handlerName] = pool
	return pool
}

func (r *Registry) getLuaState(handlerName string, L *lua.LState) *lua.LTable {
	r.mu.Lock()
	defer r.mu.Unlock()

	if state, ok := r.luaStates[handlerName]; ok {
		return state.(*lua.LTable)
	}

	state := L.NewTable()
	r.luaStates[handlerName] = state
	return state
}
