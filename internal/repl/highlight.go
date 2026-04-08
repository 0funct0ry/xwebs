package repl

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// Group 1: Go Templates {{ ... }}
	// Group 2: JSON Keys "key":
	// Group 3: JSON Strings "value"
	// Group 4: JSON Numbers
	// Group 5: JSON Booleans/Null
	keyRegex    = `"(?:[^"\\]|\\.)*"\s*:`
	strRegex    = `"(?:[^"\\]|\\.)*"`
	tmplRegex   = `\{\{.*?\}\}`
	numRegex    = `\b\d+\b`
	boolRegex   = `\b(?:true|false|null)\b`
	highlightRegex = regexp.MustCompile(fmt.Sprintf("(%s)|(%s)|(%s)|(%s)|(%s)", tmplRegex, keyRegex, strRegex, numRegex, boolRegex))

	// Standard ANSI colors used in the REPL
	colorReset   = "\033[0m"
	colorKey     = "\033[36m" // Cyan
	colorString  = "\033[32m" // Green
	colorNumber  = "\033[33m" // Yellow
	colorBool    = "\033[31m" // Red
	colorTmpl    = "\033[35m" // Magenta

	// Log specific colors
	colorError   = "\033[1;31m" // Bold Red
	colorWarn    = "\033[1;33m" // Bold Yellow
	colorInfo    = "\033[1;36m" // Bold Cyan
	colorDebug   = "\033[1;34m" // Bold Blue
	colorDim     = "\033[2m"    // Dim
)

// Highlighter implements the readline.Painter interface to provide
// real-time syntax highlighting for JSON and template expressions.
type Highlighter struct {
	display *FormattingState
}

// NewHighlighter creates a new Highlighter instance.
func NewHighlighter(display *FormattingState) *Highlighter {
	return &Highlighter{
		display: display,
	}
}

// Paint transforms the input line into a colorized version for display.
// This is called by readline on every keystroke.
func (h *Highlighter) Paint(line []rune, pos int) []rune {
	if h.display == nil || !h.display.isColorEnabled() {
		return line
	}

	str := string(line)
	if str == "" {
		return line
	}

	// Identify the start of the payload for highlighting.
	// We want to highlight arguments of :send, :sendj, :sendt, or bare text.
	highlighted := ""
	prefix := ""
	payload := str

	if strings.HasPrefix(str, ":") {
		parts := strings.SplitN(str, " ", 2)
		cmd := parts[0]
		switch cmd {
		case ":send", ":sendj", ":sendt", ":assert", ":filter":
			if len(parts) > 1 {
				prefix = parts[0] + " "
				payload = parts[1]
			}
		default:
			// For other commands, we only highlight if they contain templates
			// (e.g. :set key {{ val }})
			// We'll treat the whole line as a potential template field but NOT JSON.
			return []rune(h.highlightTmplOnly(str))
		}
	}

	highlighted = prefix + h.highlightPayload(payload)
	return []rune(highlighted)
}

// highlightPayload applies both JSON and Template highlighting.
func (h *Highlighter) highlightPayload(payload string) string {
	return highlightRegex.ReplaceAllStringFunc(payload, func(m string) string {
		if strings.HasPrefix(m, "{{") {
			return colorTmpl + m + colorReset
		}
		if strings.HasSuffix(m, ":") {
			return colorKey + m + colorReset
		}
		if strings.HasPrefix(m, "\"") {
			return colorString + m + colorReset
		}
		// Check for number (digit at start)
		if m[0] >= '0' && m[0] <= '9' {
			return colorNumber + m + colorReset
		}
		// Booleans / null
		return colorBool + m + colorReset
	})
}

// highlightTmplOnly specifically focuses on {{ ... }} without interpreting JSON tokens.
// This is used for general command arguments where JSON highlighting might be confusing.
func (h *Highlighter) highlightTmplOnly(str string) string {
	tmplRegex := regexp.MustCompile(`\{\{.*?\}\}`)
	return tmplRegex.ReplaceAllStringFunc(str, func(m string) string {
		return colorTmpl + m + colorReset
	})
}

// HighlightLine applies syntax highlighting to a single line based on file type.
func (s *FormattingState) HighlightLine(filename string, line string) string {
	if !s.isColorEnabled() {
		return line
	}

	ext := strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(filename, "."), "."))
	if idx := strings.LastIndex(filename, "."); idx != -1 {
		ext = strings.ToLower(filename[idx+1:])
	}

	switch ext {
	case "json":
		return s.highlightOutputJSON(line)
	case "yaml", "yml":
		return s.highlightYAML(line)
	case "log":
		return s.highlightLog(line)
	default:
		// Try log highlighting for common text files/outputs
		return s.highlightLog(line)
	}
}

func (s *FormattingState) highlightYAML(line string) string {
	// Simple YAML key highlighting
	keyRegex := regexp.MustCompile(`^([\s-]*)([\w.-]+)(:)`)
	line = keyRegex.ReplaceAllStringFunc(line, func(m string) string {
		parts := keyRegex.FindStringSubmatch(m)
		if len(parts) == 4 {
			return parts[1] + colorKey + parts[2] + colorReset + parts[3]
		}
		return m
	})

	// Highlight templates if present
	return s.highlightTmplOnly(line)
}

func (s *FormattingState) highlightLog(line string) string {
	// Highlight log levels
	levels := []struct {
		regex *regexp.Regexp
		color string
	}{
		{regexp.MustCompile(`(?i)\b(ERROR|FATAL|FAIL)\b`), colorError},
		{regexp.MustCompile(`(?i)\b(WARN|WARNING)\b`), colorWarn},
		{regexp.MustCompile(`(?i)\b(INFO)\b`), colorInfo},
		{regexp.MustCompile(`(?i)\b(DEBUG|TRACE)\b`), colorDebug},
	}

	for _, l := range levels {
		line = l.regex.ReplaceAllStringFunc(line, func(m string) string {
			return l.color + m + colorReset
		})
	}

	// Dim timestamps (simple heuristic: starts with date-like pattern)
	tsRegex := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T?\d{2}:\d{2}:\d{2})`)
	line = tsRegex.ReplaceAllStringFunc(line, func(m string) string {
		return colorDim + m + colorReset
	})

	return s.highlightTmplOnly(line)
}

func (s *FormattingState) highlightOutputJSON(data string) string {
	// We pass a dummy highlighter or just call the logic directly
	h := &Highlighter{display: s}
	return h.highlightPayload(data)
}

func (s *FormattingState) highlightTmplOnly(data string) string {
	h := &Highlighter{display: s}
	return h.highlightTmplOnly(data)
}
