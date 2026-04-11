package repl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
)

func newTestREPLWithHistory(t *testing.T, historyLines []string) (*REPL, string, *bytes.Buffer) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "history-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	historyFile := filepath.Join(tmpDir, "history")
	content := strings.Join(historyLines, "\n") + "\n"
	if err := os.WriteFile(historyFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write history file: %v", err)
	}

	var buf bytes.Buffer
	r, err := New(ClientMode, &Config{
		Terminal:    true,
		HistoryFile: historyFile,
		Stdout:      &nopCloser{&buf},
	})
	if err != nil {
		t.Fatalf("failed to create REPL: %v", err)
	}
	r.TemplateEngine = template.New(false)
	r.RegisterCommonCommands()
	// Disable colors for predictable test output
	r.Display.Color = "off"

	return r, tmpDir, &buf
}

func TestHistoryDefault(t *testing.T) {
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = fmt.Sprintf(":send msg_item_%03d", i)
	}
	r, _, buf := newTestREPLWithHistory(t, lines)

	err := r.ExecuteCommand(context.Background(), ":history")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "last 20") {
		t.Errorf("expected 'last 20' in output, got:\n%s", output)
	}
	// First 10 should not be shown (items 000-009)
	if strings.Contains(output, "msg_item_000") {
		t.Errorf("should not contain first entry in default view")
	}
	if strings.Contains(output, "msg_item_009") {
		t.Errorf("should not contain 10th entry in default view")
	}
	// Last entry should be shown
	if !strings.Contains(output, "msg_item_029") {
		t.Errorf("should contain last entry")
	}
	// 11th entry (index 10) should be shown since we show last 20
	if !strings.Contains(output, "msg_item_010") {
		t.Errorf("should contain 11th entry (index 10)")
	}
}

func TestHistoryNumber(t *testing.T) {
	lines := []string{":send a", ":send b", ":send c", ":send d", ":send e"}
	r, _, buf := newTestREPLWithHistory(t, lines)

	err := r.ExecuteCommand(context.Background(), ":history -n 3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "last 3") {
		t.Errorf("expected 'last 3' in output, got:\n%s", output)
	}
	if strings.Contains(output, ":send a") {
		t.Errorf("should not contain :send a")
	}
	if strings.Contains(output, ":send b") {
		t.Errorf("should not contain :send b")
	}
	if !strings.Contains(output, ":send c") {
		t.Errorf("should contain :send c")
	}
	if !strings.Contains(output, ":send e") {
		t.Errorf("should contain :send e")
	}
}

func TestHistoryPositionalBackcompat(t *testing.T) {
	lines := []string{":send a", ":send b", ":send c", ":send d", ":send e"}
	r, _, buf := newTestREPLWithHistory(t, lines)

	err := r.ExecuteCommand(context.Background(), ":history 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "last 2") {
		t.Errorf("expected 'last 2' in output, got:\n%s", output)
	}
	if !strings.Contains(output, ":send d") {
		t.Errorf("should contain :send d")
	}
	if !strings.Contains(output, ":send e") {
		t.Errorf("should contain :send e")
	}
	if strings.Contains(output, ":send c") {
		t.Errorf("should not contain :send c with limit 2")
	}
}

func TestHistorySearch(t *testing.T) {
	lines := []string{":send hello", ":status", ":send deploy", ":connect ws://localhost", ":send deploy-v2"}
	r, _, buf := newTestREPLWithHistory(t, lines)

	err := r.ExecuteCommand(context.Background(), ":history -s deploy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "2 matching") {
		t.Errorf("expected '2 matching' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "deploy") {
		t.Errorf("should contain deploy entries")
	}
	if strings.Contains(output, ":status") {
		t.Errorf("should not contain :status")
	}
}

func TestHistorySearchCaseInsensitive(t *testing.T) {
	lines := []string{":send DEPLOY", ":send deploy", ":send Deploy", ":status"}
	r, _, buf := newTestREPLWithHistory(t, lines)

	err := r.ExecuteCommand(context.Background(), ":history -s deploy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "3 matching") {
		t.Errorf("expected '3 matching' in output, got:\n%s", output)
	}
}

func TestHistoryFilterGlob(t *testing.T) {
	lines := []string{":send hello", ":status", ":send world", ":set foo bar"}
	r, _, buf := newTestREPLWithHistory(t, lines)

	err := r.ExecuteCommand(context.Background(), ":history -f :send*")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "2 filtered") {
		t.Errorf("expected '2 filtered' in output, got:\n%s", output)
	}
	if strings.Contains(output, ":status") {
		t.Errorf("should not contain :status")
	}
}

