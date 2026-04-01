package repl

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/chzyer/readline"
)

// Mode represents the operation mode of the REPL (Client or Server).
type Mode int

const (
	// ClientMode is for connecting to remote WebSocket servers.
	ClientMode Mode = iota
	// ServerMode is for managing a local WebSocket server.
	ServerMode
)

// REPL is the core engine for interactive CLI handling.
type REPL struct {
	mode   Mode
	rl     *readline.Instance
	config *Config
	
	commands map[string]Command
	aliases  map[string]string
	
	// mu protects session variables
	mu    sync.RWMutex
	vars  map[string]interface{}
	
	onInput func(ctx context.Context, text string) error
	
	done chan struct{}
}

// Config defines the configuration for the REPL.
type Config struct {
	Prompt          string
	HistoryFile     string
	InterruptPrompt string
	EOFPrompt       string
}

// New creates a new REPL instance.
func New(mode Mode, cfg *Config) (*REPL, error) {
	if cfg == nil {
		cfg = &Config{
			Prompt:          "> ",
			InterruptPrompt: "^C\n",
			EOFPrompt:       "exit",
		}
	}

	rlConfig := &readline.Config{
		Prompt:          cfg.Prompt,
		InterruptPrompt: cfg.InterruptPrompt,
		EOFPrompt:       cfg.EOFPrompt,
		HistoryFile:     cfg.HistoryFile,
	}

	r := &REPL{
		mode:     mode,
		config:   cfg,
		commands: make(map[string]Command),
		aliases:  make(map[string]string),
		vars:     make(map[string]interface{}),
		done:     make(chan struct{}),
	}

	rlConfig.AutoComplete = r
	rl, err := readline.NewEx(rlConfig)
	if err != nil {
		return nil, fmt.Errorf("initializing readline: %w", err)
	}
	r.rl = rl

	return r, nil
}

// SetOnInput sets the fallback handler for input that is not a command.
func (r *REPL) SetOnInput(f func(ctx context.Context, text string) error) {
	r.onInput = f
}

// Close closes the REPL and its readline instance.
func (r *REPL) Close() error {
	select {
	case <-r.done:
		return nil
	default:
		close(r.done)
	}
	return r.rl.Close()
}

// Printf prints a formatted string to the REPL output, ensuring it doesn't break the current prompt.
func (r *REPL) Printf(format string, args ...interface{}) {
	fmt.Fprintf(r.rl.Stdout(), format, args...)
}

// Errorf prints a formatted error string to the REPL stderr.
func (r *REPL) Errorf(format string, args ...interface{}) {
	fmt.Fprintf(r.rl.Stderr(), format, args...)
}

// Run starts the REPL input loop.
func (r *REPL) Run(ctx context.Context) error {
	defer r.Close()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.done:
			return nil
		default:
			line, err := r.rl.Readline()
			if err != nil {
				if err == readline.ErrInterrupt {
					if len(line) == 0 {
						return nil
					}
					continue
				}
				if err == io.EOF {
					return nil
				}
				return err
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if strings.HasPrefix(line, ":") {
				if err := r.executeCommand(ctx, line); err != nil {
					r.Errorf("Error: %v\n", err)
				}
			} else if r.onInput != nil {
				if err := r.onInput(ctx, line); err != nil {
					r.Errorf("Error: %v\n", err)
				}
			}
		}
	}
}

// executeCommand identifies and runs a REPL command.
func (r *REPL) executeCommand(ctx context.Context, line string) error {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return nil
	}

	fullCmd := parts[0]
	cmdName := strings.TrimPrefix(fullCmd, ":")
	args := parts[1:]

	// Check aliases
	if alias, ok := r.aliases[cmdName]; ok {
		cmdName = alias
	}

	cmd, ok := r.commands[cmdName]
	if !ok {
		return fmt.Errorf("unknown command: %s", cmdName)
	}

	return cmd.Execute(ctx, r, args)
}

// Do satisfies the readline.AutoCompleter interface.
func (r *REPL) Do(line []rune, pos int) (newLine [][]rune, length int) {
	if len(line) == 0 || line[0] != ':' {
		return nil, 0
	}

	// Only complete the first word (the command)
	currentLine := string(line[:pos])
	parts := strings.Fields(currentLine)
	if len(parts) > 1 && !strings.HasSuffix(currentLine, " ") {
		return nil, 0
	}

	prefix := strings.TrimPrefix(parts[0], ":")
	var suggestions [][]rune

	// Collect unique command names and aliases
	seen := make(map[string]bool)
	for name := range r.commands {
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			if !seen[name] {
				suggestions = append(suggestions, []rune(name[len(prefix):]))
				seen[name] = true
			}
		}
	}
	for alias := range r.aliases {
		if strings.HasPrefix(strings.ToLower(alias), strings.ToLower(prefix)) {
			if !seen[alias] {
				suggestions = append(suggestions, []rune(alias[len(prefix):]))
				seen[alias] = true
			}
		}
	}

	return suggestions, len(prefix)
}

// GetVar returns a session variable.
func (r *REPL) GetVar(name string) interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.vars[name]
}

// SetVar sets a session variable.
func (r *REPL) SetVar(name string, value interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.vars[name] = value
}

// GetVars returns all session variables.
func (r *REPL) GetVars() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Return a copy
	res := make(map[string]interface{}, len(r.vars))
	for k, v := range r.vars {
		res[k] = v
	}
	return res
}
