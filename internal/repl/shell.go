package repl

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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
	stdoutW := r.StdoutWithColor("green") // Shell output in green for clarity
	stderrW := r.StdoutWithColor("red")   // Stderr in red

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

// switchToShell drops the user into a full interactive system shell.
// It synchronizes environment variables and working directory changes.
func (r *REPL) switchToShell(ctx context.Context, skipConfirm bool) error {
	// 1. Confirmation
	if !skipConfirm {
		r.Printf("Switch to full interactive shell? (y/N) ")

		// Use a simple scanner to read confirmation without triggering the full REPL logic
		var response string
		_, _ = fmt.Scanln(&response)
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			r.Printf("Shell mode cancelled.\n")
			return nil
		}
	}

	// 2. Determine shell
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		if runtime.GOOS == "windows" {
			shellPath = "cmd.exe"
		} else {
			// Try common shells
			shells := []string{"/bin/zsh", "/bin/bash", "/bin/sh"}
			for _, s := range shells {
				if _, err := os.Stat(s); err == nil {
					shellPath = s
					break
				}
			}
		}
	}
	if shellPath == "" {
		shellPath = "sh" // Last resort
	}

	// 3. Prepare environment
	env := os.Environ()

	// Add session variables
	vars := r.GetVars()
	for k, v := range vars {
		env = append(env, fmt.Sprintf("XWEBS_VAR_%s=%v", strings.ToUpper(k), v))
	}
	env = append(env, "XWEBS_SHELL_MODE=true")
	env = append(env, "XWEBS_SHELL_PARENT_PID="+fmt.Sprint(os.Getpid()))

	// 4. Run shell command
	cmd := exec.CommandContext(ctx, shellPath)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	r.Printf("\nDropping into shell: %s\nType 'exit' or press Ctrl+D to return to xwebs.\n\n", shellPath)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	// 5. Track CWD changes in the background
	var lastKnownCwd string
	stopTracking := make(chan struct{})
	trackingDone := make(chan struct{})

	go func() {
		defer close(trackingDone)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-stopTracking:
				return
			case <-ticker.C:
				cwd := getPidCwd(cmd.Process.Pid)
				if cwd != "" {
					lastKnownCwd = cwd
				}
			}
		}
	}()

	// Wait for the shell to exit
	if err := cmd.Wait(); err != nil {
		// Non-zero exit code is fine
		if _, ok := err.(*exec.ExitError); !ok {
			r.Errorf("Shell exited with error: %v\n", err)
		}
	}

	close(stopTracking)
	<-trackingDone

	// 6. Synchronize WD back to REPL
	if lastKnownCwd != "" {
		oldCwd, _ := os.Getwd()
		absLastCwd, err := filepath.Abs(lastKnownCwd)
		if err == nil {
			lastKnownCwd = absLastCwd
		}

		if lastKnownCwd != oldCwd {
			if err := os.Chdir(lastKnownCwd); err == nil {
				r.prevDir = oldCwd
				r.Printf("\nDirectory changed to: %s\n", r.Display.colorizedText(lastKnownCwd, "cyan"))
			}
		}
	} else if runtime.GOOS == "windows" {
		r.Notify("[note: working directory synchronization is not supported on Windows]\n")
	}

	r.Printf("Returned to xwebs REPL.\n")
	r.renderPrompt()

	return nil
}

// getPidCwd returns the current working directory of a process given its PID.
// Supported on Linux and macOS. Returns empty string on other platforms.
func getPidCwd(pid int) string {
	switch runtime.GOOS {
	case "linux":
		cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
		if err != nil {
			return ""
		}
		return cwd
	case "darwin":
		// Use lsof -p <pid> -a -d cwd -Fn
		out, err := exec.Command("lsof", "-p", fmt.Sprint(pid), "-a", "-d", "cwd", "-Fn").Output()
		if err != nil {
			return ""
		}
		// Output format:
		// p123
		// n/path/to/cwd
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "n") {
				return strings.TrimPrefix(line, "n")
			}
		}
	}
	return ""
}
