package repl

import (
	"os"
	"path/filepath"
	"strings"
)

// DoContext implements the logic for context-aware completion.
// It's called by REPL.Do to delegate the actual work.
func (r *REPL) DoContext(line []rune, pos int) (newLine [][]rune, length int) {
	currentLine := string(line[:pos])

	// 1. Identify if we are in a template block {{ ... }}
	if isInsideTemplate(currentLine) {
		return r.completeTemplate(currentLine)
	}

	// 2. Identify if we are at the start of the line or in a command
	if isCommandStart(currentLine) {
		return r.completeCommands(currentLine)
	}

	// 3. Identify if we are in an argument
	return r.completeArguments(currentLine)
}

// isInsideTemplate checks if the cursor is currently inside a template block.
func isInsideTemplate(line string) bool {
	lastOpen := strings.LastIndex(line, "{{")
	lastClose := strings.LastIndex(line, "}}")
	return lastOpen > lastClose
}

// isCommandStart checks if the cursor is at the first word and it starts with ':'.
func isCommandStart(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	if !strings.HasPrefix(trimmed, ":") {
		return false
	}
	// If there's a space after the command, it's an argument context
	firstSpace := strings.Index(trimmed, " ")
	return firstSpace == -1
}

// completeTemplate suggests template functions.
func (r *REPL) completeTemplate(line string) ([][]rune, int) {
	lastOpen := strings.LastIndex(line, "{{")
	prefix := line[lastOpen+2:]
	prefix = strings.TrimLeft(prefix, " ")

	// Find the last word in the template block
	parts := strings.Fields(prefix)
	var currentWord string
	if len(parts) > 0 && !strings.HasSuffix(prefix, " ") {
		currentWord = parts[len(parts)-1]
	}

	var suggestions [][]rune

	// 1. Template functions (no dot prefix)
	funcs := r.GetCompletionData("template_funcs")
	for _, f := range funcs {
		if strings.HasPrefix(strings.ToLower(f), strings.ToLower(currentWord)) {
			suggestions = append(suggestions, []rune(f[len(currentWord):]))
		}
	}

	// 2. Context variables (with dot prefix or matching current word starting with dot)
	ctxVars := []string{".Conn", ".Msg", ".Handler", ".Server", ".Session", ".Env"}
	for _, v := range ctxVars {
		if strings.HasPrefix(strings.ToLower(v), strings.ToLower(currentWord)) {
			suggestions = append(suggestions, []rune(v[len(currentWord):]))
		}
	}

	// 3. JSON keys (usually used as dots if they are top-level or part of a map)
	jsonKeys := r.GetCompletionData("json")
	for _, k := range jsonKeys {
		// Suggest both with and without dot depending on currentWord
		dotK := "." + k
		if strings.HasPrefix(strings.ToLower(dotK), strings.ToLower(currentWord)) {
			suggestions = append(suggestions, []rune(dotK[len(currentWord):]))
		} else if strings.HasPrefix(strings.ToLower(k), strings.ToLower(currentWord)) {
			suggestions = append(suggestions, []rune(k[len(currentWord):]))
		}
	}

	// 4. Session variables (usually accessed via .Session.VAR)
	vars := r.GetVars()
	for v := range vars {
		if strings.HasPrefix(strings.ToLower(v), strings.ToLower(currentWord)) {
			suggestions = append(suggestions, []rune(v[len(currentWord):]))
		}
		// Also suggest as part of .Session.
		sessV := ".Session." + v
		if strings.HasPrefix(strings.ToLower(sessV), strings.ToLower(currentWord)) {
			suggestions = append(suggestions, []rune(sessV[len(currentWord):]))
		}
	}

	return suggestions, len(currentWord)
}

// completeCommands suggests command names and aliases.
func (r *REPL) completeCommands(line string) ([][]rune, int) {
	prefix := strings.TrimLeft(line, " ")
	prefix = strings.TrimPrefix(prefix, ":")

	var suggestions [][]rune
	seen := make(map[string]bool)

	// Command names
	for name, cmd := range r.commands {
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			if !seen[name] && cmd.IsVisible(r) {
				suggestions = append(suggestions, []rune(name[len(prefix):]))
				seen[name] = true
			}
		}
	}

	// Aliases
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

