package repl

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
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
	mu   sync.RWMutex
	vars map[string]interface{}

	// completionData stores dynamic suggestions (bookmarks, aliases, JSON keys, etc.)
	completionData map[string][]string

	onInput func(ctx context.Context, text string) error
	
	// TemplateEngine is used for :format template
	TemplateEngine *template.Engine

	// Display handles message output and filtering
	Display *FormattingState

	done chan struct{}
}

// Config defines the configuration for the REPL.
type Config struct {
	Prompt          string
	HistoryFile     string
	HistoryLimit    int
	InterruptPrompt string
	EOFPrompt       string
}

// New creates a new REPL instance.
func New(mode Mode, cfg *Config) (*REPL, error) {
	if cfg == nil {
		cfg = &Config{}
	}
	if cfg.Prompt == "" {
		cfg.Prompt = "> "
	}
	if cfg.InterruptPrompt == "" {
		cfg.InterruptPrompt = "^C\n"
	}
	if cfg.EOFPrompt == "" {
		cfg.EOFPrompt = "exit"
	}

	if cfg.HistoryFile == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			cfg.HistoryFile = filepath.Join(home, ".xwebs_history")
		}
	} else if strings.HasPrefix(cfg.HistoryFile, "~") {
		home, _ := os.UserHomeDir()
		if home != "" {
			cfg.HistoryFile = filepath.Join(home, strings.TrimPrefix(cfg.HistoryFile, "~"))
		}
	}

	if cfg.HistoryLimit <= 0 {
		cfg.HistoryLimit = 1000 // Default limit
	}

	rlConfig := &readline.Config{
		Prompt:          cfg.Prompt,
		InterruptPrompt: cfg.InterruptPrompt,
		EOFPrompt:       cfg.EOFPrompt,
		HistoryFile:     cfg.HistoryFile,
		HistoryLimit:    cfg.HistoryLimit,
	}

	r := &REPL{
		mode:           mode,
		config:         cfg,
		commands:       make(map[string]Command),
		aliases:        make(map[string]string),
		vars:           make(map[string]interface{}),
		completionData: make(map[string][]string),
		done:           make(chan struct{}),
	}

	rlConfig.AutoComplete = r
	rl, err := readline.NewEx(rlConfig)
	if err != nil {
		return nil, fmt.Errorf("initializing readline: %w", err)
	}
	r.rl = rl
	r.Display = NewFormattingState()

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
	_, _ = r.rl.Write([]byte(fmt.Sprintf(format, args...)))
}

// Errorf prints a formatted error string to the REPL stderr.
func (r *REPL) Errorf(format string, args ...interface{}) {
	_, _ = r.rl.Write([]byte(fmt.Sprintf(format, args...)))
}

// Notify prints a notification (e.g. connection event) if quiet mode is not active.
func (r *REPL) Notify(format string, args ...interface{}) {
	if !r.Display.Quiet {
		r.Printf(format, args...)
	}
}

// PrintMessage formats and prints a WebSocket message if filtering allows.
func (r *REPL) PrintMessage(msg *ws.Message) {
	formatted, ok := r.Display.FormatMessage(msg, r.GetVars(), r.TemplateEngine)
	if ok {
		r.Printf("%s\n", formatted)
	}
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
	return r.DoContext(line, pos)
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

// SetCompletionData sets dynamic completion suggestions for a given category.
func (r *REPL) SetCompletionData(category string, suggestions []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Ensure unique and sorted
	unique := make(map[string]bool)
	var sorted []string
	for _, s := range suggestions {
		if !unique[s] {
			unique[s] = true
			sorted = append(sorted, s)
		}
	}
	sort.Strings(sorted)
	r.completionData[category] = sorted
}

// GetCompletionData returns dynamic completion suggestions for a given category.
func (r *REPL) GetCompletionData(category string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data := r.completionData[category]
	res := make([]string, len(data))
	copy(res, data)
	return res
}

// AddCompletionItem adds a single item to a completion category.
func (r *REPL) AddCompletionItem(category string, item string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	data := r.completionData[category]
	for _, x := range data {
		if x == item {
			return
		}
	}
	data = append(data, item)
	sort.Strings(data)
	r.completionData[category] = data
}
