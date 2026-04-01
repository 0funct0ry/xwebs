package repl

import (
	"context"
	"fmt"
	"sort"
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
			for _, k := range keys {
				r.Printf("  %-15s = %v\n", k, vars[k])
			}
			return nil
		},
	})
}
