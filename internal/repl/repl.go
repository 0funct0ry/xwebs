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
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/mock"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/chzyer/readline"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
)

// ErrExit is returned when a command requests to exit the REPL.
var ErrExit = fmt.Errorf("exit requested")

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
	mode          Mode
	rl            *readline.Instance
	config        *Config
	IsInteractive bool // Flag to indicate if real-time interactive REPL is active
	commands      map[string]Command
	aliases       map[string]string
	scriptAliases map[string]string

	// execDepth tracks recursion depth for :source and aliases
	execDepth int

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

	// lastMsg tracks the most recently received message
	lastMsg *ws.Message
	// lastLatency tracks the RTT of the last message pair
	lastLatency time.Duration
	// lastSendTime tracks when the last command was sent
	lastSendTime time.Time

	// Session management
	Logger   *Logger
	Recorder *Recorder
	Mocker   *mock.Mocker
	Handlers *handler.Registry

	// lastInput stores the most recently typed line to suppress redundant markers
	lastInput   string
	lastInputMu sync.Mutex

	// multiLineBuffer stores partial lines for \ continuation
	multiLineBuffer []string

	// Command execution context management
	cmdMu     sync.Mutex
	cmdCancel context.CancelFunc
	cmdActive bool

	isStdoutTTY bool
	done        chan struct{}

	// Prompt management
	promptTemplate    string
	originalPrompt    string
	promptErrReported bool

	// prevDir stores the previous working directory for :cd -
	prevDir string

	clientCtx ClientContext

	// configPaths tracks loaded configuration files for :edit reloading
	configPaths []string

	// Heredoc support
	heredocDelimiter string
	heredocBuffer    []string
}

// Config defines the configuration for the REPL.
type Config struct {
	Prompt             string
	HistoryFile        string
	HistoryLimit       int
	InterruptPrompt    string
	EOFPrompt          string
	ContinuationPrompt string
	PromptTemplate     string // New: Go template for the prompt
	Stdin              io.ReadCloser
	Stdout             io.WriteCloser
	Terminal           bool // Whether to initialize the terminal (readline)
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
	if cfg.ContinuationPrompt == "" {
		cfg.ContinuationPrompt = "... "
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
		Stdin:           cfg.Stdin,
		Stdout:          cfg.Stdout,
	}

	// Avoid sending terminal control sequences if Stdout is redirected to a file.
	if cfg.Stdout != nil {
		if f, ok := cfg.Stdout.(*os.File); ok {
			if stat, err := f.Stat(); err == nil {
				if (stat.Mode() & os.ModeCharDevice) == 0 {
					// Fallback readline internally to os.Stderr if output is a file,
					// so r.Printf can write to the file cleanly via fmt.Fprintf.
					rlConfig.Stdout = os.Stderr
				}
			}
		}
	}

	r := &REPL{
		mode:           mode,
		config:         cfg,
		commands:       make(map[string]Command),
		aliases:        make(map[string]string),
		scriptAliases:  make(map[string]string),
		vars:           make(map[string]interface{}),
		completionData: make(map[string][]string),
		done:           make(chan struct{}),
		Handlers:       handler.NewRegistry(),
		originalPrompt: cfg.Prompt,
		promptTemplate: cfg.PromptTemplate,
	}

	// Detect if Stdout is a TTY
	r.isStdoutTTY = true
	if cfg.Stdout != nil {
		if f, ok := cfg.Stdout.(*os.File); ok {
			if stat, err := f.Stat(); err == nil {
				if (stat.Mode() & os.ModeCharDevice) == 0 {
					r.isStdoutTTY = false
				}
			}
		}
	} else {
		if stat, err := os.Stdout.Stat(); err == nil {
			if (stat.Mode() & os.ModeCharDevice) == 0 {
				r.isStdoutTTY = false
			}
		}
	}

	r.Display = NewFormattingState()
	r.Display.IsTTY = r.isStdoutTTY

	if cfg.Terminal {
		rlConfig.Painter = NewHighlighter(r.Display)
		rlConfig.AutoComplete = r
		rlConfig.Listener = r
		rl, err := readline.NewEx(rlConfig)
		if err != nil {
			return nil, fmt.Errorf("initializing readline: %w", err)
		}
		r.rl = rl
	}

	r.Logger = NewLogger()
	r.Recorder = NewRecorder()
	r.Mocker = mock.NewMocker()

	return r, nil
}