// completeArguments suggests arguments based on the command.
func (r *REPL) completeArguments(line string) ([][]rune, int) {
	trimmed := strings.TrimLeft(line, " ")
	if !strings.HasPrefix(trimmed, ":") {
		return nil, 0
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return nil, 0
	}

	cmdName := strings.TrimPrefix(parts[0], ":")
	if alias, ok := r.aliases[cmdName]; ok {
		cmdName = alias
	}

	// Find the current word (the one being completed)
	var currentWord string
	if !strings.HasSuffix(line, " ") {
		currentWord = parts[len(parts)-1]
	}

	var suggestions [][]rune

	switch cmdName {
	case "connect":
		// Suggest bookmarks and aliases
		bookmarks := r.GetCompletionData("bookmarks")
		for _, b := range bookmarks {
			if strings.HasPrefix(strings.ToLower(b), strings.ToLower(currentWord)) {
				suggestions = append(suggestions, []rune(b[len(currentWord):]))
			}
		}
	case "sendj", "sendt":
		// Suggest JSON keys
		jsonKeys := r.GetCompletionData("json")
		for _, k := range jsonKeys {
			if strings.HasPrefix(strings.ToLower(k), strings.ToLower(currentWord)) {
				suggestions = append(suggestions, []rune(k[len(currentWord):]))
			}
		}
	case "get", "set":
		// Suggest session variables
		vars := r.GetVars()
		for v := range vars {
			if strings.HasPrefix(strings.ToLower(v), strings.ToLower(currentWord)) {
				suggestions = append(suggestions, []rune(v[len(currentWord):]))
			}
		}
	case "write":
		// Suggest flags if current word starts with -
		if strings.HasPrefix(currentWord, "-") {
			flags := []string{"--append", "-a", "--json", "--dry-run", "-n", "--parents", "-p", "--diff", "--edit", "--last-message", "--last-response", "--current-handlers", "--clipboard"}
			for _, f := range flags {
				if strings.HasPrefix(f, currentWord) {
					suggestions = append(suggestions, []rune(f[len(currentWord):]))
				}
			}
		}
	case "history":
		// Suggest flags if current word starts with -
		if strings.HasPrefix(currentWord, "-") {
			flags := []string{"-n", "--number", "-c", "--clear", "-s", "--search",
				"-f", "--filter", "-e", "--export", "--unique", "-r", "--reverse", "--json"}
			for _, f := range flags {
				if strings.HasPrefix(f, currentWord) {
					suggestions = append(suggestions, []rune(f[len(currentWord):]))
				}
			}
		}
	case "kick", "client", "send":
		// Suggest client IDs if in server mode
		if r.serverCtx != nil {
			clients := r.serverCtx.GetClients()
			for _, c := range clients {
				if strings.HasPrefix(strings.ToLower(c.ID), strings.ToLower(currentWord)) {
					suggestions = append(suggestions, []rune(c.ID[len(currentWord):]))
				}
			}
		}

	case "topic", "publish":
		// First non-flag argument — suggest topic names
		if r.serverCtx != nil {
			for _, t := range r.serverCtx.GetTopics() {
				if strings.HasPrefix(strings.ToLower(t.Name), strings.ToLower(currentWord)) {
					suggestions = append(suggestions, []rune(t.Name[len(currentWord):]))
				}
			}
		}

	case "subscribe":
		// First arg — client IDs; second arg — topic names
		if r.serverCtx != nil {
			argIdx := len(parts)
			if !strings.HasSuffix(line, " ") {
				argIdx-- // still typing the current word
			}
			if argIdx <= 1 {
				// Suggest client IDs
				for _, c := range r.serverCtx.GetClients() {
					if strings.HasPrefix(strings.ToLower(c.ID), strings.ToLower(currentWord)) {
						suggestions = append(suggestions, []rune(c.ID[len(currentWord):]))
					}
				}
			} else {
				// Suggest topic names
				for _, t := range r.serverCtx.GetTopics() {
					if strings.HasPrefix(strings.ToLower(t.Name), strings.ToLower(currentWord)) {
						suggestions = append(suggestions, []rune(t.Name[len(currentWord):]))
					}
				}
			}
		}

	case "unsubscribe":
		// First arg — client IDs; second arg — topic names or "--all"
		if r.serverCtx != nil {
			argIdx := len(parts)
			if !strings.HasSuffix(line, " ") {
				argIdx--
			}
			if argIdx <= 1 {
				// Suggest client IDs
				for _, c := range r.serverCtx.GetClients() {
					if strings.HasPrefix(strings.ToLower(c.ID), strings.ToLower(currentWord)) {
						suggestions = append(suggestions, []rune(c.ID[len(currentWord):]))
					}
				}
			} else {
				// Suggest "--all" flag and topic names
				allFlag := "--all"
				if strings.HasPrefix(allFlag, currentWord) {
					suggestions = append(suggestions, []rune(allFlag[len(currentWord):]))
				}
				for _, t := range r.serverCtx.GetTopics() {
					if strings.HasPrefix(strings.ToLower(t.Name), strings.ToLower(currentWord)) {
						suggestions = append(suggestions, []rune(t.Name[len(currentWord):]))
					}
				}
			}
		}
	case "handler":
		if r.serverCtx != nil {
			handlers := r.serverCtx.GetHandlers()
			subCommands := []string{"add", "delete", "edit", "rename", "reset", "save"}

			// If we are on the first argument
			if len(parts) == 1 || (len(parts) == 2 && !strings.HasSuffix(line, " ")) {
				// Suggest sub-commands
				for _, sc := range subCommands {
					if strings.HasPrefix(strings.ToLower(sc), strings.ToLower(currentWord)) {
						suggestions = append(suggestions, []rune(sc[len(currentWord):]))
					}
				}
				for _, h := range handlers {
					if strings.HasPrefix(strings.ToLower(h.Name), strings.ToLower(currentWord)) {
						suggestions = append(suggestions, []rune(h.Name[len(currentWord):]))
					}
				}
			} else if len(parts) >= 2 && parts[1] == "add" {
				// Suggest flags for :handler add
				if strings.HasPrefix(currentWord, "-") {
					flags := []string{"--name", "-n", "--match", "-m", "--match-type", "-t", "--priority", "-p", "--run", "-r", "--respond", "-R",
						"--builtin", "-B", "--topic", "--exclusive", "-e", "--sequential", "-s", "--rate-limit", "-l", "--debounce", "-d",
						"--file", "--path", "--content", "--mode", "--responses", "--loop", "--per-client", "--on-error", "--duration", "--max",
						"--code", "--reason", "--script", "--max-memory"}
					for _, f := range flags {
						if strings.HasPrefix(f, currentWord) {
							suggestions = append(suggestions, []rune(f[len(currentWord):]))
						}
					}
				}
			} else if len(parts) >= 2 && (parts[1] == "delete" || parts[1] == "edit" || parts[1] == "rename" || parts[1] == "reset") {
				// We are on the second argument of :handler delete <name>, :handler edit <name>, or :handler rename <old-name>
				// but only if we are typing or just after a space
				if len(parts) == 2 || (len(parts) == 3 && !strings.HasSuffix(line, " ")) {
					for _, h := range handlers {
						if strings.HasPrefix(strings.ToLower(h.Name), strings.ToLower(currentWord)) {
							suggestions = append(suggestions, []rune(h.Name[len(currentWord):]))
						}
					}
				}
			}
		} else {
			// Client mode completion for :handler
			subCommands := []string{"add", "delete", "edit", "reset", "save"}
			if len(parts) == 1 || (len(parts) == 2 && !strings.HasSuffix(line, " ")) {
				for _, sc := range subCommands {
					if strings.HasPrefix(strings.ToLower(sc), strings.ToLower(currentWord)) {
						suggestions = append(suggestions, []rune(sc[len(currentWord):]))
					}
				}
				if r.Handlers != nil {
					for _, h := range r.Handlers.Handlers() {
						if strings.HasPrefix(strings.ToLower(h.Name), strings.ToLower(currentWord)) {
							suggestions = append(suggestions, []rune(h.Name[len(currentWord):]))
						}
					}
				}
			} else if len(parts) >= 2 && parts[1] == "add" {
				// Suggest flags for :handler add
				if strings.HasPrefix(currentWord, "-") {
					flags := []string{"--name", "-n", "--match", "-m", "--match-type", "-t", "--priority", "-p", "--run", "-r", "--respond", "-R",
						"--builtin", "-B", "--topic", "--exclusive", "-e", "--sequential", "-s", "--rate-limit", "-l", "--debounce", "-d",
						"--file", "--path", "--content", "--mode", "--responses", "--loop", "--per-client", "--on-error", "--duration", "--max",
						"--code", "--reason", "--script", "--max-memory"}
					for _, f := range flags {
						if strings.HasPrefix(f, currentWord) {
							suggestions = append(suggestions, []rune(f[len(currentWord):]))
						}
					}
				}
			} else if len(parts) >= 2 && (parts[1] == "delete" || parts[1] == "edit" || parts[1] == "reset") {
				if r.Handlers != nil {
					if len(parts) == 2 || (len(parts) == 3 && !strings.HasSuffix(line, " ")) {
						for _, h := range r.Handlers.Handlers() {
							if strings.HasPrefix(strings.ToLower(h.Name), strings.ToLower(currentWord)) {
								suggestions = append(suggestions, []rune(h.Name[len(currentWord):]))
							}
						}
					}
				}
			}
		}
	case "enable", "disable":
		// Suggest handler names if in server mode
		if r.serverCtx != nil {
			handlers := r.serverCtx.GetHandlers()
			for _, h := range handlers {
				if strings.HasPrefix(strings.ToLower(h.Name), strings.ToLower(currentWord)) {
					suggestions = append(suggestions, []rune(h.Name[len(currentWord):]))
				}
			}
		}
	}

	if len(suggestions) == 0 || cmdName == "connect" {
		fileSuggestions, fileLen := r.completeFiles(currentWord)
		if len(fileSuggestions) > 0 {
			suggestions = append(suggestions, fileSuggestions...)
			return suggestions, fileLen
		}
	}

	return suggestions, len(currentWord)
}

// completeFiles suggests local files and directories.
func (r *REPL) completeFiles(prefix string) ([][]rune, int) {
	dir := "."
	filePrefix := prefix

	if strings.Contains(prefix, string(os.PathSeparator)) {
		dir = filepath.Dir(prefix)
		filePrefix = filepath.Base(prefix)
		if strings.HasSuffix(prefix, string(os.PathSeparator)) {
			dir = prefix
			filePrefix = ""
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, 0
	}

	var suggestions [][]rune
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(filePrefix)) {
			suffix := name[len(filePrefix):]
			if entry.IsDir() {
				suffix += string(os.PathSeparator)
			}
			suggestions = append(suggestions, []rune(suffix))
		}
	}

	return suggestions, len(filePrefix)
}
