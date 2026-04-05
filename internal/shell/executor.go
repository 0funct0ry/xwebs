package shell

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ExecutionResult holds the results of a shell command execution.
type ExecutionResult struct {
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	ExitCode int           `json:"exit_code"`
	Duration time.Duration `json:"duration"`
}

// Execute runs a shell command with the given context, stdin, and environment variables.
// It uses "sh -c" to execute the command string, allowing for shell features like pipes and redirects.
func Execute(ctx context.Context, command string, stdin io.Reader, env map[string]string) (*ExecutionResult, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = stdin

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	result := &ExecutionResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			// If context was cancelled or timed out
			if ctx.Err() != nil {
				result.ExitCode = -1
				return result, fmt.Errorf("command execution timed out or was cancelled: %w", err)
			}
			result.ExitCode = -1
		}
		return result, fmt.Errorf("command execution failed: %w", err)
	}

	result.ExitCode = 0
	return result, nil
}
