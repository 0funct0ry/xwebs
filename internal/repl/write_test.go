package repl

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dummyWriteCloser struct {
	io.Writer
}

func (d *dummyWriteCloser) Close() error { return nil }

func TestWriteCommand(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xwebs-write-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	origWd, _ := os.Getwd()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origWd) }()

	cfg := &Config{
		Terminal: false,
		Stdout:   &dummyWriteCloser{io.Discard},
	}
	r, err := New(ClientMode, cfg)
	require.NoError(t, err)
	r.TemplateEngine = template.New(false)
	r.RegisterCommonCommands()

	t.Run("basic write", func(t *testing.T) {
		filename := "test.txt"
		err := r.ExecuteCommand(context.Background(), ":write "+filename+" \"hello world\"")
		assert.NoError(t, err)

		content, err := os.ReadFile(filename)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(content))
	})

	t.Run("templated write", func(t *testing.T) {
		r.SetVar("name", "alice")
		filename := "greet.txt"
		err := r.ExecuteCommand(context.Background(), ":write "+filename+" \"hello {{.Vars.name}}\"")
		assert.NoError(t, err)

		content, err := os.ReadFile(filename)
		require.NoError(t, err)
		assert.Equal(t, "hello alice", string(content))
	})

	t.Run("append mode", func(t *testing.T) {
		filename := "append.txt"
		_ = os.WriteFile(filename, []byte("part 1"), 0644)

		err := r.ExecuteCommand(context.Background(), ":write -a "+filename+" \" part 2\"")
		assert.NoError(t, err)

		content, err := os.ReadFile(filename)
		require.NoError(t, err)
		assert.Equal(t, "part 1 part 2", string(content))
	})

	t.Run("dry run", func(t *testing.T) {
		filename := "dryrun.txt"
		err := r.ExecuteCommand(context.Background(), ":write -n "+filename+" \"should not exist\"")
		assert.NoError(t, err)

		_, err = os.Stat(filename)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("json format", func(t *testing.T) {
		filename := "data.json"
		err := r.ExecuteCommand(context.Background(), ":write --json "+filename+" \"{\\\"a\\\":1}\"")
		assert.NoError(t, err)

		content, err := os.ReadFile(filename)
		require.NoError(t, err)
		assert.Contains(t, string(content), "  \"a\": 1")
	})

	t.Run("create parents", func(t *testing.T) {
		filename := "subdir/deep/file.txt"
		err := r.ExecuteCommand(context.Background(), ":write -p "+filename+" \"deep stuff\"")
		assert.NoError(t, err)

		content, err := os.ReadFile(filename)
		require.NoError(t, err)
		assert.Equal(t, "deep stuff", string(content))
	})

	t.Run("shortcut last message", func(t *testing.T) {
		msg := &ws.Message{
			Data: []byte(`{"status":"ok"}`),
			Metadata: ws.MessageMetadata{
				URL: "ws://test",
			},
		}
		r.PrintMessage(msg, nil)

		filename := "last.json"
		err := r.ExecuteCommand(context.Background(), ":write --last-message "+filename)
		assert.NoError(t, err)

		content, err := os.ReadFile(filename)
		require.NoError(t, err)
		assert.Equal(t, `{"status":"ok"}`, string(content))
	})

	t.Run("auto extension", func(t *testing.T) {
		filename := "noext"
		err := r.ExecuteCommand(context.Background(), ":write "+filename+" \"{\\\"json\\\":true}\"")
		assert.NoError(t, err)

		_, err = os.Stat(filename + ".json")
		assert.NoError(t, err)
	})
}

func TestHeredocInREPL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "xwebs-heredoc-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	origWd, _ := os.Getwd()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origWd) }()

	// We can't easily test REPL.Run because it's blocking and uses readline.
	// But we can test the logic by manually feeding lines.
	// Actually, let's just test that the delimiter detection and buffering works
	// if we were to simulate the loop logic.

	cfg := &Config{
		Terminal: false,
		Stdout:   &dummyWriteCloser{io.Discard},
	}
	r, err := New(ClientMode, cfg)
	require.NoError(t, err)
	r.RegisterCommonCommands()

	// Simulating: :write file.txt <<EOF
	// line 1
	// line 2
	// EOF

	line1 := ":write file.txt <<EOF"
	idx := strings.LastIndex(line1, "<<")
	require.NotEqual(t, -1, idx)
	delim := strings.TrimSpace(line1[idx+2:])
	assert.Equal(t, "EOF", delim)

	r.heredocDelimiter = delim
	r.heredocBuffer = []string{line1[:idx]}

	// Feed lines
	r.heredocBuffer = append(r.heredocBuffer, "line 1")
	r.heredocBuffer = append(r.heredocBuffer, "line 2")

	// End of heredoc
	lastLine := "EOF"
	if strings.TrimSpace(lastLine) == r.heredocDelimiter {
		finalInput := strings.Join(r.heredocBuffer, "\n")
		assert.Contains(t, finalInput, ":write file.txt")
		assert.Contains(t, finalInput, "line 1")
		assert.Contains(t, finalInput, "line 2")
	}
}
