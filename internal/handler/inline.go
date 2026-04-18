package handler

import (
	"fmt"
	"regexp"
	"strings"
)

var segmentRegex = regexp.MustCompile(`\s+::\s*`)

// ParseInlineHandler parses an inline handler string according to the xwebs spec:
// '<match> :: <action> [:: <action>] [:: <option>]'
// defaultRespond is the template to use if no respond: segment is provided.
// index is the 1-based index for the handler name (e.g. inline-1).
func ParseInlineHandler(hStr string, defaultRespond string, index int) (Handler, error) {
	segments := segmentRegex.Split(hStr, -1)
	if len(segments) == 0 || (len(segments) == 1 && segments[0] == "") {
		return Handler{}, fmt.Errorf("empty inline handler")
	}

	matchExpr := strings.TrimSpace(segments[0])
	if matchExpr == "" {
		return Handler{}, fmt.Errorf("missing match expression in inline handler (use '*' to match all messages)")
	}

	h := Handler{
		Name:  fmt.Sprintf("inline-%d", index),
		Match: AutoDetectMatcher(matchExpr),
	}

	// Default respond if provided
	if defaultRespond != "" {
		h.Respond = defaultRespond
	}

	// Parse subsequent segments
	for i := 1; i < len(segments); i++ {
		seg := strings.TrimSpace(segments[i])
		if seg == "" {
			continue
		}

		switch {
		case strings.HasPrefix(seg, "run:"):
			h.Run = strings.TrimSpace(strings.TrimPrefix(seg, "run:"))
		case strings.HasPrefix(seg, "respond:"):
			h.Respond = strings.TrimSpace(strings.TrimPrefix(seg, "respond:"))
		case strings.HasPrefix(seg, "timeout:"):
			h.Timeout = strings.TrimSpace(strings.TrimPrefix(seg, "timeout:"))
		case strings.HasPrefix(seg, "builtin:"):
			h.Builtin = strings.TrimSpace(strings.TrimPrefix(seg, "builtin:"))
		case strings.HasPrefix(seg, "topic:"):
			h.Topic = strings.TrimSpace(strings.TrimPrefix(seg, "topic:"))
		case seg == "exclusive":
			h.Exclusive = true
		default:
			// If it doesn't have a known prefix, it might be an invalid action or option.
			return Handler{}, fmt.Errorf("unknown segment in inline handler: %q", seg)
		}
	}

	// At least one action or respond is required (it can be provided via defaultRespond)
	if h.Run == "" && h.Respond == "" && h.Builtin == "" {
		return Handler{}, fmt.Errorf("inline handler %q must have at least one action (run: or respond:)", h.Name)
	}

	return h, nil
}

// AutoDetectMatcher creates a Matcher by detecting the type from the expression.
// Explicit prefixes (jq:, glob:, regex:, template:) override detection.
func AutoDetectMatcher(expr string) Matcher {
	original := expr

	// Check for explicit prefixes
	switch {
	case strings.HasPrefix(expr, "jq:"):
		return Matcher{Type: "jq", Pattern: strings.TrimPrefix(expr, "jq:")}
	case strings.HasPrefix(expr, "glob:"):
		return Matcher{Type: "glob", Pattern: strings.TrimPrefix(expr, "glob:")}
	case strings.HasPrefix(expr, "regex:"):
		return Matcher{Type: "regex", Pattern: strings.TrimPrefix(expr, "regex:")}
	case strings.HasPrefix(expr, "template:"):
		return Matcher{Type: "template", Pattern: strings.TrimPrefix(expr, "template:")}
	}

	// Match Shorthand
	if expr == "*" {
		return Matcher{Type: "glob", Pattern: "*"}
	}

	// Auto-detection rules
	// . -> jq
	// * or ? -> glob
	// ^ or ( -> regex
	// Anything else -> glob

	if strings.HasPrefix(expr, ".") {
		return Matcher{Type: "jq", Pattern: expr}
	}
	if strings.HasPrefix(expr, "^") || strings.HasPrefix(expr, "(") {
		return Matcher{Type: "regex", Pattern: expr}
	}
	if strings.ContainsAny(expr, "*?") {
		return Matcher{Type: "glob", Pattern: expr}
	}

	// Default to glob (Safe default — literal strings are valid globs)
	return Matcher{Type: "glob", Pattern: original}
}
