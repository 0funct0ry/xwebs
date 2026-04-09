package repl

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/itchyny/gojq"
)

// DisplayFormat represents the available message formatting modes.
type DisplayFormat string

const (
	// FormatRaw displays the message as-is.
	FormatRaw DisplayFormat = "raw"
	// FormatJSON pretty-prints the message as JSON.
	FormatJSON DisplayFormat = "json"
	// FormatHex displays a hex dump of the message.
	FormatHex DisplayFormat = "hex"
	// FormatTemplate renders a custom Go template.
	FormatTemplate DisplayFormat = "template"
	// FormatJSONL displays the full message metadata and data as a single JSON line.
	FormatJSONL DisplayFormat = "jsonl"
)

// FormattingState contains the current REPL display settings.
type FormattingState struct {
	Format       DisplayFormat
	Template     string
	Filter       string
	IsRegex      bool
	Quiet        bool
	Verbose      bool
	Timestamps   bool
	TimestampUTC bool
	Color        string // "on", "off", "auto"
	NoIndicators bool   // Suppress direction symbols and prefixes for clean output
	IsTTY        bool   // Whether the output is a TTY (for color auto-detection)

	// Compiled state
	jqFilter    *gojq.Query
	regexFilter *regexp.Regexp
}

// NewFormattingState returns a state with default values.
func NewFormattingState() *FormattingState {
	return &FormattingState{
		Format: FormatRaw,
		Color:  "auto",
	}
}

// SetFilter updates and compiles the current filter.
func (s *FormattingState) SetFilter(expr string) error {
	if expr == "" || expr == "off" {
		s.Filter = ""
		s.jqFilter = nil
		s.regexFilter = nil
		s.IsRegex = false
		return nil
	}

	// Determine if it's a regex (enclosed in / /)
	if strings.HasPrefix(expr, "/") && strings.HasSuffix(expr, "/") && len(expr) > 2 {
		re, err := regexp.Compile(expr[1 : len(expr)-1])
		if err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}
		s.Filter = expr
		s.regexFilter = re
		s.jqFilter = nil
		s.IsRegex = true
		return nil
	}

	// Otherwise treat as JQ
	query, err := gojq.Parse(expr)
	if err != nil {
		return fmt.Errorf("invalid jq expression: %w", err)
	}

	s.Filter = expr
	s.jqFilter = query
	s.regexFilter = nil
	s.IsRegex = false
	return nil
}

// FormatMessage applies formatting and filtering to a message.
// Returns the formatted string and a boolean indicating if it should be displayed.
func (s *FormattingState) FormatMessage(msg *ws.Message, vars map[string]interface{}, engine *template.Engine) (string, bool) {
	// 1. Filtering (received messages only)
	if msg.Metadata.Direction == "received" && s.Filter != "" {
		if !s.matchesFilter(msg) {
			return "", false
		}
	}
	if s.Format == FormatJSONL {
		return s.formatBody(msg, vars, engine), true
	}

	if s.NoIndicators {
		// Clean output mode: received messages only by default
		if msg.Metadata.Direction == "sent" && !s.Verbose {
			return "", false
		}

		body := s.formatBody(msg, vars, engine)
		// Specifically for NoIndicators (pipes), if format is JSON and unmarshal failed,
		// formatBody returns empty string or prefix. We'll skip if it's empty.
		if body == "" {
			return "", false
		}
		return body, true
	}

	var sb strings.Builder

	// 2. Timestamps
	if s.Timestamps {
		ts := msg.Metadata.Timestamp
		if s.TimestampUTC {
			ts = ts.UTC()
		}
		sb.WriteString(ts.Format("2006-01-02T15:04:05.000Z07:00 "))
	}

	// 3. Direction Indicator
	if msg.Metadata.Direction == "sent" {
		sb.WriteString("⬆ ")
	} else {
		sb.WriteString("⬇ ")
	}

	// 4. Verbose Metadata
	if s.Verbose {
		typeStr := "unknown"
		switch msg.Type {
		case ws.TextMessage:
			typeStr = "text"
		case ws.BinaryMessage:
			typeStr = "binary"
		case ws.PingMessage:
			typeStr = "ping"
		case ws.PongMessage:
			typeStr = "pong"
		}
		sb.WriteString(fmt.Sprintf("[%s len=%d compress=%v] ", typeStr, msg.Metadata.Length, msg.Metadata.Compressed))
	}

	// 5. Body Formatting
	body := s.formatBody(msg, vars, engine)
	sb.WriteString(body)

	return sb.String(), true
}

