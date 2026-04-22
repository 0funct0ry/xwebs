package repl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHeditBlockSelection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "hedit-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	historyFile := filepath.Join(tmpDir, "history")
	historyContent := strings.Join([]string{
		"line 1",
		"line 2 \\",
		"line 3",
		"line 4",
		":send <<EOF",
		"  content 1",
		"  content 2",
		"EOF",
		"line 9",
		":hedit",
	}, "\n") + "\n"

	if err := os.WriteFile(historyFile, []byte(historyContent), 0644); err != nil {
		t.Fatalf("failed to write history file: %v", err)
	}

	tests := []struct {
		name      string
		num       int
		wantLines []string
	}{
		{"default (skip :hedit, item 9)", 0, []string{"line 9"}},
		{"item 1 (single)", 1, []string{"line 1"}},
		{"item 2 (slash block)", 2, []string{"line 2 \\", "line 3"}},
		{"item 3 (slash block)", 3, []string{"line 2 \\", "line 3"}},
		{"item 5 (heredoc block start)", 5, []string{":send <<EOF", "  content 1", "  content 2", "EOF"}},
		{"item 6 (heredoc content)", 6, []string{":send <<EOF", "  content 1", "  content 2", "EOF"}},
		{"item 8 (heredoc end)", 8, []string{":send <<EOF", "  content 1", "  content 2", "EOF"}},
		{"item 9 (single)", 9, []string{"line 9"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, _ := os.ReadFile(historyFile)
			lines := strings.Split(string(data), "\n")
			var history []string
			for _, l := range lines {
				if strings.TrimSpace(l) != "" {
					history = append(history, l)
				}
			}

			var targetIdx int = -1
			if tt.num > 0 {
				targetIdx = tt.num - 1
			} else {
				// Search backwards for the first item that is NOT :hedit
				for i := len(history) - 1; i >= 0; i-- {
					if !strings.HasPrefix(strings.TrimSpace(history[i]), ":hedit") {
						targetIdx = i
						break
					}
				}
			}

			var gotLines []string
			if targetIdx >= 0 {
				startBlock, endBlock := findHistoryBlockMock(history, targetIdx)
				gotLines = history[startBlock : endBlock+1]
			}

			if strings.Join(gotLines, "\n") != strings.Join(tt.wantLines, "\n") {
				t.Errorf("got %v, want %v", gotLines, tt.wantLines)
			}
		})
	}
}

// findHistoryBlockMock is a copy of the logic in command.go for testing purposes in the same package (internal/repl)
func findHistoryBlockMock(history []string, targetIdx int) (start, end int) {
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
			trimmed := strings.TrimRight(line, " \t")
			if idx := strings.LastIndex(line, "<<"); idx != -1 {
				delim := strings.TrimSpace(line[idx+2:])
				if delim != "" && !strings.ContainsAny(delim, " \t\"'") {
					currentStart = i
					heredocDelim = delim
					continue
				}
			}
			if strings.HasSuffix(trimmed, "\\") {
				currentStart = i
				continue
			}
			blocks = append(blocks, block{i, i})
		} else {
			if heredocDelim != "" {
				if strings.TrimSpace(line) == heredocDelim {
					blocks = append(blocks, block{currentStart, i})
					currentStart = -1
					heredocDelim = ""
				}
				continue
			}
			trimmed := strings.TrimRight(line, " \t")
			if !strings.HasSuffix(trimmed, "\\") {
				blocks = append(blocks, block{currentStart, i})
				currentStart = -1
			}
		}
	}
	if currentStart != -1 {
		blocks = append(blocks, block{currentStart, len(history) - 1})
	}

	for _, b := range blocks {
		if targetIdx >= b.start && targetIdx <= b.end {
			return b.start, b.end
		}
	}
	return targetIdx, targetIdx
}

func TestHistoryGroupingOutput(t *testing.T) {
	history := []string{
		"line 1",
		"line 2 \\",
		"line 3",
		"line 4",
	}

	outputs := []string{}
	for i := 0; i < len(history); i++ {
		line := history[i]
		isContinuation := false
		if i > 0 {
			prev := strings.TrimRight(history[i-1], " \t")
			if strings.HasSuffix(prev, "\\") {
				isContinuation = true
			}
		}

		if isContinuation {
			outputs = append(outputs, "        "+line)
		} else {
			outputs = append(outputs, fmt.Sprintf("  %4d  %s", i+1, line))
		}
	}

	if outputs[2] != "        line 3" {
		t.Errorf("line 3 should be indented as continuation, got %q", outputs[2])
	}
}
