package template

import (
	"fmt"
	"strings"

	"github.com/spf13/cast"
)

// registerColorFuncs adds ANSI color and style functions to the engine's function map.
func (e *Engine) registerColorFuncs() {
	// Base color function
	e.funcs["color"] = func(color, s interface{}) string {
		val := cast.ToString(s)
		if !e.ColorsEnabled {
			return val
		}
		c := cast.ToString(color)
		code := getColorCode(c)
		if code == "" {
			return val
		}
		return fmt.Sprintf("\033[%sm%s\033[0m", code, val)
	}

	// Foreground colors
	colors := map[string]string{
		"black":   "30",
		"red":     "31",
		"green":   "32",
		"yellow":  "33",
		"blue":    "34",
		"magenta": "35",
		"cyan":    "36",
		"white":   "37",
		"grey":    "90",
		"dim":     "2", // Actually a style but often used as color
	}

	for name, code := range colors {
		n := name // capture for closure
		c := code
		e.funcs[n] = func(s interface{}) string {
			val := cast.ToString(s)
			if !e.ColorsEnabled {
				return val
			}
			return fmt.Sprintf("\033[%sm%s\033[0m", c, val)
		}
	}

	// Styles
	styles := map[string]string{
		"bold":      "1",
		"faint":     "2",
		"italic":    "3",
		"underline": "4",
		"inverse":   "7",
	}

	for name, code := range styles {
		n := name
		c := code
		e.funcs[n] = func(s interface{}) string {
			val := cast.ToString(s)
			if !e.ColorsEnabled {
				return val
			}
			return fmt.Sprintf("\033[%sm%s\033[0m", c, val)
		}
	}

	e.funcs["reset"] = func() string {
		if !e.ColorsEnabled {
			return ""
		}
		return "\033[0m"
	}
}

func getColorCode(name string) string {
	switch strings.ToLower(name) {
	case "black":
		return "30"
	case "red":
		return "31"
	case "green":
		return "32"
	case "yellow":
		return "33"
	case "blue":
		return "34"
	case "magenta":
		return "35"
	case "cyan":
		return "36"
	case "white":
		return "37"
	case "grey":
		return "90"
	case "bold":
		return "1"
	case "dim", "faint":
		return "2"
	case "italic":
		return "3"
	case "underline":
		return "4"
	case "inverse":
		return "7"
	default:
		// Check if it's already a numeric code
		return name
	}
}
