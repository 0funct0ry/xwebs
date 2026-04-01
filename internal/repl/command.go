package repl

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
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
			return r.Close()
		},
	})
	r.RegisterAlias("quit", "exit")

	r.RegisterCommand(&BuiltinCommand{
		name: "clear",
		help: "Clear the screen",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			fmt.Printf("\033[H\033[2J") // Standard ANSI clear
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
}