// GetConfig returns the REPL configuration.
func (r *REPL) GetConfig() *Config {
	return r.config
}

// SetOnInput sets the fallback handler for input that is not a command.
func (r *REPL) SetOnInput(f func(ctx context.Context, text string) error) {
	r.onInput = f
}

// Done returns a channel that is closed when the REPL is closed.
func (r *REPL) Done() <-chan struct{} {
	return r.done
}

// Close closes the REPL and its readline instance.
func (r *REPL) Close() error {
	select {
	case <-r.done:
		return nil
	default:
		close(r.done)
	}
	// Close readline and wait for it to finish.
	if r.rl != nil {
		err := r.rl.Close()
		return err
	}
	return nil
}

// IsStdoutTTY returns true if the REPL's output is a terminal.
func (r *REPL) IsStdoutTTY() bool {
	return r.isStdoutTTY
}

// Printf prints a formatted string to the REPL output.
func (r *REPL) Printf(format string, args ...interface{}) {
	if r.rl != nil && r.IsInteractive {
		_, _ = r.rl.Write([]byte(fmt.Sprintf(format, args...)))
	} else if r.config != nil && r.config.Stdout != nil {
		_, _ = fmt.Fprintf(r.config.Stdout, format, args...)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, format, args...)
	}
}

// Errorf prints a formatted error string to the REPL stderr.
func (r *REPL) Errorf(format string, args ...interface{}) {
	if r.rl != nil && r.IsInteractive {
		_, _ = r.rl.Write([]byte(fmt.Sprintf(format, args...)))
	} else if r.config != nil && r.config.Stdout != nil {
		_, _ = fmt.Fprintf(r.config.Stdout, format, args...)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, format, args...)
	}
}

// Notify prints a notification (e.g. connection event) if quiet mode is not active.
func (r *REPL) Notify(format string, args ...interface{}) {
	if r.Display.Quiet {
		return
	}
	// If we are not in interactive mode, send notifications to stderr
	// to avoid polluting data streams on stdout.
	if !r.IsInteractive {
		_, _ = fmt.Fprintf(os.Stderr, format, args...)
		return
	}
	r.Printf(format, args...)
}

// PrintMessage formats and prints a WebSocket message if filtering allows.
func (r *REPL) PrintMessage(msg *ws.Message, conn *ws.Connection) {
	if msg.Metadata.Timestamp.IsZero() {
		msg.Metadata.Timestamp = time.Now()
	}
	r.mu.Lock()
	r.lastMsg = msg
	if !r.lastSendTime.IsZero() && msg.Metadata.Direction == "received" {
		r.lastLatency = time.Since(r.lastSendTime)
		r.lastSendTime = time.Time{} // Reset after receiving
	}
	r.mu.Unlock()

	// Handle logging and recording
	if r.Logger.IsActive() {
		_ = r.Logger.LogMessage(msg)
	}
	if r.Recorder.IsActive() {
		_ = r.Recorder.RecordMessage(msg)
	}

	// Handle mocking
	if msg.Metadata.Direction == "received" && r.Mocker.IsActive() {
		// If mock handles it, we might still want to print the original message
		// based on story requirements, usually mocks respond to incoming.
		// We'll pass a logger function to Mocker for observability.
		r.Mocker.MatchAndRespond(context.Background(), msg, conn, func(f string, a ...interface{}) {
			r.Notify(f+"\n", a...)
		})
	}

	formatted, ok := r.Display.FormatMessage(msg, r.GetVars(), r.TemplateEngine)
	if ok {
		// Suppression logic: if this is a sent message that exactly matches the last typed input,
		// we skip the terminal display to avoid redundancy, but we still log/record it (done above).
		if msg.Metadata.Direction == "sent" && msg.Type == ws.TextMessage {
			r.lastInputMu.Lock()
			li := r.lastInput
			r.lastInputMu.Unlock()
			if string(msg.Data) == li {
				return
			}
		}

		r.Printf("%s\n", formatted)
	}
}

