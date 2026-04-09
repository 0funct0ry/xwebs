package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/0funct0ry/xwebs/internal/repl"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipeSupport_Redirection(t *testing.T) {
	// Create a temporary file to act as non-TTY Stdout
	tmpFile, err := os.CreateTemp("", "xwebs-stdout-pipe-*")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	cfg := &repl.Config{
		Stdout: tmpFile,
	}

	r, err := repl.New(repl.ClientMode, cfg)
	require.NoError(t, err)

	// Ensure detection works
	assert.False(t, r.IsStdoutTTY(), "Expected Stdout to be detected as non-TTY")

	// Capture Stderr
	oldStderr := os.Stderr
	r_pipe, w_pipe, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w_pipe
	defer func() { os.Stderr = oldStderr }()

	// Notify should go to Stderr when not interactive and Stdout is piped
	r.IsInteractive = false
	r.Notify("Connecting to %s...", "wss://example.com")

	w_pipe.Close()
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r_pipe)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "Connecting to wss://example.com...", "Notify should have redirected to Stderr")

	// Verify Stdout is clean
	stdoutContent, err := os.ReadFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Empty(t, string(stdoutContent), "Stdout should be empty (no info messages)")
}

func TestPipeSupport_FormattingNoIndicators(t *testing.T) {
	fs := repl.NewFormattingState()
	fs.NoIndicators = true

	msg := &ws.Message{
		Data: []byte("hello world"),
		Metadata: ws.MessageMetadata{
			Direction: "received",
		},
	}

	formatted, ok := fs.FormatMessage(msg, nil, nil)
	assert.True(t, ok)
	assert.Equal(t, "hello world", formatted, "Expected clean output with no indicators")

	fs.NoIndicators = false
	formatted, ok = fs.FormatMessage(msg, nil, nil)
	assert.True(t, ok)
	assert.Contains(t, formatted, "⬇", "Expected indicator when NoIndicators is false")
}
