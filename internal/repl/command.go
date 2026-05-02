package repl

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/spf13/pflag"
	"golang.design/x/clipboard"
	"gopkg.in/yaml.v3"
	"runtime"
)

// Command defines the interface for a REPL command.
type Command interface {
	Name() string
	Help() string
	Execute(ctx context.Context, r *REPL, args []string) error
	IsVisible(r *REPL) bool
}

// BuiltinCommand is a simple implementation of the Command interface.
type BuiltinCommand struct {
	name         string
	help         string
	handler      func(ctx context.Context, r *REPL, args []string) error
	hideInServer bool
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

func (c *BuiltinCommand) IsVisible(r *REPL) bool {
	if c.hideInServer && r.mode == ServerMode {
		return false
	}
	return true
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
			for name, cmd := range r.commands {
				if cmd.IsVisible(r) {
					cmds = append(cmds, name)
				}
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
		name: "!",
		help: "Execute a shell command: :! [-i] <command>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: :! [-i] <command>")
			}
			// Note: This handler is a fallback. Direct :! usage is intercepted
			// in ExecuteCommand to preserve the raw command string including quotes/pipes.
			return r.executeShellCommand(ctx, strings.Join(args, " "))
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "shell",
		help: "Switch to a full interactive shell session: :shell [--yes|-y]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			var skipConfirm bool
			fs := pflag.NewFlagSet("shell", pflag.ContinueOnError)
			fs.SetOutput(nil)
			fs.BoolVarP(&skipConfirm, "yes", "y", false, "Skip confirmation prompt")

			if err := fs.Parse(args); err != nil {
				return fmt.Errorf("parsing flags: %w", err)
			}
			return r.switchToShell(ctx, skipConfirm)
		},
	})
	r.RegisterAlias("sh", "shell")

	r.RegisterCommand(&BuiltinCommand{
		name:         "set",
		help:         "Set a session variable: :set <key> <value>",
		hideInServer: true,
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if r.mode == ServerMode {
				r.Printf(":set is not available in server mode. Use :kv set / :kv get for shared state.\n")
				return nil
			}
			if len(args) < 2 {
				return fmt.Errorf("usage: :set <key> <value>")
			}
			r.SetVar(args[0], strings.Join(args[1:], " "))
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name:         "get",
		help:         "Get a session variable: :get <key>",
		hideInServer: true,
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if r.mode == ServerMode {
				r.Printf(":get is not available in server mode. Use :kv set / :kv get for shared state.\n")
				return nil
			}
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
		name:         "vars",
		help:         "List all session variables",
		hideInServer: true,
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if r.mode == ServerMode {
				r.Printf(":vars is not available in server mode. Use :kv set / :kv get for shared state.\n")
				return nil
			}
			vars := r.GetVars()
			if len(vars) == 0 {
				r.Printf("No session variables defined.\n")
				return nil
			}

			tw := table.NewWriter()
			tw.SetOutputMirror(nil)
			tw.AppendHeader(table.Row{"Variable", "Value"})

			keys := make([]string, 0, len(vars))
			for k := range vars {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				tw.AppendRow(table.Row{k, vars[k]})
			}

			tw.SetStyle(table.StyleColoredDark)
			tw.Style().Options.SeparateRows = false
			tw.Style().Options.SeparateColumns = true
			tw.Style().Options.DrawBorder = true

			r.Printf("\nSession Variables:\n%s\n", tw.Render())
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
		name: "cd",
		help: "Change the current working directory: :cd [path|-]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting current directory: %w", err)
			}

			var target string
			if len(args) == 0 {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("getting home directory: %w", err)
				}
				target = home
			} else if args[0] == "-" {
				if r.prevDir == "" {
					return fmt.Errorf("no previous directory")
				}
				target = r.prevDir
			} else {
				target = args[0]
			}

			// Clean path (remove quotes if any)
			target = strings.Trim(target, "\"'")

			if err := os.Chdir(target); err != nil {
				return fmt.Errorf("changing directory to %s: %w", target, err)
			}

			// Update prevDir to the OLD cwd
			r.prevDir = cwd

			// Get the NEW cwd for display and prompt
			newCwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting new current directory: %w", err)
			}

			colorized := r.Display.colorizedText(newCwd, "cyan")
			r.Printf("Directory changed to: %s\n", colorized)

			// Update the prompt if it depends on the directory
			r.renderPrompt()
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "ls",
		help: "List directory contents: :ls [-l] [path]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			var long bool
			var path string = "."

			fs := pflag.NewFlagSet("ls", pflag.ContinueOnError)
			fs.SetOutput(nil)
			fs.BoolVarP(&long, "long", "l", false, "Detailed listing")

			if err := fs.Parse(args); err != nil {
				return fmt.Errorf("parsing flags: %w", err)
			}

			remaining := fs.Args()
			if len(remaining) > 0 {
				path = remaining[0]
			}

			// Clean path (remove quotes if any - though splitCommand should have done it)
			path = strings.Trim(path, "\"'")

			entries, err := os.ReadDir(path)
			if err != nil {
				return fmt.Errorf("reading directory: %w", err)
			}

			if len(entries) == 0 {
				return nil
			}

			if long {
				r.Printf("\n%-12s %10s %-20s %s\n", "Mode", "Size", "Modified", "Name")
				r.Printf("%s\n", strings.Repeat("-", 62))
				for _, entry := range entries {
					info, err := entry.Info()
					if err != nil {
						continue
					}

					name := entry.Name()
					if entry.IsDir() {
						name = r.Display.colorizedText(name+"/", "cyan")
					}

					r.Printf("%-12s %10d %-20s %s\n",
						info.Mode().String(),
						info.Size(),
						info.ModTime().Format("2006-01-02 15:04:05"),
						name)
				}
			} else {
				for _, entry := range entries {
					name := entry.Name()
					if entry.IsDir() {
						name = r.Display.colorizedText(name+"/", "cyan")
					}
					r.Printf("  %s\n", name)
				}
			}
			return nil
		},
	})
	r.RegisterCommand(&BuiltinCommand{
		name: "mkdir",
		help: "Create a new directory: :mkdir [-p] <dirname>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			var parents bool
			fs := pflag.NewFlagSet("mkdir", pflag.ContinueOnError)
			fs.SetOutput(nil)
			fs.BoolVarP(&parents, "parents", "p", false, "Create parent directories as needed")

			if err := fs.Parse(args); err != nil {
				return fmt.Errorf("parsing flags: %w", err)
			}

			remaining := fs.Args()
			if len(remaining) < 1 {
				return fmt.Errorf("usage: :mkdir [-p] <dirname>")
			}

			path := strings.Trim(remaining[0], "\"'")

			var err error
			if parents {
				err = os.MkdirAll(path, 0755)
			} else {
				err = os.Mkdir(path, 0755)
			}

			if err != nil {
				return fmt.Errorf("creating directory %s: %w", path, err)
			}

			colorized := r.Display.colorizedText(path, "cyan")
			r.Printf("Directory created: %s\n", colorized)
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "cat",
		help: "Display file contents: :cat <filename>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: :cat <filename>")
			}
			path := strings.Trim(args[0], "\"'")

			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("opening file: %w", err)
			}
			defer f.Close()

			info, err := f.Stat()
			if err != nil {
				return fmt.Errorf("getting file info: %w", err)
			}
			if info.IsDir() {
				return fmt.Errorf("%s is a directory", path)
			}

			const maxLines = 500
			const maxSize = 1024 * 1024 // 1MB

			scanner := bufio.NewScanner(f)
			lineNum := 0
			totalSize := 0
			truncated := false

			for scanner.Scan() {
				lineNum++
				line := scanner.Text()
				totalSize += len(line) + 1

				if lineNum > maxLines || totalSize > maxSize {
					truncated = true
					break
				}

				highlighted := r.Display.HighlightLine(path, line)
				r.Printf("%s\n", highlighted)
			}

			if err := scanner.Err(); err != nil {
				return fmt.Errorf("reading file: %w", err)
			}

			if truncated {
				msg := r.Display.colorizedText(fmt.Sprintf("\n[Output truncated. File is too large. Showing first %d lines]", maxLines), "yellow")
				r.Printf("%s\n", msg)
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "edit",
		help: "Open a file in $EDITOR: :edit <filename>",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("usage: :edit <filename>")
			}
			path := strings.Trim(args[0], "\"'")

			// Handle absolute/relative paths
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}

			// Check if file exists
			info, err := os.Stat(absPath)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("file not found: %s", path)
				}
				return err
			}
			if info.IsDir() {
				return fmt.Errorf("%s is a directory", path)
			}

			// Get editor
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				if runtime.GOOS == "windows" {
					editor = "notepad"
				} else {
					editor = "vim"
				}
			}

			// Record mod time
			oldModTime := info.ModTime()

			// Run editor
			cmd := exec.Command(editor, absPath)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("editor %q exited with error: %w", editor, err)
			}

			// Check if modified
			newInfo, err := os.Stat(absPath)
			if err != nil {
				return fmt.Errorf("stat file after edit: %w", err)
			}

			if newInfo.ModTime().After(oldModTime) {
				// File was modified. Check if it's a known config file.
				isKnownConfig := false
				r.mu.RLock()
				for _, p := range r.configPaths {
					if p == absPath {
						isKnownConfig = true
						break
					}
				}
				r.mu.RUnlock()

				ext := strings.ToLower(filepath.Ext(absPath))
				isPotentialConfig := ext == ".yaml" || ext == ".yml" || ext == ".json"

				if isKnownConfig || isPotentialConfig {
					r.Printf("\nFile %q modified. Reload configuration? (y/N) ", path)

					// Read user input - Note: this might be tricky in some terminal environments
					// but since we are in a REPL, it should be okay.
					var answer string
					_, _ = fmt.Scanln(&answer)
					answer = strings.TrimSpace(strings.ToLower(answer))

					if answer == "y" || answer == "yes" {
						if err := r.ReloadConfig(absPath); err != nil {
							return fmt.Errorf("reloading configuration: %w", err)
						}
						r.Printf("Configuration reloaded successfully.\n")
					}
				}
			}

			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "history",
		help: "Display/manage command history: :history [-n N] [-s term] [-f pattern] [-e file] [--unique] [-r] [--json] [-c]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if r.config.HistoryFile == "" {
				return fmt.Errorf("history is not enabled (no history file configured)")
			}

			// Parse flags
			var (
				number     int
				clear      bool
				search     string
				filter     string
				exportFile string
				unique     bool
				reverse    bool
				jsonOut    bool
			)

			fs := pflag.NewFlagSet("history", pflag.ContinueOnError)
			fs.SetOutput(nil)
			fs.IntVarP(&number, "number", "n", 0, "Show last N commands")
			fs.BoolVarP(&clear, "clear", "c", false, "Clear the entire history")
			fs.StringVarP(&search, "search", "s", "", "Search history for term")
			fs.StringVarP(&filter, "filter", "f", "", "Filter with glob or /regex/ pattern")
			fs.StringVarP(&exportFile, "export", "e", "", "Export history to file")
			fs.BoolVar(&unique, "unique", false, "Show only unique commands")
			fs.BoolVarP(&reverse, "reverse", "r", false, "Display in reverse chronological order")
			fs.BoolVar(&jsonOut, "json", false, "Output in structured JSON format")

			if err := fs.Parse(args); err != nil {
				return fmt.Errorf("parsing flags: %w", err)
			}

			// Backward compatibility: :history <N> (positional arg)
			if number == 0 {
				remaining := fs.Args()
				if len(remaining) > 0 {
					if val, err := strconv.Atoi(remaining[0]); err == nil {
						number = val
					}
				}
			}
			if number <= 0 {
				number = 20
			}

			// Handle --clear first (doesn't need to load full history)
			if clear {
				return r.historyClear()
			}

			// Load history
			data, err := os.ReadFile(r.config.HistoryFile)
			if err != nil {
				if os.IsNotExist(err) {
					r.Printf("No history found.\n")
					return nil
				}
				return fmt.Errorf("reading history file: %w", err)
			}

			lines := strings.Split(string(data), "\n")
			var entries []historyEntry
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					entries = append(entries, historyEntry{Index: len(entries) + 1, Cmd: line})
				}
			}

			if len(entries) == 0 {
				r.Printf("No history found.\n")
				return nil
			}

			// Pipeline: search -> filter -> unique -> slice -> reverse

			// Stage: Search
			if search != "" {
				var filtered []historyEntry
				lowerSearch := strings.ToLower(search)
				for _, e := range entries {
					if strings.Contains(strings.ToLower(e.Cmd), lowerSearch) {
						filtered = append(filtered, e)
					}
				}
				entries = filtered
			}

			// Stage: Filter (glob or regex)
			if filter != "" {
				var filtered []historyEntry
				if strings.HasPrefix(filter, "/") && strings.HasSuffix(filter, "/") && len(filter) > 2 {
					// Regex mode
					re, err := regexp.Compile(filter[1 : len(filter)-1])
					if err != nil {
						return fmt.Errorf("invalid regex pattern: %w", err)
					}
					for _, e := range entries {
						if re.MatchString(e.Cmd) {
							filtered = append(filtered, e)
						}
					}
				} else {
					// Glob mode
					for _, e := range entries {
						matched, err := filepath.Match(filter, e.Cmd)
						if err != nil {
							return fmt.Errorf("invalid glob pattern: %w", err)
						}
						if matched {
							filtered = append(filtered, e)
						}
					}
				}
				entries = filtered
			}

			// Stage: Unique (keep last occurrence)
			if unique {
				seen := make(map[string]int) // cmd -> index in result
				var deduped []historyEntry
				for _, e := range entries {
					if idx, ok := seen[e.Cmd]; ok {
						// Replace earlier occurrence
						deduped[idx] = e
					} else {
						seen[e.Cmd] = len(deduped)
						deduped = append(deduped, e)
					}
				}
				// Compact: remove entries that were replaced
				// Rebuild to keep only the latest of each command in order
				seen2 := make(map[string]bool)
				var compacted []historyEntry
				for i := len(entries) - 1; i >= 0; i-- {
					if !seen2[entries[i].Cmd] {
						seen2[entries[i].Cmd] = true
						compacted = append(compacted, entries[i])
					}
				}
				// Reverse to restore chronological order
				for i, j := 0, len(compacted)-1; i < j; i, j = i+1, j-1 {
					compacted[i], compacted[j] = compacted[j], compacted[i]
				}
				entries = compacted
			}

			// Stage: Slice (last N)
			if number > 0 && number < len(entries) {
				entries = entries[len(entries)-number:]
			}

			// Stage: Reverse
			if reverse {
				for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
					entries[i], entries[j] = entries[j], entries[i]
				}
			}

			// Handle export
			if exportFile != "" {
				return r.historyExport(exportFile, entries, jsonOut)
			}

			// Handle JSON output
			if jsonOut {
				jsonData, err := json.MarshalIndent(entries, "", "  ")
				if err != nil {
					return fmt.Errorf("marshalling JSON: %w", err)
				}
				r.Printf("%s\n", string(jsonData))
				return nil
			}

			// Plain text output
			if len(entries) == 0 {
				r.Printf("No matching history entries.\n")
				return nil
			}

			label := fmt.Sprintf("last %d", len(entries))
			if search != "" {
				label = fmt.Sprintf("%d matching", len(entries))
			} else if filter != "" {
				label = fmt.Sprintf("%d filtered", len(entries))
			}
			r.Printf("\nCommand History (%s):\n", label)

			for _, e := range entries {
				displayCmd := e.Cmd
				// Highlight search term if present
				if search != "" {
					displayCmd = r.highlightSearchTerm(displayCmd, search)
				}
				r.Printf("  %4d  %s\n", e.Index, displayCmd)
			}
			return nil
		},
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "hedit",
		help: "Edit a previous command in $EDITOR: :hedit [-n <number>]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			if r.config.HistoryFile == "" {
				return fmt.Errorf("history is not enabled (no history file configured)")
			}

			var historyNum int
			fs := pflag.NewFlagSet("hedit", pflag.ContinueOnError)
			fs.SetOutput(nil)
			fs.IntVarP(&historyNum, "number", "n", 0, "History item number to edit")

			if err := fs.Parse(args); err != nil {
				return fmt.Errorf("parsing flags: %w", err)
			}

			data, err := os.ReadFile(r.config.HistoryFile)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("no history found")
				}
				return fmt.Errorf("reading history file: %w", err)
			}

			lines := strings.Split(string(data), "\n")
			var history []string
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					history = append(history, line)
				}
			}

			if len(history) == 0 {
				return fmt.Errorf("no history found")
			}

			// Find target index
			var targetIdx int = -1
			if historyNum > 0 {
				targetIdx = historyNum - 1
				if targetIdx >= len(history) {
					return fmt.Errorf("history item %d not found (max %d)", historyNum, len(history))
				}
			} else {
				// Search backwards for the first item that is NOT :hedit
				for i := len(history) - 1; i >= 0; i-- {
					if !strings.HasPrefix(strings.TrimSpace(history[i]), ":hedit") {
						targetIdx = i
						break
					}
				}
			}

			var content string
			if targetIdx >= 0 {
				// Use robust block detection
				startBlock, endBlock := findHistoryBlock(history, targetIdx)
				targetLines := history[startBlock : endBlock+1]
				content = strings.Join(targetLines, "\n")
			}

			// Check for editor
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				r.Printf("Note: $EDITOR not set, checking defaults...\n")
			}

			// Open in editor
			newContent, err := r.openInEditor(ctx, content)
			if err != nil {
				return err
			}

			// Remove trailing newline if editor added one and original didn't have it
			// or just normalize.
			newContent = strings.TrimRight(newContent, "\n\r")

			if newContent == content || newContent == "" {
				return nil
			}

			// Feed the edited content line-by-line through the main loop.
			// This ensures heredocs and slash continuations are processed correctly.
			r.pendingLines = strings.Split(newContent, "\n")
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

			tw := table.NewWriter()
			tw.SetOutputMirror(nil)
			tw.AppendHeader(table.Row{"Name", "Matches", "Avg Latency", "Errors", "Status", "Pattern"})

			for _, h := range handlers {
				matches, totalLatency, errors, ok := r.Handlers.GetStats(h.Name)
				avgLatencyStr := "-"
				if ok && matches > 0 {
					avgLatencyStr = (totalLatency / time.Duration(matches)).Round(time.Microsecond).String()
				}
				matchesStr := "-"
				if ok {
					matchesStr = fmt.Sprintf("%d", matches)
				}
				errorsStr := "-"
				if ok {
					errorsStr = fmt.Sprintf("%d", errors)
				}

				status := text.FgGreen.Sprint("enabled")
				if r.Handlers.IsDisabled(h.Name) {
					reason := r.Handlers.GetDisabledReason(h.Name)
					if reason != "" && reason != "user" {
						status = text.FgRed.Sprintf("disabled (%s)", reason)
					} else {
						status = text.FgRed.Sprint("disabled")
					}
				}

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

				tw.AppendRow(table.Row{
					h.Name,
					matchesStr,
					avgLatencyStr,
					errorsStr,
					status,
					matcherStr,
				})
			}

			tw.SetStyle(table.StyleColoredDark)
			tw.Style().Options.SeparateRows = false
			tw.Style().Options.SeparateColumns = true
			tw.Style().Options.DrawBorder = true

			r.Printf("%s\n", tw.Render())

			// Detailed actions list below the table for clarity
			r.Printf("\nDetailed Actions:\n")
			for i, h := range handlers {
				r.Printf("  %d. %s:\n", i+1, h.Name)
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
				r.Printf("  -B, --builtin <name>  Builtin action (noop, redis-subscribe, ollama-classify, ollama-generate, ollama-chat, ollama-embed, openai-chat)\n")
				r.Printf("  --topic <template>    Topic name template for builtin actions\n")
				r.Printf("  --exclusive           Stop further matching if this handler matches\n")
				r.Printf("  --sequential          Run handler actions sequentially (disable concurrency)\n")
				r.Printf("  --rate-limit <limit>  Rate limit (e.g. '10/s')\n")
				r.Printf("  --debounce <duration> Debounce time (e.g. '500ms')\n")
				r.Printf("  -w, --window <dur>    Window duration for 'debounce' or 'throttle-broadcast'\n")
				r.Printf("  --scope <type>        Scope for 'debounce' or 'rate-limit' (client|global)\n")
				r.Printf("  --file <path>         File path for 'file-send' or 'template' builtin\n")
				r.Printf("  --path <path>         File path for 'file-write' builtin\n")
				r.Printf("  --content <template>  Content template for 'file-write' builtin\n")
				r.Printf("  --mode <type>         Mode for 'file-send' (text|binary) or 'file-write' (overwrite|append)\n")
				r.Printf("  --responses <list>    Responses for 'sequence' builtin (repeatable)\n")
				r.Printf("  --loop                Loop the sequence\n")
				r.Printf("  --per-client          Track sequence position independently per client\n")
				r.Printf("  --code <code>         Close code for 'close' builtin (e.g. 1000, template)\n")
				r.Printf("  --reason <reason>     Close reason for 'close' builtin (supports templates)\n")
				r.Printf("  --body <template>     Body for 'http' builtin\n")
				r.Printf("  --timeout <duration>  Timeout for 'http' builtin (e.g. '5s')\n")
				r.Printf("  --message <template>  Message template for 'log' builtin\n")
				r.Printf("  --target <type>       Target for 'forward' (URL), 'log' (stdout|file|both), or 'shadow' (handler name)\n")
				r.Printf("  --labels <key=val,...> Labels for 'metric' (k=v) or 'ollama-classify' (list)\n")
				r.Printf("  --script <script>     Inline Lua script\n")
				r.Printf("  --max-memory <n>      Memory limit for Lua VM in bytes\n")
				r.Printf("  --targets <ids>       Comma-separated list of client IDs or JSON array for 'multicast' builtin\n")
				r.Printf("  --rule-when, -W <matcher> Condition for rule-engine rule (repeatable)\n")
				r.Printf("  --rule-respond, -S <tmpl> Response for rule-engine rule (repeatable)\n")
				r.Printf("  --default, -D <tmpl>      Default response for rule-engine or KV builtins\n")
				r.Printf("  --key <template>      Key template for KV or Redis builtins\n")
				r.Printf("  --value <template>    Value template for KV or Redis builtins\n")
				r.Printf("  --ttl <duration>      TTL template for KV or Redis builtins\n")
				r.Printf("  --channel <name>      Redis channel for 'redis-publish' or 'redis-subscribe'\n")
				r.Printf("  --by <n>              Increment value for 'redis-incr' builtin\n")
				r.Printf("  --reconnect-interval <dur> Reconnect interval for 'redis-subscribe'\n")
				r.Printf("  --on-error <tmpl>     Error template for 'redis-subscribe'\n")
				r.Printf("  --model <name>        Ollama model name (template)\n")
				r.Printf("  --prompt <template>   Prompt template for 'ollama-generate'\n")
				r.Printf("  --labels <list>       Comma-separated labels for 'ollama-classify' (template)\n")
				r.Printf("  --ollama-url <url>    Ollama API URL override (template)\n")
				r.Printf("  --stream-ollama       Enable streaming for 'ollama-generate'\n")
				r.Printf("  --system <template>   System prompt for 'ollama-chat' or 'openai-chat'\n")
				r.Printf("  --max-history <n>     Max message history to retain for 'ollama-chat' or 'openai-chat'\n")
				r.Printf("  -i, --input <template> Input template for 'ollama-embed'\n")
				r.Printf("  --api-url <url>       API URL for 'openai-chat' (template)\n")
				r.Printf("  --api-key <key>       API Key for 'openai-chat' (template)\n")
				r.Printf("  --temperature <n>     Temperature for 'openai-chat'\n")
				r.Printf("  --top-p <n>           Top-P for 'openai-chat'\n")
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
					if err := cfg.Validate(r.getHandlerMode()); err != nil {
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

					if err := newCfg.Validate(r.getHandlerMode()); err != nil {
						return fmt.Errorf("validation failed: %w", err)
					}

					// Apply changes
					if err := r.Handlers.ReplaceHandlers(newCfg.Handlers); err != nil {
						return err
					}
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
				r.Handlers = handler.NewRegistry(r.getHandlerMode())
			}

			// Parse flags robustly using pflag
			fs := pflag.NewFlagSet("handler add", pflag.ContinueOnError)
			fs.SetOutput(nil) // Suppress automatic usage printing on error

			var name, match, matchType, run, respond, builtin, topic, rateLimit, debounce, code, reason, message, target, script, targets, window, scope, channel, reconnectInterval, onError string
			var key, value, ttl, by, model, prompt, ollamaURL, system, input, apiKey, apiURL, brokerURL, mqttTopic, qos, natsURL, natsSubject string
			var retain bool
			var labels []string
			var priority, maxMemory, maxHistory int
			var exclusive, sequential, streamOllama bool
			var temperature, topP float64

			fs.StringVarP(&name, "name", "n", "", "Name of the handler")
			fs.StringVarP(&match, "match", "m", "", "Match pattern")
			fs.StringVarP(&matchType, "match-type", "t", "", "Match type")
			fs.IntVarP(&priority, "priority", "p", 0, "Priority")
			fs.StringVarP(&run, "run", "r", "", "Shell command")
			fs.StringVarP(&respond, "respond", "R", "", "Response template")
			fs.StringVarP(&builtin, "builtin", "B", "", "Builtin action (noop)")
			fs.StringVar(&topic, "topic", "", "Topic name template for builtin actions")
			fs.BoolVarP(&exclusive, "exclusive", "e", false, "Short-circuit match")
			fs.BoolVarP(&sequential, "sequential", "s", false, "Run actions sequentially")
			fs.StringVarP(&rateLimit, "rate-limit", "l", "", "Rate limit")
			fs.StringVar(&code, "code", "", "Close code for close builtin")
			fs.StringVar(&reason, "reason", "", "Close reason for close builtin")
			fs.StringVarP(&debounce, "debounce", "d", "", "Debounce duration")
			fs.StringVarP(&window, "window", "w", "", "Window duration")
			fs.StringVar(&scope, "scope", "", "Scope")
			fs.StringVarP(&message, "message", "M", "", "Message template (for broadcast or log)")
			fs.StringVar(&target, "target", "", "Target URL (forward) or destination type (log: stdout|file|both)")
			fs.StringVar(&key, "key", "", "Key template for KV or Redis builtins")
			fs.StringVar(&value, "value", "", "Value template for KV or Redis builtins")
			fs.StringVar(&ttl, "ttl", "", "TTL template for KV or Redis builtins")
			fs.StringVar(&by, "by", "", "Increment value template for redis-incr builtin")
			var file, path, content, mode string
			var responses []string
			var loop, perClient bool
			fs.StringVar(&file, "file", "", "File path")
			fs.StringVar(&path, "path", "", "Path for file-write")
			fs.StringVar(&content, "content", "", "Content for file-write")
			fs.StringVar(&mode, "mode", "", "Mode (text|binary or overwrite|append)")
			fs.StringArrayVar(&responses, "responses", nil, "Sequence responses")
			fs.BoolVar(&loop, "loop", false, "Loop sequence")
			fs.BoolVar(&perClient, "per-client", false, "Track per client")
			var url, method, body, timeout string
			var headers []string
			fs.StringVar(&url, "url", "", "URL for http builtin")
			fs.StringVar(&method, "method", "", "Method for http builtin")
			fs.StringArrayVar(&headers, "header", nil, "Headers for http builtin (key:value)")
			fs.StringVar(&body, "body", "", "Body for http builtin")
			fs.StringVar(&timeout, "timeout", "", "Timeout for http builtin")
			fs.StringVar(&script, "script", "", "Inline Lua script")
			fs.IntVar(&maxMemory, "max-memory", 0, "Max memory for Lua VM")
			fs.StringVar(&targets, "targets", "", "Targets (comma-separated list or JSON array) for multicast builtin")
			fs.StringSliceVar(&labels, "labels", nil, "Labels for metric builtin (k=v) or ollama-classify (list)")
			var ruleWhens, ruleResponds []string
			var defaultResp string
			fs.StringArrayVarP(&ruleWhens, "rule-when", "W", nil, "Condition for rule-engine rule")
			fs.StringArrayVarP(&ruleResponds, "rule-respond", "S", nil, "Response for rule-engine rule")
			fs.StringVarP(&defaultResp, "default", "D", "", "Default response template")
			var field, handlerA, handlerB string
			var split int
			fs.StringVar(&field, "field", "", "Field to hash for ab-test")
			fs.IntVar(&split, "split", 0, "Percentage for handler_a in ab-test")
			fs.StringVar(&handlerA, "handler-a", "", "Handler A for ab-test")
			fs.StringVar(&handlerB, "handler-b", "", "Handler B for ab-test")
			fs.StringVar(&channel, "channel", "", "Redis channel name for redis-publish or redis-subscribe")
			fs.StringVar(&reconnectInterval, "reconnect-interval", "", "Reconnect interval for redis-subscribe")
			fs.StringVar(&onError, "on-error", "", "Error response template")
			fs.StringVar(&model, "model", "", "Ollama model name")
			fs.StringVar(&prompt, "prompt", "", "Prompt template for ollama-generate")
			fs.StringVar(&ollamaURL, "ollama-url", "", "Ollama API URL")
			fs.BoolVar(&streamOllama, "stream-ollama", false, "Enable streaming")
			fs.StringVar(&system, "system", "", "System prompt template")
			fs.IntVar(&maxHistory, "max-history", 0, "Max history turns to retain")
			fs.StringVarP(&input, "input", "i", "", "Input template for ollama-embed")
			fs.StringVar(&apiKey, "api-key", "", "API Key for openai-chat")
			fs.StringVar(&apiURL, "api-url", "", "API URL for openai-chat")
			fs.Float64Var(&temperature, "temperature", 0, "Temperature for openai-chat")
			fs.Float64Var(&topP, "top-p", 0, "Top-P for openai-chat")

			// MQTT flags
			fs.StringVar(&brokerURL, "broker-url", "", "Broker URL for mqtt-publish")
			fs.StringVar(&mqttTopic, "mqtt-topic", "", "Topic for mqtt-publish")
			fs.StringVar(&qos, "qos", "", "QoS for mqtt-publish")
			fs.BoolVar(&retain, "retain", false, "Retain flag for mqtt-publish")
 
			// NATS flags
			fs.StringVar(&natsURL, "nats-url", "", "NATS server URL for nats-publish")
			fs.StringVar(&natsSubject, "subject", "", "Subject for nats-publish")

			if err := fs.Parse(args[1:]); err != nil {
				if errors.Is(err, pflag.ErrHelp) {
					// Use our centralized help printer for uniformity
					_ = r.commands["handler"].Execute(ctx, r, nil)
					return nil
				}
				return fmt.Errorf("parsing flags: %w", err)
			}

			// Validation
			if match == "" && builtin != "redis-subscribe" && builtin != "mqtt-subscribe" {
				return fmt.Errorf("--match is required")
			}

			// Auto-generate name if missing
			if name == "" {
				name = namesgenerator.GetRandomName(0)
			}

			// Construct handler
			h := handler.Handler{
				Name:      name,
				Priority:  priority,
				Exclusive: exclusive,
				Run:       run,
				Respond:   respond,
				Builtin:   builtin,
				Topic:     topic,
				Message:   message,
				Target:    target,
				Match: func() handler.Matcher {
					m := handler.AutoDetectMatcher(match)
					if matchType != "" {
						m.Type = matchType
					}
					return m
				}(),
				RateLimit: rateLimit,
				Debounce:  debounce,
				Window:    window,
				Scope:     scope,
				Code:      code,
				Reason:    reason,
				File:      file,
				Path:      path,
				Content:   content,
				URL:       url,
				Method:    method,
				Body:      body,
				Timeout:   timeout,
				Mode:      mode,
				Responses: responses,
				Loop:      loop,
				PerClient: perClient,
				APIKey:    apiKey,
				APIURL:    apiURL,
				BrokerURL: brokerURL,
				QoS:       qos,
				Retain:    retain,
				Temperature: func() *float64 {
					if fs.Changed("temperature") {
						return &temperature
					}
					return nil
				}(),
				TopP: func() *float64 {
					if fs.Changed("top-p") {
						return &topP
					}
					return nil
				}(),
				Labels: func() handler.FlexLabels {
					var fl handler.FlexLabels
					if len(labels) > 0 {
						// Try to detect if it's a map (metric) or a list (classify)
						isMap := true
						for _, l := range labels {
							if !strings.Contains(l, "=") {
								isMap = false
								break
							}
						}

						if isMap {
							m := make(map[string]string)
							for _, l := range labels {
								parts := strings.SplitN(l, "=", 2)
								if len(parts) == 2 {
									m[parts[0]] = parts[1]
								}
							}
							fl.Map = m
						} else {
							fl.List = labels
						}
					}
					return fl
				}(),
				Script:    script,
				MaxMemory: maxMemory,
				Targets:   targets,
				Default:   defaultResp,
				Field:     field,
				HandlerA:  handlerA,
				HandlerB:  handlerB,
				Key:       key,
				Value:     value,
				By:        by,
				TTL:       ttl,
				Channel:   channel,
				ReconnectInterval: reconnectInterval,
				OnErrorMsg:        onError,
				Model:             model,
				Prompt:            prompt,
				OllamaURL:         ollamaURL,
				System:            system,
				MaxHistory:        maxHistory,
				Input:             input,
				NatsURL:           natsURL,
				Subject:           natsSubject,
			}
			if streamOllama {
				h.Stream = "true"
			}

			if mqttTopic != "" {
				h.Topic = mqttTopic
			}

			if fs.Changed("split") {
				h.Split = &split
			}

			if len(ruleWhens) > 0 {
				if len(ruleWhens) != len(ruleResponds) {
					return fmt.Errorf("number of --rule-when must match number of --rule-respond")
				}
				for i := range ruleWhens {
					h.Rules = append(h.Rules, handler.Rule{
						When:    handler.AutoDetectMatcher(ruleWhens[i]),
						Respond: ruleResponds[i],
					})
				}
			}

			if len(headers) > 0 {
				h.Headers = make(map[string]string)
				for _, hdr := range headers {
					parts := strings.SplitN(hdr, ":", 2)
					if len(parts) == 2 {
						h.Headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
					}
				}
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

	r.RegisterCommand(&BuiltinCommand{
		name:    "write",
		help:    "Save content to a file: :write [flags] <filename> [content]",
		handler: r.executeWrite,
	})

	r.RegisterCommand(&BuiltinCommand{
		name: "shortcuts",
		help: "List all active keyboard shortcuts: :shortcuts [-d|--defaults]",
		handler: func(ctx context.Context, r *REPL, args []string) error {
			var showDefaults bool
			fs := pflag.NewFlagSet("shortcuts", pflag.ContinueOnError)
			fs.SetOutput(nil)
			fs.BoolVarP(&showDefaults, "defaults", "d", false, "Show default readline key bindings")

			if err := fs.Parse(args); err != nil {
				return fmt.Errorf("parsing flags: %w", err)
			}

			if showDefaults {
				r.Printf("\nDefault Readline Shortcuts:\n")
				r.Printf("  %-10s  %s\n", "Shortcut", "Action")
				r.Printf("  %-10s  %s\n", "----------", "------")
				r.Printf("  %-10s  %s\n", "Ctrl+A", "Beginning of line")
				r.Printf("  %-10s  %s\n", "Ctrl+B", "Backward one character")
				r.Printf("  %-10s  %s\n", "Ctrl+E", "End of line")
				r.Printf("  %-10s  %s\n", "Ctrl+F", "Forward one character")
				r.Printf("  %-10s  %s\n", "Ctrl+H", "Delete previous character (Backspace)")
				r.Printf("  %-10s  %s\n", "Ctrl+I", "Completion (Tab)")
				r.Printf("  %-10s  %s\n", "Ctrl+K", "Cut text to the end of line")
				r.Printf("  %-10s  %s\n", "Ctrl+L", "Clear screen")
				r.Printf("  %-10s  %s\n", "Ctrl+N", "Next line in history")
				r.Printf("  %-10s  %s\n", "Ctrl+P", "Previous line in history")
				r.Printf("  %-10s  %s\n", "Ctrl+R", "Search backwards in history")
				r.Printf("  %-10s  %s\n", "Ctrl+S", "Search forwards in history")
				r.Printf("  %-10s  %s\n", "Ctrl+U", "Cut text to the beginning of line")
				r.Printf("  %-10s  %s\n", "Ctrl+W", "Cut previous word")
				r.Printf("  %-10s  %s\n", "Meta+B", "Backward one word")
				r.Printf("  %-10s  %s\n", "Meta+F", "Forward one word")
				r.Printf("  %-10s  %s\n", "Meta+D", "Delete one word")
				return nil
			}

			if len(r.shortcuts) == 0 {
				r.Printf("No custom shortcuts defined. Use -d to see defaults.\n")
				return nil
			}

			// Reverse map for display (rune -> name)
			runeToName := map[rune]string{
				1: "Ctrl+A", 2: "Ctrl+B", 3: "Ctrl+C", 4: "Ctrl+D", 5: "Ctrl+E", 6: "Ctrl+F", 7: "Ctrl+G",
				8: "Ctrl+H", 9: "Ctrl+I", 10: "Ctrl+J", 11: "Ctrl+K", 12: "Ctrl+L", 13: "Ctrl+M", 14: "Ctrl+N",
				15: "Ctrl+O", 16: "Ctrl+P", 17: "Ctrl+Q", 18: "Ctrl+R", 19: "Ctrl+S", 20: "Ctrl+T", 21: "Ctrl+U",
				22: "Ctrl+V", 23: "Ctrl+W", 24: "Ctrl+X", 25: "Ctrl+Y", 26: "Ctrl+Z",
			}

			var keys []rune
			for k := range r.shortcuts {
				keys = append(keys, k)
			}
			sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

			r.Printf("\nActive Custom Shortcuts:\n")
			for _, k := range keys {
				name, ok := runeToName[k]
				if !ok {
					name = fmt.Sprintf("Key(%d)", k)
				}
				r.Printf("  %-10s -> %s\n", name, r.shortcuts[k])
			}
			return nil
		},
	})
	r.RegisterAlias("keys", "shortcuts")
}

func (r *REPL) executeWrite(ctx context.Context, _ *REPL, args []string) error {
	var appendMode, prettyJSON, dryRun, parents, showDiff, edit bool
	var lastMsg, lastResp, curHandlers, clip bool

	fs := pflag.NewFlagSet("write", pflag.ContinueOnError)
	fs.SetOutput(nil)
	fs.BoolVarP(&appendMode, "append", "a", false, "Append to file instead of overwriting")
	fs.BoolVar(&prettyJSON, "json", false, "Pretty-print content as JSON")
	fs.BoolVarP(&dryRun, "dry-run", "n", false, "Preview rendered output without writing to disk")
	fs.BoolVarP(&parents, "parents", "p", false, "Create parent directories if they don't exist")
	fs.BoolVar(&showDiff, "diff", false, "Show diff summary when overwriting")
	fs.BoolVar(&edit, "edit", false, "Open the written file in $EDITOR immediately")
	fs.BoolVar(&lastMsg, "last-message", false, "Use the last message as content")
	fs.BoolVar(&lastResp, "last-response", false, "Use the last response from server as content")
	fs.BoolVar(&curHandlers, "current-handlers", false, "Use the currently loaded handlers as content")
	fs.BoolVar(&clip, "clipboard", false, "Use clipboard content")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	remaining := fs.Args()
	if len(remaining) < 1 && !(lastMsg || lastResp || curHandlers || clip) {
		return fmt.Errorf("usage: :write [flags] <filename> [content]")
	}

	var filename string
	var rawContent string

	if len(remaining) > 0 {
		filename = remaining[0]
		if len(remaining) > 1 {
			rawContent = strings.Join(remaining[1:], " ")
		}
	}

	// Determine content source
	sourcesCount := 0
	if rawContent != "" {
		sourcesCount++
	}
	if lastMsg {
		sourcesCount++
	}
	if lastResp {
		sourcesCount++
	}
	if curHandlers {
		sourcesCount++
	}
	if clip {
		sourcesCount++
	}

	if sourcesCount > 1 {
		return fmt.Errorf("multiple content sources provided. Choose only one (content arg, --last-message, --last-response, --current-handlers, or --clipboard)")
	}

	if filename == "" {
		// Try to guess filename if not provided but source is
		if lastMsg || lastResp {
			filename = "message.json"
		} else if curHandlers {
			filename = "handlers.yaml"
		} else if clip {
			filename = "clipboard.txt"
		} else {
			return fmt.Errorf("filename is required")
		}
	}

	var finalContent string
	if lastMsg || lastResp {
		msg := r.GetLastMessage()
		if msg == nil {
			return fmt.Errorf("no last message available")
		}
		finalContent = string(msg.Data)
	} else if curHandlers {
		handlers := r.Handlers.Handlers()
		data, err := yaml.Marshal(handlers)
		if err != nil {
			return fmt.Errorf("marshaling handlers: %w", err)
		}
		finalContent = string(data)
	} else if clip {
		err := clipboard.Init()
		if err != nil {
			return fmt.Errorf("initializing clipboard: %w", err)
		}
		data := clipboard.Read(clipboard.FmtText)
		finalContent = string(data)
	} else {
		// Render template content
		tmplCtx := template.NewContext()
		tmplCtx.Session = r.GetVars()
		tmplCtx.Vars = tmplCtx.Session
		tmplCtx.Env = make(map[string]string)
		for _, e := range os.Environ() {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				tmplCtx.Env[parts[0]] = parts[1]
			}
		}

		if last := r.GetLastMessage(); last != nil {
			tmplCtx.Last = string(last.Data)
			tmplCtx.Msg = &template.MessageContext{
				Data:      string(last.Data),
				Raw:       last.Data,
				Length:    last.Metadata.Length,
				Timestamp: last.Metadata.Timestamp,
			}
		}
		tmplCtx.LastLatencyMs = r.GetLastLatency().Milliseconds()

		rendered, err := r.TemplateEngine.Execute("write", rawContent, tmplCtx)
		if err != nil {
			return fmt.Errorf("rendering content: %w", err)
		}
		finalContent = rendered
	}

	// JSON pretty-print
	if prettyJSON {
		var data interface{}
		if err := json.Unmarshal([]byte(finalContent), &data); err == nil {
			pretty, _ := json.MarshalIndent(data, "", "  ")
			finalContent = string(pretty)
		}
	}

	// Guess extension if missing
	if filepath.Ext(filename) == "" {
		if prettyJSON || lastMsg || lastResp {
			// If we explicitly asked for JSON or are using messages (usually JSON), default to .json
			filename += ".json"
		} else if curHandlers {
			filename += ".yaml"
		} else {
			trimmed := strings.TrimSpace(finalContent)
			// Strip outer quotes if any (can happen if splitCommand didn't strip them due to nested braces)
			if (strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'")) ||
				(strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"")) {
				trimmed = trimmed[1 : len(trimmed)-1]
			}
			trimmed = strings.TrimSpace(trimmed)

			if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
				var data interface{}
				if err := json.Unmarshal([]byte(trimmed), &data); err == nil {
					filename += ".json"
				} else {
					filename += ".txt"
				}
			} else if strings.Contains(finalContent, ": ") {
				filename += ".yaml"
			} else {
				filename += ".log"
			}
		}
	}

	if dryRun {
		r.Printf("\n--- DRY RUN: %s ---\n", filename)
		r.Printf("%s\n", finalContent)
		r.Printf("--- END DRY RUN ---\n")
		return nil
	}

	// Prepare directories
	if parents {
		dir := filepath.Dir(filename)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directories: %w", err)
		}
	}

	// Check for diff if overwriting
	var oldContent string
	exists := false
	if !appendMode {
		data, err := os.ReadFile(filename)
		if err == nil {
			oldContent = string(data)
			exists = true
		}
	}

	// Write to file
	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	f, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		return fmt.Errorf("opening file for writing: %w", err)
	}
	defer f.Close()

	n, err := f.WriteString(finalContent)
	if err != nil {
		return fmt.Errorf("writing to file: %w", err)
	}

	// Success message
	action := "Written"
	if appendMode {
		action = "Appended"
	}
	color := "green"
	if appendMode {
		color = "cyan"
	}

	r.Printf("%s %d bytes to %s\n", r.Display.colorizedText(action, color), n, r.Display.colorizedText(filename, "cyan"))

	// Show diff
	if exists && showDiff && oldContent != finalContent {
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(oldContent),
			B:        difflib.SplitLines(finalContent),
			FromFile: "Old",
			ToFile:   "New",
			Context:  1,
		}
		text, _ := difflib.GetUnifiedDiffString(diff)
		r.Printf("\nChanges:\n%s\n", text)
	}

	if edit {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}
		cmd := exec.Command(editor, filename)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	return nil
}

