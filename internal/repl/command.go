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

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/matoous/go-nanoid/v2"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
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
		name: "pwd",
		help: "Display the current working directory: :pwd [varname]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			if len(args) > 0 {
				r.SetVar(args[0], cwd)
				r.Printf("Saved to variable %s: ", args[0])
			}

			colorized := r.Display.colorizedText(cwd, "cyan")
			r.Printf("%s\n", colorized)
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
		name: "prompt",
		help: "Customize the REPL prompt: :prompt (set <tmpl>|reset|default)",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				r.Printf("Usage:\n")
				r.Printf("  :prompt set <template>\n")
				r.Printf("  :prompt reset | default\n")
				return nil
			}

			subcmd := args[0]
			switch subcmd {
			case "set":
				if len(args) < 2 {
					return fmt.Errorf("usage: :prompt set <template>")
				}
				r.promptTemplate = strings.Join(args[1:], " ")
				r.renderPrompt()
				r.Printf("Prompt template updated.\n")
			case "reset", "default":
				r.promptTemplate = ""
				r.renderPrompt()
				r.Printf("Prompt reset to default.\n")
			default:
				return fmt.Errorf("unknown prompt subcommand: %s", subcmd)
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
				matcherStr := ""
				if h.Match.Regex != "" {
					matcherStr = "regex:" + h.Match.Regex
				} else if h.Match.JQ != "" {
					matcherStr = "jq:" + h.Match.JQ
				} else if h.Match.Type != "" {
					matcherStr = fmt.Sprintf("%s:%s", h.Match.Type, h.Match.Pattern)
				} else {
					matcherStr = "text:" + h.Match.Pattern
				}

				extraInfo := ""
				if h.Exclusive {
					extraInfo += " [exclusive]"
				}
				if h.Concurrent != nil && !*h.Concurrent {
					extraInfo += " [sequential]"
				}
				if h.RateLimit != "" {
					extraInfo += fmt.Sprintf(" [limit:%s]", h.RateLimit)
				}
				if h.Debounce != "" {
					extraInfo += fmt.Sprintf(" [debounce:%s]", h.Debounce)
				}

				r.Printf("  %2d. %-20s [%s]%s match=%s\n", i+1, h.Name, priorityStr, extraInfo, matcherStr)
				for _, a := range h.Actions {
					desc := a.Command
					if desc == "" {
						desc = a.Message
					}
					r.Printf("      - %-8s %s\n", a.Type, desc)
				}
				if h.Run != "" {
					r.Printf("      - %-8s %s\n", "run", h.Run)
				}
				if h.Respond != "" {
					r.Printf("      - %-8s %s\n", "respond", h.Respond)
				}
				if h.Builtin != "" {
					r.Printf("      - %-8s %s\n", "builtin", h.Builtin)
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

	r.RegisterAlias("h", "handlers")

	r.RegisterCommand(&BuiltinCommand{
		name: "handler",
		help: "Manage message handlers: :handler (add|delete|edit|save) <args>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				r.Printf("Usage:\n")
				r.Printf("  :handler add <flags>\n")
				r.Printf("  :handler delete <name>\n")
				r.Printf("  :handler edit [name]\n")
				r.Printf("  :handler save <filename> [--force]\n")
				r.Printf("\nFlags for 'add':\n")
				r.Printf("  --name <name>         (optional) Unique handler name\n")
				r.Printf("  --match <pattern>     (required) Match pattern\n")
				r.Printf("  --match-type <type>   Match type (text, glob, regex, jq, etc.)\n")
				r.Printf("  --priority <n>        Numeric priority (higher runs first)\n")
				r.Printf("  --run <cmd>           Shell command to run on match\n")
				r.Printf("  --respond <tmpl>      Response template to send after run\n")
				r.Printf("  --exclusive           Stop further matching if this handler matches\n")
				r.Printf("  --sequential          Run handler actions sequentially (disable concurrency)\n")
				r.Printf("  --rate-limit <limit>  Rate limit (e.g. '10/s')\n")
				r.Printf("  --debounce <duration> Debounce time (e.g. '500ms')\n")
				return nil
			}

			subcmd := args[0]
			if subcmd == "delete" {
				if len(args) < 2 {
					return fmt.Errorf("usage: :handler delete <name>")
				}
				if r.Handlers == nil {
					return fmt.Errorf("no handlers registered")
				}
				name := args[1]
				if err := r.Handlers.Delete(name); err != nil {
					return err
				}
				r.Printf("Handler %q deleted successfully.\n", name)
				return nil
			}

			if subcmd == "edit" {
				if r.Handlers == nil {
					return fmt.Errorf("no handlers registered")
				}

				if len(args) > 1 {
					// Edit specific handler
					name := args[1]
					h, ok := r.Handlers.GetHandler(name)
					if !ok {
						return fmt.Errorf("handler %q not found", name)
					}

					data, err := yaml.Marshal(h)
					if err != nil {
						return fmt.Errorf("marshaling handler: %w", err)
					}

					edited, err := r.openInEditor(ctx, string(data))
					if err != nil {
						return err
					}

					// Check if any changes were made
					if strings.TrimSpace(edited) == "" || strings.TrimSpace(edited) == strings.TrimSpace(string(data)) {
						r.Printf("No changes made.\n")
						return nil
					}

					var updatedh handler.Handler
					if err := yaml.Unmarshal([]byte(edited), &updatedh); err != nil {
						return fmt.Errorf("unmarshaling edited handler: %w", err)
					}

					// Validate (wrap in Config for full validation)
					cfg := handler.Config{Handlers: []handler.Handler{updatedh}}
					if err := cfg.Validate(); err != nil {
						return fmt.Errorf("validation failed: %w", err)
					}

					if err := r.Handlers.UpdateHandler(updatedh); err != nil {
						return err
					}
					r.Printf("Handler %q updated successfully.\n", updatedh.Name)
					return nil
				} else {
					// Edit full current configuration
					cfg := handler.Config{
						Variables: r.GetVars(),
						Handlers:  r.Handlers.Handlers(),
					}

					data, err := yaml.Marshal(cfg)
					if err != nil {
						return fmt.Errorf("marshaling configuration: %w", err)
					}

					edited, err := r.openInEditor(ctx, string(data))
					if err != nil {
						return err
					}

					// Check if any changes were made
					if strings.TrimSpace(edited) == "" || strings.TrimSpace(edited) == strings.TrimSpace(string(data)) {
						r.Printf("No changes made.\n")
						return nil
					}

					var newCfg handler.Config
					if err := yaml.Unmarshal([]byte(edited), &newCfg); err != nil {
						return fmt.Errorf("unmarshaling edited configuration: %w", err)
					}

					if err := newCfg.Validate(); err != nil {
						return fmt.Errorf("validation failed: %w", err)
					}

					// Apply changes
					r.Handlers.ReplaceHandlers(newCfg.Handlers)
					if newCfg.Variables != nil {
						r.ReplaceVars(newCfg.Variables)
					}
					
					r.Printf("Handler configuration updated successfully.\n")
					return nil
				}
			}

			if subcmd == "save" {
				if len(args) < 2 {
					return fmt.Errorf("usage: :handler save <filename> [--force|-f]")
				}
				filename := args[1]

				// Parse flags
				fs := pflag.NewFlagSet("handler save", pflag.ContinueOnError)
				fs.SetOutput(nil)
				var force bool
				fs.BoolVarP(&force, "force", "f", false, "Overwrite existing file")
				if err := fs.Parse(args[2:]); err != nil {
					return fmt.Errorf("parsing flags: %w", err)
				}

				// Check if file exists
				if _, err := os.Stat(filename); err == nil && !force {
					return fmt.Errorf("file %q already exists (use --force or -f to overwrite)", filename)
				}

				// Prepare config
				cfg := handler.Config{
					Variables: r.GetVars(),
					Handlers:  r.Handlers.Handlers(),
				}

				// Marshal
				data, err := yaml.Marshal(cfg)
				if err != nil {
					return fmt.Errorf("marshaling handlers: %w", err)
				}

				// Write
				if err := os.WriteFile(filename, data, 0644); err != nil {
					return fmt.Errorf("writing to file: %w", err)
				}

				r.Printf("Saved %d handlers and session variables to %q.\n", len(cfg.Handlers), filename)
				return nil
			}

			if subcmd != "add" {
				return fmt.Errorf("unknown handler subcommand: %s (use 'add', 'delete' or 'edit')", subcmd)
			}

			// Safety check: ensure Handlers registry is initialized
			if r.Handlers == nil {
				r.Handlers = handler.NewRegistry()
			}

			// Parse flags robustly using pflag
			fs := pflag.NewFlagSet("handler add", pflag.ContinueOnError)
			fs.SetOutput(nil) // Suppress automatic usage printing on error

			var name, match, matchType, run, respond, rateLimit, debounce string
			var priority int
			var exclusive, sequential bool

			fs.StringVar(&name, "name", "", "Name of the handler")
			fs.StringVar(&match, "match", "", "Match pattern")
			fs.StringVar(&matchType, "match-type", "", "Match type")
			fs.IntVar(&priority, "priority", 0, "Priority")
			fs.StringVar(&run, "run", "", "Shell command")
			fs.StringVar(&respond, "respond", "", "Response template")
			fs.BoolVar(&exclusive, "exclusive", false, "Short-circuit match")
			fs.BoolVar(&sequential, "sequential", false, "Run actions sequentially")
			fs.StringVar(&rateLimit, "rate-limit", "", "Rate limit")
			fs.StringVar(&debounce, "debounce", "", "Debounce duration")

			if err := fs.Parse(args[1:]); err != nil {
				return fmt.Errorf("parsing flags: %w", err)
			}

			// Validation
			if match == "" {
				return fmt.Errorf("--match is required")
			}

			// Auto-generate name if missing
			if name == "" {
				id, err := gonanoid.Generate("abcdefghijklmnopqrstuvwxyz0123456789", 6)
				if err != nil {
					// Fallback to timestamp if nanoid fails
					name = fmt.Sprintf("h-%d", time.Now().Unix()%10000)
				} else {
					name = "h-" + id
				}
			}

			// Construct handler
			h := handler.Handler{
				Name:      name,
				Priority:  priority,
				Exclusive: exclusive,
				Run:       run,
				Respond:   respond,
				Match: handler.Matcher{
					Pattern: match,
					Type:    matchType,
				},
				RateLimit: rateLimit,
				Debounce:  debounce,
			}

			if sequential {
				f := false
				h.Concurrent = &f
			}

			// Add to registry
			if err := r.Handlers.Add(h); err != nil {
				return err
			}

			r.Printf("Handler %q added successfully.\n", name)
			return nil
		},
	})
}