// Run starts the REPL input loop.
func (r *REPL) Run(ctx context.Context) error {
	if r.rl == nil {
		return fmt.Errorf("REPL cannot run without an initialized terminal (Config.Terminal must be true)")
	}
	defer r.Close()

	// Handle SIGINT for command interruption
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	go func() {
		for {
			select {
			case <-sigCh:
				r.cmdMu.Lock()
				cancel := r.cmdCancel
				active := r.cmdActive
				r.cmdMu.Unlock()

				if active && cancel != nil {
					// Interrupt the current command
					cancel()
				}
			case <-r.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.done:
			return nil
		default:
			r.renderPrompt()
			line, err := r.rl.Readline()
			if err != nil {
				if err == readline.ErrInterrupt {
					if len(r.multiLineBuffer) > 0 {
						r.multiLineBuffer = nil
						r.rl.SetPrompt(r.config.Prompt)
						continue
					}
					// Clear the current line and continue the loop
					continue
				}
				if err == io.EOF {
					return nil
				}
				return err
			}

			// Handle multi-line continuation with \
			trimmedRight := strings.TrimRight(line, " \t")
			if strings.HasSuffix(trimmedRight, "\\") {
				r.multiLineBuffer = append(r.multiLineBuffer, strings.TrimSuffix(trimmedRight, "\\"))
				r.rl.SetPrompt(r.config.ContinuationPrompt)
				continue
			}

			// Combine buffer if any
			var finalInput string
			if r.heredocDelimiter != "" {
				if strings.TrimSpace(line) == r.heredocDelimiter {
					finalInput = strings.Join(r.heredocBuffer, "\n")
					r.heredocDelimiter = ""
					r.heredocBuffer = nil
					r.rl.SetPrompt(r.config.Prompt)
				} else {
					r.heredocBuffer = append(r.heredocBuffer, line)
					continue
				}
			} else if len(r.multiLineBuffer) > 0 {
				r.multiLineBuffer = append(r.multiLineBuffer, line)
				finalInput = strings.Join(r.multiLineBuffer, "\n")
				r.multiLineBuffer = nil
				r.rl.SetPrompt(r.config.Prompt)
			} else {
				// Detect heredoc start: <<EOF
				if idx := strings.LastIndex(line, "<<"); idx != -1 {
					delim := strings.TrimSpace(line[idx+2:])
					if delim != "" && !strings.ContainsAny(delim, " \t\"'") {
						r.heredocDelimiter = delim
						r.heredocBuffer = []string{line[:idx]}
						r.rl.SetPrompt(r.config.ContinuationPrompt)
						continue
					}
				}
				finalInput = line
			}

			trimmed := strings.TrimSpace(finalInput)
			if trimmed == "" {
				continue
			}

			if strings.HasPrefix(trimmed, ":") {
				if err := r.ExecuteCommand(ctx, trimmed); err != nil {
					if err == ErrExit {
						return nil
					}
					r.Errorf("Error: %v\n", err)
				}
			} else if r.onInput != nil {
				r.lastInputMu.Lock()
				r.lastInput = trimmed
				r.lastInputMu.Unlock()
				if err := r.onInput(ctx, finalInput); err != nil {
					r.Errorf("Error: %v\n", err)
				}
			}
		}
	}
}

// ExecuteCommand identifies and runs a REPL command.
func (r *REPL) ExecuteCommand(ctx context.Context, line string) error {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}

	// Handle bare text (non-command) as :send if it doesn't start with :
	if !strings.HasPrefix(trimmed, ":") {
		return r.ExecuteCommand(ctx, ":send "+trimmed)
	}

	// Intercept shell commands (:! <cmd>) before splitting
	// to preserve the raw command string (quotes, pipes, etc.)
	if strings.HasPrefix(trimmed, ":!") {
		shellCmd := strings.TrimPrefix(trimmed, ":!")
		return r.executeShellCommand(ctx, shellCmd)
	}

	parts := splitCommand(trimmed)
	if len(parts) == 0 {
		return nil
	}

	fullCmd := parts[0]
	cmdName := strings.TrimPrefix(fullCmd, ":")
	args := parts[1:]

	// 1. Check for script aliases first (with positional args)
	if aliasBody, ok := r.scriptAliases[cmdName]; ok {
		if r.execDepth >= 10 {
			return fmt.Errorf("maximum alias recursion depth (10) exceeded")
		}
		r.execDepth++
		defer func() { r.execDepth-- }()

		// Perform positional argument substitution
		expanded := aliasBody
		for i, arg := range args {
			placeholder := fmt.Sprintf("$%d", i+1)
			expanded = strings.ReplaceAll(expanded, placeholder, arg)
		}
		// Handle $@ (all args)
		expanded = strings.ReplaceAll(expanded, "$@", strings.Join(args, " "))

		// Optional: clean up unused placeholders $1-$9 (expanded to empty string as per story)
		for i := 1; i <= 9; i++ {
			expanded = strings.ReplaceAll(expanded, fmt.Sprintf("$%d", i), "")
		}

		return r.ExecuteCommand(ctx, expanded)
	}

	// 2. Check simple command aliases
	if alias, ok := r.aliases[cmdName]; ok {
		cmdName = alias
	}

	cmd, ok := r.commands[cmdName]
	if !ok {
		return fmt.Errorf("unknown command: %s", cmdName)
	}

	// Create a command-specific context that can be canceled by SIGINT
	cmdCtx, cancel := context.WithCancel(ctx)
	r.cmdMu.Lock()
	r.cmdCancel = cancel
	r.cmdActive = true
	r.cmdMu.Unlock()

	defer func() {
		r.cmdMu.Lock()
		r.cmdActive = false
		r.cmdCancel = nil
		r.cmdMu.Unlock()
		cancel()
	}()

	if err := cmd.Execute(cmdCtx, r, args); err != nil {
		if err == ErrExit {
			return ErrExit
		}
		// If command was canceled by us, return nil to continue REPL
		if cmdCtx.Err() == context.Canceled {
			return nil
		}
		return fmt.Errorf("command error: %w", err)
	}
	return nil
}