// findHistoryBlock identifies the start and end indices of the multiline block containing targetIdx.
func findHistoryBlock(history []string, targetIdx int) (start, end int) {
	if targetIdx < 0 || targetIdx >= len(history) {
		return targetIdx, targetIdx
	}

	type block struct {
		start int
		end   int
	}
	var blocks []block

	var currentStart int = -1
	var heredocDelim string

	for i := 0; i < len(history); i++ {
		line := history[i]

		if currentStart == -1 {
			// Not currently in a block
			trimmed := strings.TrimRight(line, " \t")

			// Check for heredoc start
			if idx := strings.LastIndex(line, "<<"); idx != -1 {
				delim := strings.TrimSpace(line[idx+2:])
				if delim != "" && !strings.ContainsAny(delim, " \t\"'") {
					currentStart = i
					heredocDelim = delim
					continue
				}
			}

			// Check for slash continuation
			if strings.HasSuffix(trimmed, "\\") {
				currentStart = i
				continue
			}

			// Single line block
			blocks = append(blocks, block{i, i})
		} else {
			// Currently in a block
			if heredocDelim != "" {
				// Heredoc mode
				if strings.TrimSpace(line) == heredocDelim {
					blocks = append(blocks, block{currentStart, i})
					currentStart = -1
					heredocDelim = ""
				}
				continue
			}

			// Slash continuation mode
			trimmed := strings.TrimRight(line, " \t")
			if !strings.HasSuffix(trimmed, "\\") {
				blocks = append(blocks, block{currentStart, i})
				currentStart = -1
			}
		}
	}

	// Handle trailing open block
	if currentStart != -1 {
		blocks = append(blocks, block{currentStart, len(history) - 1})
	}

	// Find which block contains targetIdx
	for _, b := range blocks {
		if targetIdx >= b.start && targetIdx <= b.end {
			return b.start, b.end
		}
	}

	return targetIdx, targetIdx
}

