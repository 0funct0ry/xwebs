package handler

import (
	"context"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
)

type mockBuiltin struct {
	name  string
	scope BuiltinScope
}

func (m *mockBuiltin) Name() string            { return m.name }
func (m *mockBuiltin) Description() string     { return "Mock description" }
func (m *mockBuiltin) Scope() BuiltinScope     { return m.scope }
func (m *mockBuiltin) Validate(a Action) error { return nil }
func (m *mockBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	return nil
}

func TestRegister(t *testing.T) {
	// Already registered builtins should exists
	_, ok := GetBuiltin("subscribe")
	if !ok {
		t.Error("expected 'subscribe' to be registered via init()")
	}

	// Test case-insensitivity
	_, ok = GetBuiltin("SUBSCRIBE")
	if !ok {
		t.Error("expected GetBuiltin to be case-insensitive")
	}

	// Test duplicate registration
	m := &mockBuiltin{name: "subscribe", scope: Shared}
	err := Register(m)
	if err == nil {
		t.Error("expected error when registering duplicate builtin name")
	}

	// Test successful registration
	mNew := &mockBuiltin{name: "new-builtin", scope: Shared}
	err = Register(mNew)
	if err != nil {
		t.Errorf("expected no error when registering new builtin, got %v", err)
	}

	_, ok = GetBuiltin("new-builtin")
	if !ok {
		t.Error("expected 'new-builtin' to be registered")
	}
}

func TestIsBuiltinAllowed(t *testing.T) {
	tests := []struct {
		name    string
		mode    RegistryMode
		allowed bool
		exists  bool
		scope   BuiltinScope
	}{
		{"subscribe", ServerMode, true, true, ServerOnly},
		{"subscribe", ClientMode, false, true, ServerOnly},
		{"noop", ServerMode, true, true, Shared},
		{"noop", ClientMode, true, true, Shared},
		{"non-existent", ServerMode, false, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name+"-"+string(tt.mode), func(t *testing.T) {
			allowed, exists, scope := IsBuiltinAllowed(tt.name, tt.mode)
			if allowed != tt.allowed {
				t.Errorf("allowed = %v, want %v", allowed, tt.allowed)
			}
			if exists != tt.exists {
				t.Errorf("exists = %v, want %v", exists, tt.exists)
			}
			if scope != tt.scope {
				t.Errorf("scope = %v, want %v", scope, tt.scope)
			}
		})
	}
}

func TestBuiltinValidation(t *testing.T) {
	// Note: We are testing Action.Validate which now delegates to bh.Validate
	tests := []struct {
		name    string
		action  Action
		mode    RegistryMode
		wantErr bool
	}{
		{
			name:    "valid subscribe",
			action:  Action{Type: "builtin", Command: "subscribe", Topic: "foo"},
			mode:    ServerMode,
			wantErr: false,
		},
		{
			name:    "missing topic subscribe",
			action:  Action{Type: "builtin", Command: "subscribe"},
			mode:    ServerMode,
			wantErr: true,
		},
		{
			name:    "valid kv-set",
			action:  Action{Type: "builtin", Command: "kv-set", Key: "k", Value: "v"},
			mode:    ServerMode,
			wantErr: false,
		},
		{
			name:    "missing key kv-set",
			action:  Action{Type: "builtin", Command: "kv-set", Value: "v"},
			mode:    ServerMode,
			wantErr: true,
		},
		{
			name:    "valid echo",
			action:  Action{Type: "builtin", Command: "echo"},
			mode:    ServerMode,
			wantErr: false,
		},
		{
			name:    "client mode echo (Shared)",
			action:  Action{Type: "builtin", Command: "echo"},
			mode:    ClientMode,
			wantErr: false,
		},
		{
			name:    "client mode subscribe (ServerOnly)",
			action:  Action{Type: "builtin", Command: "subscribe", Topic: "foo"},
			mode:    ClientMode,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action.Validate(tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