func TestHistoryFilterRegex(t *testing.T) {
	lines := []string{":send hello", ":status", ":send world", ":set foo bar"}
	r, _, buf := newTestREPLWithHistory(t, lines)

	err := r.ExecuteCommand(context.Background(), ":history -f /^:send/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "2 filtered") {
		t.Errorf("expected '2 filtered' in output, got:\n%s", output)
	}
	if strings.Contains(output, ":status") {
		t.Errorf("should not contain :status")
	}
	if strings.Contains(output, ":set") {
		t.Errorf("should not contain :set")
	}
}

func TestHistoryUnique(t *testing.T) {
	lines := []string{":send hello", ":status", ":send hello", ":send world", ":status"}
	r, _, buf := newTestREPLWithHistory(t, lines)

	err := r.ExecuteCommand(context.Background(), ":history --unique -n 100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should only have 3 unique entries: :send hello, :send world, :status
	if !strings.Contains(output, "last 3") {
		t.Errorf("expected 'last 3' in output (3 unique commands), got:\n%s", output)
	}
	// Count occurrences of :send hello - should be 1
	count := strings.Count(output, ":send hello")
	if count != 1 {
		t.Errorf("expected exactly 1 occurrence of ':send hello', got %d", count)
	}
	// Count occurrences of :status - should be 1
	statusCount := strings.Count(output, ":status")
	if statusCount != 1 {
		t.Errorf("expected exactly 1 occurrence of ':status', got %d", statusCount)
	}
}

func TestHistoryReverse(t *testing.T) {
	lines := []string{":send a", ":send b", ":send c"}
	r, _, buf := newTestREPLWithHistory(t, lines)

	err := r.ExecuteCommand(context.Background(), ":history -r")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// In reverse, :send c should appear before :send a
	idxC := strings.Index(output, ":send c")
	idxA := strings.Index(output, ":send a")
	if idxC == -1 || idxA == -1 {
		t.Fatalf("expected both entries in output, got:\n%s", output)
	}
	if idxC > idxA {
		t.Errorf("expected :send c before :send a in reverse order")
	}
}

func TestHistoryJSON(t *testing.T) {
	lines := []string{":send hello", ":status"}
	r, _, buf := newTestREPLWithHistory(t, lines)

	err := r.ExecuteCommand(context.Background(), ":history --json -n 100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	var result []struct {
		Index   int    `json:"index"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, output)
	}

	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
	if result[0].Command != ":send hello" {
		t.Errorf("expected first command to be ':send hello', got %q", result[0].Command)
	}
	if result[1].Command != ":status" {
		t.Errorf("expected second command to be ':status', got %q", result[1].Command)
	}
}

func TestHistoryExportPlainText(t *testing.T) {
	lines := []string{":send hello", ":status", ":send world"}
	r, tmpDir, _ := newTestREPLWithHistory(t, lines)

	exportPath := filepath.Join(tmpDir, "export.txt")
	err := r.ExecuteCommand(context.Background(), ":history -e "+exportPath+" -n 100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, ":send hello") {
		t.Errorf("export should contain ':send hello'")
	}
	if !strings.Contains(content, ":status") {
		t.Errorf("export should contain ':status'")
	}
	if !strings.Contains(content, ":send world") {
		t.Errorf("export should contain ':send world'")
	}

	// Should NOT be JSON
	if strings.Contains(content, "{") {
		t.Errorf("plain text export should not contain JSON")
	}
}

func TestHistoryExportJSONL(t *testing.T) {
	lines := []string{":send hello", ":status"}
	r, tmpDir, _ := newTestREPLWithHistory(t, lines)

	exportPath := filepath.Join(tmpDir, "export.jsonl")
	err := r.ExecuteCommand(context.Background(), ":history -e "+exportPath+" -n 100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("failed to read export file: %v", err)
	}

	jsonLines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(jsonLines) != 2 {
		t.Fatalf("expected 2 JSONL lines, got %d", len(jsonLines))
	}

	var entry struct {
		Index   int    `json:"index"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(jsonLines[0]), &entry); err != nil {
		t.Fatalf("invalid JSONL line: %v", err)
	}
	if entry.Command != ":send hello" {
		t.Errorf("expected ':send hello', got %q", entry.Command)
	}
}

func TestHistoryClear(t *testing.T) {
	lines := []string{":send hello", ":status"}
	r, _, buf := newTestREPLWithHistory(t, lines)

	// historyClear reads from stdin via fmt.Scanln, but we can test the
	// method more directly. The integration test for the clear command
	// would require piping "y\n" to stdin. Instead, we verify that:
	// 1. The command parses the flag correctly
	// 2. The helper performs the truncation when called directly

	// Directly test historyClear by simulating: write "y" to stdin wouldn't work
	// in unit test context. Instead test that -c flag is recognized.
	err := r.ExecuteCommand(context.Background(), ":history -c")
	// This will hang waiting for input in a real scenario, but in test
	// fmt.Scanln on empty stdin reads "" which maps to "no" -> "Cancelled."
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Cancelled") {
		t.Errorf("expected 'Cancelled' in output when no confirmation given, got:\n%s", output)
	}

	// Verify history file is not cleared
	data, _ := os.ReadFile(r.config.HistoryFile)
	if len(strings.TrimSpace(string(data))) == 0 {
		t.Errorf("history file should not be cleared without confirmation")
	}
}

