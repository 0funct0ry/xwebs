package repl

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
)

// Command defines the interface for a REPL command.
type Command interface {
	Name() string
	Help() string
	Execute(ctx context.Context, r *REPL, args []string) error
}

// BuiltinCommand is a simple implementation of the Command interface.
type BuiltinCommand struct {
	name    string
	help    string
	handler func(ctx context.Context, r *REPL, args []string) error
}

func (c *BuiltinCommand) Name() string {
	return c.name
}

func (c *BuiltinCommand) Help() string {
	return c.help
}

func (c *BuiltinCommand) Execute(ctx context.Context, r *REPL, args []string) error {
	return c.handler(ctx, r, args)
}

// Subsystem is a group of commands that can be added to the REPL.
type Subsystem struct {
	Commands []Command
}

// RegisterCommand adds a command to the REPL.
func (r *REPL) RegisterCommand(cmd Command) {
	r.commands[cmd.Name()] = cmd
}

// RegisterAlias adds an alias for a command.
func (r *REPL) RegisterAlias(name, cmdName string) {
	r.aliases[name] = cmdName
}

// RegisterCommonCommands adds the standard REPL commands.
func (r *REPL) RegisterCommonCommands() {
	r.RegisterCommand(&BuiltinCommand{
		name: "help",
		help: "List all commands and their descriptions",
		handler: func(ctx context.Context, _ *REPL, args []string) error {
			cmds := make([]string, 0, len(r.commands))
			for name := range r.commands {
				cmds = append(cmds, name)
			}
			sort.Strings(cmds)

			r.Printf("\nAvailable commands:\n")
			for _, name := range cmds {
				r.Printf("  :%-15s %s\n", name, r.commands[name].Help())
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "exit",
		help: "Disconnect and exit",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			_ = r.Close()
			return ErrExit
		},
	})
	r.RegisterAlias("quit", "exit")

	r.RegisterCommand(&BuiltinCommand{
		name: "clear",
		help: "Clear the screen",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			r.Printf("\033[H\033[2J") // Standard ANSI clear
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "set",
		help: "Set a session variable: :set <key> <value>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("usage: :set <key> <value>")
			}
			r.SetVar(args[0], strings.Join(args[1:], " "))
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "get",
		help: "Get a session variable: :get <key>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: :get <key>")
			}
			val := r.GetVar(args[0])
			if val == nil {
				r.Printf("%s is not set\n", args[0])
			} else {
				r.Printf("%v\n", val)
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "vars",
		help: "List all session variables",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			vars := r.GetVars()
			if len(vars) == 0 {
				r.Printf("No session variables defined.\n")
				return nil
			}
			keys := make([]string, 0, len(vars))
			for k := range vars {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			r.Printf("\nSession Variables:\n")
			for _, k := range keys {
				r.Printf("  %-15s = %v\n", k, vars[k])
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "env",
		help: "List all environment variables",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			envs := os.Environ()
			sort.Strings(envs)
			r.Printf("\nEnvironment Variables:\n")
			for _, e := range envs {
				r.Printf("  %s\n", e)
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "history",
		help: "Display command history: :history [n]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if r.config.HistoryFile == "" {
				return fmt.Errorf("history is not enabled (no history file configured)")
			}

			data, err := os.ReadFile(r.config.HistoryFile)
			if err != nil {
				if os.IsNotExist(err) {
					r.Printf("No history found.\n")
					return nil
				}
				return fmt.Errorf("reading history file: %w", err)
			}

			lines := strings.Split(string(data), "\n")
			// Filter empty lines
			var history []string
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					history = append(history, line)
				}
			}

			n := 20 // Default to last 20
			if len(args) > 0 {
				if val, err := strconv.Atoi(args[0]); err == nil {
					n = val
				}
			}

			if n > len(history) {
				n = len(history)
			}

			start := len(history) - n
			r.Printf("\nCommand History (last %d):\n", n)
			for i := start; i < len(history); i++ {
				r.Printf("  %4d  %s\n", i+1, history[i])
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "format",
		help: "Set message display format: :format [json|raw|hex|template <tmpl>]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				r.Printf("format: %s\n", r.Display.Format)
				return nil
			}
			format := DisplayFormat(args[0])
			switch format {
			case FormatJSON, FormatRaw, FormatHex, FormatTemplate:
				r.Display.Format = format
				if format == FormatTemplate && len(args) > 1 {
					r.Display.Template = strings.Join(args[1:], " ")
				}
				r.Printf("format set to %s\n", format)
			default:
				return fmt.Errorf("invalid format: %s (choose json, raw, hex, template)", format)
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "filter",
		help: "Set a display filter (JQ or Regex): :filter [.jq-expr|/regex/|off]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				if r.Display.Filter == "" {
					r.Printf("filter: off\n")
				} else {
					r.Printf("filter: %s\n", r.Display.Filter)
				}
				return nil
			}
			expr := strings.Join(args, " ")
			if err := r.Display.SetFilter(expr); err != nil {
				return err
			}
			if r.Display.Filter == "" {
				r.Printf("filter cleared\n")
			} else {
				r.Printf("filter set: %s\n", r.Display.Filter)
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "quiet",
		help: "Toggle non-message output suppression",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			r.Display.Quiet = !r.Display.Quiet
			r.Printf("quiet mode: %v\n", r.Display.Quiet)
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "verbose",
		help: "Toggle frame-level metadata display",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			r.Display.Verbose = !r.Display.Verbose
			r.Printf("verbose mode: %v\n", r.Display.Verbose)
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "timestamps",
		help: "Control ISO 8601 message timestamps: :timestamps [on|off]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				status := "off"
				if r.Display.Timestamps {
					status = "on"
				}
				r.Printf("timestamps: %s\n", status)
				return nil
			}
			switch args[0] {
			case "on":
				r.Display.Timestamps = true
			case "off":
				r.Display.Timestamps = false
			default:
				return fmt.Errorf("usage: :timestamps [on|off]")
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "color",
		help: "Control output coloring: :color [on|off|auto]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				r.Printf("color: %s\n", r.Display.Color)
				return nil
			}
			switch args[0] {
			case "on", "off", "auto":
				r.Display.Color = args[0]
				r.Printf("color set to %s\n", args[0])
			default:
				return fmt.Errorf("usage: :color [on|off|auto]")
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "source",
		help: "Execute commands from a file: :source <file>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: :source <file>")
			}
			path := args[0]
			if r.execDepth >= 10 {
				return fmt.Errorf("maximum source recursion depth (10) exceeded")
			}
			r.execDepth++
			defer func() { r.execDepth-- }()

			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("opening script %s: %w", path, err)
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			lineNum := 0
			for scanner.Scan() {
				lineNum++
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}

				// Template expanded before execution
				tmplCtx := template.NewContext()
				tmplCtx.Session = r.GetVars()
				tmplCtx.Env = make(map[string]string)
				for _, e := range os.Environ() {
					parts := strings.SplitN(e, "=", 2)
					if len(parts) == 2 {
						tmplCtx.Env[parts[0]] = parts[1]
					}
				}
				
				// Add scripting context
				if lastMsg := r.GetLastMessage(); lastMsg != nil {
					tmplCtx.Last = string(lastMsg.Data)
				}
				tmplCtx.LastLatencyMs = r.GetLastLatency().Milliseconds()

				expanded, err := r.TemplateEngine.Execute(fmt.Sprintf("%s:%d", path, lineNum), line, tmplCtx)
				if err != nil {
					return fmt.Errorf("%s:%d: template error: %w", path, lineNum, err)
				}

				if err := r.ExecuteCommand(ctx, expanded); err != nil {
					if err == ErrExit {
						return ErrExit
					}
					return fmt.Errorf("%s:%d: %w", path, lineNum, err)
				}
			}
			return scanner.Err()
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "alias",
		help: "Define or list command aliases: :alias [name] [cmd]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				if len(r.scriptAliases) == 0 {
					r.Printf("No aliases defined.\n")
					return nil
				}
				keys := make([]string, 0, len(r.scriptAliases))
				for k := range r.scriptAliases {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				r.Printf("\nAliases:\n")
				for _, k := range keys {
					r.Printf("  :%-15s %s\n", k, r.scriptAliases[k])
				}
				return nil
			}
			if len(args) == 1 {
				val, ok := r.scriptAliases[args[0]]
				if !ok {
					return fmt.Errorf("alias not found: %s", args[0])
				}
				r.Printf(":%s = %s\n", args[0], val)
				return nil
			}
			r.scriptAliases[args[0]] = strings.Join(args[1:], " ")
			r.Printf("alias registered: :%s\n", args[0])
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "unalias",
		help: "Remove a command alias: :unalias <name>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: :unalias <name>")
			}
			if _, ok := r.scriptAliases[args[0]]; !ok {
				return fmt.Errorf("alias not found: %s", args[0])
			}
			delete(r.scriptAliases, args[0])
			r.Printf("alias removed: :%s\n", args[0])
			return nil
		},
	})

	waitHandler := func(ctx context.Context, r *REPL, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("usage: :wait <duration>")
		}
		d, err := time.ParseDuration(args[0])
		if err != nil {
			return fmt.Errorf("invalid duration: %w", err)
		}
		select {
		case <-time.After(d):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-r.done:
			return nil
		}
	}
	r.RegisterCommand(&BuiltinCommand{
		name:    "wait",
		help:    "Pause execution for a duration: :wait <duration> (e.g. 500ms, 1s)",
		handler: waitHandler,
	})
	r.RegisterCommand(&BuiltinCommand{
		name:    "sleep",
		help:    "Alias for :wait",
		handler: waitHandler,
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "assert",
		help: "Assert that a template expression is true: :assert <expr> [msg]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: :assert <expression> [message]")
			}
			expr := args[0]
			msg := ""
			if len(args) > 1 {
				msg = strings.Join(args[1:], " ")
			}

			tmplCtx := template.NewContext()
			tmplCtx.Session = r.GetVars()
			// Add scripting context
			if lastMsg := r.GetLastMessage(); lastMsg != nil {
				tmplCtx.Last = string(lastMsg.Data)
			}
			tmplCtx.LastLatencyMs = r.GetLastLatency().Milliseconds()

			res, err := r.TemplateEngine.Execute("assert", expr, tmplCtx)
			if err != nil {
				return fmt.Errorf("assertion template error: %w", err)
			}

			// Clean result for check
			res = strings.TrimSpace(strings.ToLower(res))
			
			// Fails if empty, "false", or "0"
			if res == "" || res == "false" || res == "0" {
				if msg == "" {
					msg = expr
				}
				r.Display.Color = "on" // Temporarily force color for error if auto? No, use colorizedText logic.
				failMsg := r.Display.colorizedText(fmt.Sprintf("ASSERT FAILED: %s", msg), "red")
				r.Printf("%s\n", failMsg)
				return fmt.Errorf("assertion failed: %s", msg)
			}

			if r.Display.Verbose {
				successMsg := r.Display.colorizedText(fmt.Sprintf("ASSERT OK: %s", msg), "green")
				r.Printf("%s\n", successMsg)
			}
			return nil
		},
	})
	r.RegisterCommand(&BuiltinCommand{
		name: "handlers",
		help: "List all loaded handlers in priority order",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if r.Handlers == nil {
				r.Printf("No handlers loaded.\n")
				return nil
			}
			handlers := r.Handlers.Handlers()
			if len(handlers) == 0 {
				r.Printf("No handlers registered.\n")
				return nil
			}

			r.Printf("\nLoaded Handlers (priority order):\n")
			for i, h := range handlers {
				priorityStr := fmt.Sprintf("p=%d", h.Priority)
				matcherStr := fmt.Sprintf("%s:%s", h.Match.Type, h.Match.Pattern)
				if h.Match.Type == "" {
					matcherStr = "text:" + h.Match.Pattern
				}

				r.Printf("  %2d. %-20s [%s] match=%s\n", i+1, h.Name, priorityStr, matcherStr)
				for _, a := range h.Actions {
					desc := a.Command
					if desc == "" {
						desc = a.Message
					}
					r.Printf("      - %-8s %s\n", a.Type, desc)
				}
				if len(h.OnConnect) > 0 {
					r.Printf("      (on_connect: %d actions)\n", len(h.OnConnect))
				}
				if len(h.OnDisconnect) > 0 {
					r.Printf("      (on_disconnect: %d actions)\n", len(h.OnDisconnect))
				}
				if len(h.OnError) > 0 {
					r.Printf("      (on_error: %d actions)\n", len(h.OnError))
				}
			}
			return nil
		},
	})
}
