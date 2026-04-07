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
// If allowlist is provided, it validates the command before execution.
func Execute(ctx context.Context, command string, stdin io.Reader, env map[string]string, allowlist []string) (*ExecutionResult, error) {
	if allowlist != nil {
		if err := ValidateCommand(command, allowlist); err != nil {
			return &ExecutionResult{ExitCode: -1, Stderr: err.Error()}, err
		}
	}

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
			return result, nil // Non-zero exit code is a defined state, not a fatal execution error 
		}
		
		// Fatal execution errors (cannot start, timeout, context cancel)
		result.ExitCode = -1
		if ctx.Err() != nil {
			return result, fmt.Errorf("command execution timed out or was cancelled: %w", err)
		}
		return result, fmt.Errorf("command execution failed: %w", err)
	}


	result.ExitCode = 0
	return result, nil
}

// ValidateCommand checks if a command string is allowed under the given allowlist.
func ValidateCommand(command string, allowlist []string) error {
	if len(allowlist) == 0 {
		return fmt.Errorf("command execution rejected: sandbox enabled but allowlist is empty")
	}

	// Reject command substitution which is hard to parse securely
	if strings.Contains(command, "$(") || strings.Contains(command, "`") {
		return fmt.Errorf("command execution rejected: command substitution ($() or ``) is not allowed in sandbox mode")
	}

	// Split by common shell delimiters
	delimiters := []string{";", "&&", "||", "|", "&"}
	parts := []string{command}
	for _, dev := range delimiters {
		var newParts []string
		for _, p := range parts {
			split := strings.Split(p, dev)
			newParts = append(newParts, split...)
		}
		parts = newParts
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Skip environment variable assignments at the start of a command
		// e.g. "DEBUG=1 ls -l" -> we want to check "ls"
		words := strings.Fields(part)
		if len(words) == 0 {
			continue
		}

		cmdIdx := 0
		for cmdIdx < len(words) {
			// Simple check for assignment: contains '=' and doesn't start with '-'
			if strings.Contains(words[cmdIdx], "=") && !strings.HasPrefix(words[cmdIdx], "-") {
				cmdIdx++
				continue
			}
			break
		}

		if cmdIdx >= len(words) {
			continue // Only assignments
		}

		baseCmd := words[cmdIdx]
		
		allowed := false
		for _, a := range allowlist {
			if baseCmd == a {
				allowed = true
				break
			}
		}

		if !allowed {
			return fmt.Errorf("command execution rejected: %q is not in the allowlist", baseCmd)
		}
	}

	return nil
}