func TestHistoryCombinedFlags(t *testing.T) {
	lines := []string{":send deploy-v1", ":status", ":send deploy-v2", ":send hello", ":send deploy-v1"}
	r, _, buf := newTestREPLWithHistory(t, lines)

	err := r.ExecuteCommand(context.Background(), ":history -s deploy --unique --json -n 100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	var result []struct {
		Index   int    `json:"index"`
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("invalid JSON output: %v\noutput: %s", err, output)
	}

	// Search for "deploy" -> 3 entries: deploy-v1, deploy-v2, deploy-v1
	// Unique -> 2 entries: deploy-v2, deploy-v1 (last occurrences kept)
	if len(result) != 2 {
		t.Errorf("expected 2 entries after search+unique, got %d: %+v", len(result), result)
	}
}

func TestHistoryNoHistoryFile(t *testing.T) {
	var buf bytes.Buffer
	r, err := New(ClientMode, &Config{
		Terminal:    true,
		HistoryFile: "",
		Stdout:      &nopCloser{&buf},
	})
	if err != nil {
		t.Fatalf("failed to create REPL: %v", err)
	}
	// Override HistoryFile explicitly to empty
	r.config.HistoryFile = ""
	r.TemplateEngine = template.New(false)
	r.RegisterCommonCommands()

	err = r.ExecuteCommand(context.Background(), ":history")
	if err == nil {
		t.Fatalf("expected error when no history file configured")
	}
	if !strings.Contains(err.Error(), "not enabled") {
		t.Errorf("expected 'not enabled' error, got: %v", err)
	}
}

func TestHistoryEmptyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "history-empty-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	historyFile := filepath.Join(tmpDir, "history")
	if err := os.WriteFile(historyFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write history file: %v", err)
	}

	var buf bytes.Buffer
	r, err := New(ClientMode, &Config{
		Terminal:    true,
		HistoryFile: historyFile,
		Stdout:      &nopCloser{&buf},
	})
	if err != nil {
		t.Fatalf("failed to create REPL: %v", err)
	}
	r.TemplateEngine = template.New(false)
	r.RegisterCommonCommands()
	r.Display.Color = "off"

	err = r.ExecuteCommand(context.Background(), ":history")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No history found") {
		t.Errorf("expected 'No history found' for empty file, got:\n%s", output)
	}
}

func TestHistoryFilterInvalidRegex(t *testing.T) {
	lines := []string{":send hello"}
	r, _, _ := newTestREPLWithHistory(t, lines)

	err := r.ExecuteCommand(context.Background(), ":history -f /[invalid/")
	if err == nil {
		t.Fatalf("expected error for invalid regex")
	}
	if !strings.Contains(err.Error(), "invalid regex") {
		t.Errorf("expected 'invalid regex' error, got: %v", err)
	}
}

func TestHistorySearchNoResults(t *testing.T) {
	lines := []string{":send hello", ":status"}
	r, _, buf := newTestREPLWithHistory(t, lines)

	err := r.ExecuteCommand(context.Background(), ":history -s nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No matching history entries") {
		t.Errorf("expected 'No matching history entries', got:\n%s", output)
	}
}

func TestHighlightSearchTerm(t *testing.T) {
	r, _, _ := newTestREPLWithHistory(t, []string{":send hello"})

	tests := []struct {
		name   string
		text   string
		term   string
		expect string // just check it contains the term still
	}{
		{"basic", "hello world", "world", "world"},
		{"case insensitive", "Hello WORLD", "world", "WORLD"},
		{"multiple matches", "foo bar foo", "foo", "foo"},
		{"no match", "hello", "xyz", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.highlightSearchTerm(tt.text, tt.term)
			if !strings.Contains(result, tt.expect) {
				t.Errorf("expected result to contain %q, got %q", tt.expect, result)
			}
		})
	}
}
