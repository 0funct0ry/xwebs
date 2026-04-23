package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogBuiltin(t *testing.T) {
	// Setup temporary directory for log files
	tmpDir, err := os.MkdirTemp("", "xwebs-log-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tmplEngine := template.New(false)

	tests := []struct {
		name       string
		action     Action
		tmplCtx    *template.TemplateContext
		setup      func()
		verify     func(t *testing.T, stdout string, stderr string)
		wantError  bool
	}{
		{
			name: "Log to stdout (default)",
			action: Action{
				Message: "Hello {{.ConnectionID}}",
			},
			tmplCtx: &template.TemplateContext{
				ConnectionID: "conn-123",
			},
			verify: func(t *testing.T, stdout string, stderr string) {
				assert.Contains(t, stdout, "conn-123")
				var log map[string]interface{}
				err := json.Unmarshal([]byte(stdout), &log)
				assert.NoError(t, err)
				assert.Equal(t, "Hello conn-123", log["message"])
				assert.Equal(t, "conn-123", log["conn_id"])
				assert.NotEmpty(t, log["timestamp"])
			},
		},
		{
			name: "Log to file",
			action: Action{
				Target:  "file",
				Path:    filepath.Join(tmpDir, "test.log"),
				Message: "Log entry",
			},
			tmplCtx: &template.TemplateContext{
				ConnectionID: "conn-file",
			},
			verify: func(t *testing.T, stdout string, stderr string) {
				path := filepath.Join(tmpDir, "test.log")
				data, err := os.ReadFile(path)
				assert.NoError(t, err)
				
				var log map[string]interface{}
				err = json.Unmarshal(data, &log)
				assert.NoError(t, err)
				assert.Equal(t, "Log entry", log["message"])
				assert.Equal(t, "conn-file", log["conn_id"])
			},
		},
		{
			name: "Log to file with template in path",
			action: Action{
				Target:  "file",
				Path:    filepath.Join(tmpDir, "log-{{.ConnectionID}}.log"),
				Message: "Templated path",
			},
			tmplCtx: &template.TemplateContext{
				ConnectionID: "c99",
			},
			verify: func(t *testing.T, stdout string, stderr string) {
				path := filepath.Join(tmpDir, "log-c99.log")
				data, err := os.ReadFile(path)
				assert.NoError(t, err)
				assert.Contains(t, string(data), "Templated path")
			},
		},
		{
			name: "Log to both",
			action: Action{
				Target:  "both",
				Path:    filepath.Join(tmpDir, "both.log"),
				Message: "Both targets",
			},
			tmplCtx: &template.TemplateContext{
				ConnectionID: "conn-both",
			},
			verify: func(t *testing.T, stdout string, stderr string) {
				// Check stdout
				assert.Contains(t, stdout, "Both targets")
				
				// Check file
				path := filepath.Join(tmpDir, "both.log")
				data, err := os.ReadFile(path)
				assert.NoError(t, err)
				assert.Contains(t, string(data), "Both targets")
			},
		},
		{
			name: "Validation error: missing path for file target",
			action: Action{
				Target: "file",
			},
			tmplCtx:   &template.TemplateContext{},
			wantError: true,
		},
		{
			name: "Invalid target",
			action: Action{
				Target: "nowhere",
			},
			tmplCtx:   &template.TemplateContext{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr strings.Builder
			d := &Dispatcher{
				templateEngine: tmplEngine,
				Log: func(f string, a ...interface{}) {
					stdout.WriteString(fmt.Sprintf(f, a...))
				},
				Error: func(f string, a ...interface{}) {
					stderr.WriteString(fmt.Sprintf(f, a...))
				},
			}

			if tt.setup != nil {
				tt.setup()
			}

			builtin, ok := GetBuiltin("log")
			require.True(t, ok)

			err := builtin.Validate(tt.action)
			if err == nil {
				err = builtin.Execute(context.Background(), d, &tt.action, tt.tmplCtx)
			}
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.verify != nil {
					tt.verify(t, stdout.String(), stderr.String())
				}
			}
		})
	}
}

func TestLogBuiltinAppend(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xwebs-log-append-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "append.log")
	tmplEngine := template.New(false)
	d := &Dispatcher{
		templateEngine: tmplEngine,
		Log:            func(f string, a ...interface{}) {},
		Error:          func(f string, a ...interface{}) {},
	}

	builtin, _ := GetBuiltin("log")
	action := Action{
		Target:  "file",
		Path:    logPath,
		Message: "First",
	}

	// First write
	err = builtin.Execute(context.Background(), d, &action, &template.TemplateContext{ConnectionID: "1"})
	assert.NoError(t, err)

	// Second write
	action.Message = "Second"
	err = builtin.Execute(context.Background(), d, &action, &template.TemplateContext{ConnectionID: "2"})
	assert.NoError(t, err)

	// Read and verify two lines
	data, err := os.ReadFile(logPath)
	assert.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	assert.Equal(t, 2, len(lines))
	assert.Contains(t, lines[0], "First")
	assert.Contains(t, lines[1], "Second")
}
