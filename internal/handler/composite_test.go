package handler

import (
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_CompositeMatch(t *testing.T) {
	reg := NewRegistry(ServerMode)
	falseVal := false

	err := reg.AddHandlers([]Handler{
		{
			Name: "all_match",
			Match: Matcher{
				All: []Matcher{
					{Binary: &falseVal}, // must be text
					{Regex: ".*error.*"},
				},
			},
			Respond: "ok",
		},
		{
			Name: "any_match",
			Match: Matcher{
				Any: []Matcher{
					{Pattern: "ping"},
					{Pattern: "pong"},
				},
			},
			Respond: "ok",
		},
		{
			Name: "nested_match",
			Match: Matcher{
				All: []Matcher{
					{Regex: ".*critical.*"},
					{
						Any: []Matcher{
							{JQ: ".level == \"error\""},
							{JQ: ".level == \"fatal\""},
						},
					},
				},
			},
			Respond: "ok",
		},
		{
			Name: "mixed_top_and_composite",
			Match: Matcher{
				Regex: ".*alert.*",
				All: []Matcher{
					{JQ: ".source == \"system\""},
				},
			},
			Respond: "ok",
		},
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		input   string
		msgType ws.MessageType
		expect  []string
	}{
		{"all match success", "system error occurred", ws.TextMessage, []string{"all_match"}},
		{"all match binary fail", "system error occurred", ws.BinaryMessage, nil},
		{"all match regex fail", "system ok", ws.TextMessage, nil},
		{"any match ping", "ping", ws.TextMessage, []string{"any_match"}},
		{"any match pong", "pong", ws.TextMessage, []string{"any_match"}},
		{"any match fail", "pang", ws.TextMessage, nil},
		{"nested match error", `{"level": "error", "msg": "critical failure"}`, ws.TextMessage, []string{"all_match", "nested_match"}},
		{"nested match fatal", `{"level": "fatal", "msg": "critical failure"}`, ws.TextMessage, []string{"nested_match"}},
		{"nested match fail level", `{"level": "info", "msg": "critical failure"}`, ws.TextMessage, nil},
		{"nested match fail regex", `{"level": "error", "msg": "low failure"}`, ws.TextMessage, []string{"all_match"}},
		{"mixed match success", `{"source": "system", "msg": "alert: overheating"}`, ws.TextMessage, []string{"mixed_top_and_composite"}},
		{"mixed match fail top regex", `{"source": "system", "msg": "overheating"}`, ws.TextMessage, nil},
		{"mixed match fail all jq", `{"source": "user", "msg": "alert: overheating"}`, ws.TextMessage, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := template.New(false)
			ctx := template.NewContext()
			ctx.Message = tt.input

			msg := &ws.Message{Data: []byte(tt.input), Type: tt.msgType}
			matches, err := reg.Match(msg, engine, ctx)
			require.NoError(t, err)

			var names []string
			for _, m := range matches {
				names = append(names, m.Handler.Name)
			}
			assert.ElementsMatch(t, tt.expect, names)
		})
	}
}