// renderPrompt evaluates the prompt template and updates the readline instance.
func (r *REPL) renderPrompt() {
	if r.rl == nil {
		return
	}

	// Do not override the continuation prompt if we are in a multi-line or heredoc state.
	if r.heredocDelimiter != "" || len(r.multiLineBuffer) > 0 {
		return
	}

	if r.promptTemplate == "" {
		if r.rl.Config.Prompt != r.originalPrompt {
			r.rl.Config.Prompt = r.originalPrompt
			r.rl.SetPrompt(r.originalPrompt)
		}
		return
	}

	tmplCtx := template.NewContext()
	r.mu.RLock()
	tmplCtx.Session = make(map[string]interface{}, len(r.vars))
	for k, v := range r.vars {
		tmplCtx.Session[k] = v
	}
	r.mu.RUnlock()
	tmplCtx.Vars = tmplCtx.Session

	// Populate connection info
	var conn *ws.Connection
	if r.clientCtx != nil {
		conn = r.clientCtx.GetConnection()
	}

	if conn != nil && !conn.IsClosed() {
		tmplCtx.URL = conn.GetURL()
		tmplCtx.ConnectionID = conn.ID
		tmplCtx.Subprotocol = conn.GetSubprotocol()
		tmplCtx.RemoteAddr = conn.RemoteAddr()
		tmplCtx.LocalAddr = conn.LocalAddr()
		tmplCtx.ConnectedSince = conn.ConnectedAt()
		tmplCtx.Uptime = time.Since(conn.ConnectedAt())
		tmplCtx.UptimeFormatted = template.FormatUptime(tmplCtx.Uptime)
		tmplCtx.MessageCount = conn.MessageCount()

		tmplCtx.Conn = &template.ConnectionContext{
			URL:                conn.GetURL(),
			Subprotocol:        conn.GetSubprotocol(),
			CompressionEnabled: conn.IsCompressionEnabled(),
			RemoteAddr:         conn.RemoteAddr(),
			LocalAddr:          conn.LocalAddr(),
			ConnectedAt:        conn.ConnectedAt(),
			Uptime:             tmplCtx.Uptime,
			UptimeFormatted:    tmplCtx.UptimeFormatted,
			MessageCount:       tmplCtx.MessageCount,
		}

		// Extract Host from URL
		if parts := strings.Split(tmplCtx.URL, "://"); len(parts) > 1 {
			host := strings.Split(parts[1], "/")[0]
			tmplCtx.Host = host
		}
	} else {
		tmplCtx.Host = "not connected"
		tmplCtx.ConnectionID = "🔌" // Plugs symbol for disconnected
		tmplCtx.RemoteAddr = "❓"
		tmplCtx.LocalAddr = "❓"
		tmplCtx.ClientIP = "❓"
		tmplCtx.MessageCount = 0
		tmplCtx.Uptime = 0
		tmplCtx.UptimeFormatted = "0s"
	}

	// Always set session meta even if not connected
	tmplCtx.SessionID = r.TemplateEngine.GetSessionID()

	res, err := r.TemplateEngine.Execute("prompt", r.promptTemplate, tmplCtx)
	if err != nil {
		if !r.promptErrReported {
			r.Errorf("\n[prompt template error: %v]\n", err)
			r.promptErrReported = true
		}
		r.rl.SetPrompt(r.originalPrompt)
		return
	}

	r.promptErrReported = false
	r.rl.Config.Prompt = res
	r.rl.SetPrompt(res)
}