// historyEntry is used by the :history command for JSON serialization.
type historyEntry struct {
	Index int    `json:"index"`
	Cmd   string `json:"command"`
}

// historyClear clears the history file after user confirmation.
func (r *REPL) historyClear() error {
	data, err := os.ReadFile(r.config.HistoryFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading history file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}

	if count == 0 {
		r.Printf("History is already empty.\n")
		return nil
	}

	warning := r.Display.colorizedText(
		fmt.Sprintf("⚠  This will permanently clear %d entries from %s.", count, r.config.HistoryFile),
		"yellow",
	)
	r.Printf("%s\n", warning)
	r.Printf("   Are you sure? (y/N): ")

	var answer string
	_, _ = fmt.Scanln(&answer)
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" && answer != "yes" {
		r.Printf("Cancelled.\n")
		return nil
	}

	if err := os.Truncate(r.config.HistoryFile, 0); err != nil {
		return fmt.Errorf("clearing history file: %w", err)
	}

	r.Printf("%s\n", r.Display.colorizedText("✓ History cleared.", "green"))
	return nil
}

// historyExport writes history entries to a file in JSONL or plain text format.
func (r *REPL) historyExport(filename string, entries []historyEntry, jsonFormat bool) error {
	var content strings.Builder

	isJSONL := strings.HasSuffix(strings.ToLower(filename), ".jsonl")

	if isJSONL || jsonFormat {
		// JSONL format: one JSON object per line
		for _, e := range entries {
			line, err := json.Marshal(e)
			if err != nil {
				return fmt.Errorf("marshalling entry: %w", err)
			}
			content.Write(line)
			content.WriteByte('\n')
		}
	} else {
		// Plain text: one command per line
		for _, e := range entries {
			content.WriteString(e.Cmd)
			content.WriteByte('\n')
		}
	}

	if err := os.WriteFile(filename, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("writing export file: %w", err)
	}

	formatLabel := "plain text"
	if isJSONL || jsonFormat {
		formatLabel = "JSONL"
	}

	r.Printf("%s\n", r.Display.colorizedText(
		fmt.Sprintf("✓ Exported %d entries to %s (%s)", len(entries), filename, formatLabel),
		"green",
	))
	return nil
}

// highlightSearchTerm highlights all case-insensitive occurrences of term in text.
func (r *REPL) highlightSearchTerm(text, term string) string {
	lowerText := strings.ToLower(text)
	lowerTerm := strings.ToLower(term)
	termLen := len(term)

	var result strings.Builder
	lastEnd := 0

	for {
		idx := strings.Index(lowerText[lastEnd:], lowerTerm)
		if idx == -1 {
			break
		}
		matchStart := lastEnd + idx
		matchEnd := matchStart + termLen

		// Write text before match
		result.WriteString(text[lastEnd:matchStart])
		// Write highlighted match (bold + yellow)
		result.WriteString(r.Display.colorizedText(text[matchStart:matchEnd], "yellow"))

		lastEnd = matchEnd
	}

	// Write remaining text
	result.WriteString(text[lastEnd:])
	return result.String()
}
