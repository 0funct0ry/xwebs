package shell

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		allowlist []string
		wantErr   bool
	}{
		{
			name:      "simple allowed",
			command:   "ls -l",
			allowlist: []string{"ls", "git"},
			wantErr:   false,
		},
		{
			name:      "simple disallowed",
			command:   "rm -rf /",
			allowlist: []string{"ls", "git"},
			wantErr:   true,
		},
		{
			name:      "pipelined allowed",
			command:   "ls | grep foo",
			allowlist: []string{"ls", "grep"},
			wantErr:   false,
		},
		{
			name:      "pipelined disallowed",
			command:   "ls | rm -rf /",
			allowlist: []string{"ls", "grep"},
			wantErr:   true,
		},
		{
			name:      "chained allowed",
			command:   "ls && git status",
			allowlist: []string{"ls", "git"},
			wantErr:   false,
		},
		{
			name:      "chained disallowed",
			command:   "ls && rm -rf /",
			allowlist: []string{"ls", "git"},
			wantErr:   true,
		},
		{
			name:      "env assignment allowed",
			command:   "DEBUG=1 ls -l",
			allowlist: []string{"ls"},
			wantErr:   false,
		},
		{
			name:      "multiple env assignments allowed",
			command:   "FOO=bar BAZ=qux git commit",
			allowlist: []string{"git"},
			wantErr:   false,
		},
		{
			name:      "command substitution rejected $()",
			command:   "ls $(whoami)",
			allowlist: []string{"ls", "whoami"},
			wantErr:   true,
		},
		{
			name:      "command substitution rejected ``",
			command:   "ls `whoami`",
			allowlist: []string{"ls", "whoami"},
			wantErr:   true,
		},
		{
			name:      "empty allowlist",
			command:   "ls",
			allowlist: []string{},
			wantErr:   true,
		},
		{
			name:      "redirects allowed if command is allowed",
			command:   "ls > /tmp/foo",
			allowlist: []string{"ls"},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCommand(tt.command, tt.allowlist)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecuteWithSandbox(t *testing.T) {
	ctx := context.Background()

	t.Run("allowed command", func(t *testing.T) {
		result, err := Execute(ctx, "echo hello", nil, nil, []string{"echo"})
		assert.NoError(t, err)
		assert.Equal(t, "hello\n", result.Stdout)
		assert.Equal(t, 0, result.ExitCode)
	})

	t.Run("disallowed command", func(t *testing.T) {
		_, err := Execute(ctx, "ls", nil, nil, []string{"echo"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in the allowlist")
	})

	t.Run("no sandbox", func(t *testing.T) {
		// Should allow anything when allowlist is nil
		result, err := Execute(ctx, "echo hello", nil, nil, nil)
		assert.NoError(t, err)
		assert.Equal(t, "hello\n", result.Stdout)
	})
}