func (s *FormattingState) matchesFilter(msg *ws.Message) bool {
	if s.IsRegex {
		return s.regexFilter.Match(msg.Data)
	}

	if s.jqFilter == nil {
		return true
	}

	var data interface{}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return false // Non-JSON doesn't match JQ
	}

	iter := s.jqFilter.Run(data)
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := v.(error); ok {
			// Runtime error, ignore message
			_ = err
			return false
		}
		if v != nil && v != false {
			return true
		}
	}
	return false
}

func (s *FormattingState) formatBody(msg *ws.Message, vars map[string]interface{}, engine *template.Engine) string {
	// Binary frames always use hex dump
	if msg.Type == ws.BinaryMessage {
		return s.hexDump(msg.Data)
	}

	format := s.Format
	switch format {
	case FormatJSON:
		var data interface{}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			if s.NoIndicators {
				// Avoid polluting JSON stream with "not JSON" errors
				return ""
			}
			return s.colorizedText("[not JSON] ", "dim") + string(msg.Data)
		}
		pretty, _ := json.MarshalIndent(data, "", "  ")
		if s.isColorEnabled() {
			return s.highlightOutputJSON(string(pretty))
		}
		return string(pretty)

	case FormatHex:
		return s.hexDump(msg.Data)

	case FormatTemplate:
		if engine == nil {
			return string(msg.Data)
		}

		typeStr := "text"
		switch msg.Type {
		case ws.BinaryMessage:
			typeStr = "binary"
		case ws.PingMessage:
			typeStr = "ping"
		case ws.PongMessage:
			typeStr = "pong"
		}

		tmplCtx := template.NewContext()
		tmplCtx.Session = vars
		tmplCtx.Msg = &template.MessageContext{
			Type:      typeStr,
			Data:      string(msg.Data),
			Raw:       msg.Data,
			Length:    msg.Metadata.Length,
			Timestamp: msg.Metadata.Timestamp,
		}
		tmplCtx.Conn = &template.ConnectionContext{
			URL: msg.Metadata.URL,
		}

		// Populating top-level convenience fields
		tmplCtx.Message = string(msg.Data)
		tmplCtx.MessageBytes = msg.Data
		tmplCtx.MessageLen = msg.Metadata.Length
		tmplCtx.MessageType = typeStr
		tmplCtx.MessageIndex = msg.Metadata.MessageIndex
		tmplCtx.Timestamp = msg.Metadata.Timestamp
		tmplCtx.Direction = msg.Metadata.Direction
		// Direction and ID can be used in session if needed, or I could extend TemplateContext
		// For now, let's just use the basic ones.

		res, err := engine.Execute("format", s.Template, tmplCtx)
		if err != nil {
			return s.colorizedText(fmt.Sprintf("[template error: %v] ", err), "red") + string(msg.Data)
		}
		return res

	case FormatJSONL:
		output, _ := json.Marshal(msg)
		return string(output)

	default:
		if s.isColorEnabled() {
			return s.highlightOutputJSON(string(msg.Data))
		}
		return string(msg.Data)
	}
}

func (s *FormattingState) hexDump(data []byte) string {
	var sb strings.Builder
	for i := 0; i < len(data); i += 16 {
		// Offset
		sb.WriteString(fmt.Sprintf("%08x  ", i))

		// Hex bytes
		for j := 0; j < 16; j++ {
			if i+j < len(data) {
				sb.WriteString(fmt.Sprintf("%02x ", data[i+j]))
			} else {
				sb.WriteString("   ")
			}
			if j == 7 {
				sb.WriteString(" ")
			}
		}

		// ASCII sidebar
		sb.WriteString(" |")
		for j := 0; j < 16; j++ {
			if i+j < len(data) {
				b := data[i+j]
				if b >= 32 && b <= 126 {
					sb.WriteByte(b)
				} else {
					sb.WriteByte('.')
				}
			}
		}
		sb.WriteString("|\n")
	}
	return strings.TrimSuffix(sb.String(), "\n")
}

func (s *FormattingState) isColorEnabled() bool {
	if s.Color == "on" {
		return true
	}
	if s.Color == "off" {
		return false
	}
	// "auto" mode: follow TTY status
	return s.IsTTY
}

func (s *FormattingState) colorizedText(text string, color string) string {
	if !s.isColorEnabled() {
		return text
	}
	switch color {
	case "dim":
		return "\033[2m" + text + "\033[0m"
	case "cyan":
		return "\033[36m" + text + "\033[0m"
	case "green":
		return "\033[32m" + text + "\033[0m"
	case "yellow":
		return "\033[33m" + text + "\033[0m"
	case "red":
		return "\033[31m" + text + "\033[0m"
	default:
		return text
	}
}
