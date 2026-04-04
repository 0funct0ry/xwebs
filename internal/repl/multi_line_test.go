package repl

import (
	"context"
	"io"
	"os"
	"testing"
	"time"
)

func TestREPLMultiLineRun(t *testing.T) {
	// Create a pipe to simulate stdin
	rPipe, wPipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	defer rPipe.Close()
	defer wPipe.Close()

	// cfg.Stdin will use the pipe, no need for os.Stdin redirection.

	cfg := &Config{
		Prompt:             "> ",
		ContinuationPrompt: "... ",
		Stdin:              rPipe,
		Stdout:             wPipe,
		Terminal:           true,
	}
	r, err := New(ClientMode, cfg)
	if err != nil {
		t.Fatalf("failed to create REPL: %v", err)
	}

	var capturedInputs []string
	r.SetOnInput(func(ctx context.Context, text string) error {
		capturedInputs = append(capturedInputs, text)
		return nil
	})

	// Start REPL in a goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- r.Run(ctx)
	}()

	// Feed inputs through the pipe
	inputs := []string{
		"single line\n",
		"part1 \\\n",
		"part2\n",
		"{ \\\n",
		"  \"foo\": 1 \\\n",
		"}\n",
	}

	for _, input := range inputs {
		_, _ = wPipe.WriteString(input)
		time.Sleep(10 * time.Millisecond) // Give REPL time to process
	}

	// Close the pipe and wait for EOF
	wPipe.Close()
	
	select {
	case err := <-done:
		if err != nil && err != io.EOF {
			t.Errorf("REPL exited with error: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Errorf("REPL timed out")
	}

	expected := []string{
		"single line",
		"part1 \npart2",
		"{ \n  \"foo\": 1 \n}",
	}

	if len(capturedInputs) != len(expected) {
		t.Fatalf("Expected %d inputs, got %d: %v", len(expected), len(capturedInputs), capturedInputs)
	}

	for i, exp := range expected {
		if capturedInputs[i] != exp {
			t.Errorf("Input %d: expected %q, got %q", i, exp, capturedInputs[i])
		}
	}
}