// GetPrompt returns the current rendered prompt.
func (r *REPL) GetPrompt() string {
	if r.rl == nil {
		return ""
	}
	return r.rl.Config.Prompt
}

// GetLastMessage returns the last received message.
func (r *REPL) GetLastMessage() *ws.Message {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastMsg
}

// GetLastLatency returns the last message pair RTT.
func (r *REPL) GetLastLatency() time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lastLatency
}

// SetLastSendTime records the start of a message RTT measurement.
func (r *REPL) SetLastSendTime(t time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastSendTime = t
}

// Do satisfies the readline.AutoCompleter interface.
func (r *REPL) Do(line []rune, pos int) (newLine [][]rune, length int) {
	return r.DoContext(line, pos)
}

// OnChange implements the readline.Listener interface.
// It intercepts Ctrl+O (rune 15) to trigger shell mode.
func (r *REPL) OnChange(line []rune, pos int, key rune) (newLine []rune, newPos int, ok bool) {
	// Ctrl+O is rune 15. We intercept it to inject ":shell" command.
	if key == 15 {
		// We return the command with --yes to allow immediate entry upon pressing Enter.
		return []rune(":shell --yes"), 12, true
	}
	return nil, 0, false
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

// ReplaceVars replaces all session variables.
func (r *REPL) ReplaceVars(vars map[string]interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.vars = vars
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

func splitCommand(line string) []string {
	var parts []string
	var current strings.Builder
	inQuotes := false
	braceLevel := 0

	for i := 0; i < len(line); i++ {
		c := line[i]

		// Handle escapes
		if c == '\\' && i+1 < len(line) {
			current.WriteByte(line[i+1])
			i++
			continue
		}

		switch {
		case c == '"' || c == '\'':
			inQuotes = !inQuotes
			// Preserve quotes if we are inside a braced expression (like JSON)
			if braceLevel > 0 {
				current.WriteByte(c)
			}
		case c == '{' && !inQuotes:
			braceLevel++
			current.WriteByte(c)
		case c == '}' && !inQuotes:
			if braceLevel > 0 {
				braceLevel--
			}
			current.WriteByte(c)
		case c <= ' ' && !inQuotes && braceLevel == 0:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// openInEditor opens the given content in an external editor and returns the modified content.
func (r *REPL) openInEditor(ctx context.Context, initialContent string) (string, error) {
	// 1. Determine editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "vim"
		}
	}

	// 2. Create temporary file
	tmpFile, err := os.CreateTemp("", "xwebs-handler-*.yaml")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(initialContent); err != nil {
		_ = tmpFile.Close()
		return "", fmt.Errorf("writing to temp file: %w", err)
	}
	_ = tmpFile.Close()

	// 3. Spawn editor
	// We need to use exec.Command and connect Stdin/Stdout/Stderr to the real terminal.
	// We don't use CommandContext here because we want the editor to handle signals itself
	// (e.g. ^C in vim should not kill the editor process if the user is just typing).
	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor %q exited with error: %w", editor, err)
	}

	// 4. Read back
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("reading temp file: %w", err)
	}

	return string(content), nil
}

// AddConfigPath registers a configuration file for change tracking.
func (r *REPL) AddConfigPath(path string) {
	if path == "" {
		return
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, p := range r.configPaths {
		if p == abs {
			return
		}
	}
	r.configPaths = append(r.configPaths, abs)
}

// ReloadConfig reloads handlers and variables from a configuration file.
func (r *REPL) ReloadConfig(path string) error {
	cfg, err := handler.LoadConfig(path)
	if err != nil {
		return err
	}

	// Apply variables
	if cfg.Variables != nil {
		r.ReplaceVars(cfg.Variables)
	}

	// Replace handlers
	if r.Handlers == nil {
		r.Handlers = handler.NewRegistry()
	}
	r.Handlers.ReplaceHandlers(cfg.Handlers)

	return nil
}
