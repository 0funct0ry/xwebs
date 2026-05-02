package handler

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNATSSubscribeBuiltin_Validate(t *testing.T) {
	b := &NATSSubscribeBuiltin{}

	tests := []struct {
		name    string
		action  Action
		wantErr bool
	}{
		{
			name: "valid",
			action: Action{
				NatsURL: "nats://localhost:4222",
				Subject: "test",
			},
			wantErr: false,
		},
		{
			name: "missing nats_url",
			action: Action{
				Subject: "test",
			},
			wantErr: true,
		},
		{
			name: "missing subject",
			action: Action{
				NatsURL: "nats://localhost:4222",
			},
			wantErr: true,
		},
		{
			name: "invalid reconnect_interval",
			action: Action{
				NatsURL:           "nats://localhost:4222",
				Subject:           "test",
				ReconnectInterval: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := b.Validate(tt.action)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNATSSubscribeBuiltin_Execute(t *testing.T) {
	b := &NATSSubscribeBuiltin{}
	err := b.Execute(context.Background(), nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source action")
}
