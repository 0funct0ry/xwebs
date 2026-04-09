package repl

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/0funct0ry/xwebs/internal/shell"
)

// executeShellCommand runs an arbitrary shell command and streams its output.
// It supports an optional -i flag for interactive mode.
func (r *REPL) executeShellCommand(ctx context.Context, shellCmd string) error {
	shellCmd = strings.TrimSpace(shellCmd)
	if shellCmd == "" {
		return fmt.Errorf("no command provided")
	}

	interactive := false
	if strings.HasPrefix(shellCmd, "-i ") {
		interactive = true
		shellCmd = strings.TrimPrefix(shellCmd, "-i ")
	} else if shellCmd == "-i" {
		interactive = true
		shellCmd = ""
	}

	if shellCmd == "" && interactive {
		return fmt.Errorf("usage: :! -i <command>")
	}

	// Create a command-specific context that can be canceled by SIGINT
	// Note: ExecuteCommand already sets up a cancelable context, but we 
	// manage it here again to be safe and clear about the command's lifecycle.
	// However, we use the passed ctx which is already tied to the REPL's signal handling.
	
	if interactive {
		r.Notify("Starting interactive shell: %s\n", shellCmd)
		
		// In interactive mode, we connect directly to the system TTY.
		res, err := shell.ExecuteStreaming(ctx, shellCmd, os.Stdout, os.Stderr, os.Stdin, nil)
		if err != nil {
			return err
		}
		
		if res.ExitCode != 0 {
			r.Errorf("\nCommand exited with code %d (duration: %v)\n", res.ExitCode, res.Duration)
		} else {
			r.Notify("\nCommand completed (duration: %v)\n", res.Duration)
		}
		return nil
	}

	// Captured streaming mode
	r.Notify("Executing: %s\n", shellCmd)
	
	// Create writers that stream to the REPL output
	stdoutW := &replWriter{r: r, color: "green"} // Shell output in green for clarity
	stderrW := &replWriter{r: r, color: "red"}   // Stderr in red

	res, err := shell.ExecuteStreaming(ctx, shellCmd, stdoutW, stderrW, os.Stdin, nil)
	if err != nil {
		return err
	}

	if res.ExitCode != 0 {
		r.Errorf("\nCommand exited with code %d (duration: %v)\n", res.ExitCode, res.Duration)
	} else {
		r.Notify("\nCommand completed (duration: %v)\n", res.Duration)
	}

	return nil
}

// replWriter implements io.Writer by calling r.Printf for each chunk.
type replWriter struct {
	r     *REPL
	color string
}

func (w *replWriter) Write(p []byte) (n int, err error) {
	text := string(p)
	if w.color != "" {
		text = w.r.Display.colorizedText(text, w.color)
	}
	w.r.Printf("%s", text)
	return len(p), nil
}
