package handler

import (
	"context"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricBuiltin(t *testing.T) {
	b := &MetricBuiltin{}
	assert.Equal(t, "metric", b.Name())
	assert.Equal(t, Shared, b.Scope())

	t.Run("Validation", func(t *testing.T) {
		err := b.Validate(Action{Type: "builtin", Command: "metric"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing 'name'")

		err = b.Validate(Action{Type: "builtin", Command: "metric", Name: "test_metric"})
		assert.NoError(t, err)
	})

	t.Run("Execute", func(t *testing.T) {
		reg := NewRegistry(ServerMode)
		engine := template.New(false)
		d := NewDispatcher(reg, nil, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")
		tmplCtx := template.NewContext()
		tmplCtx.Message = "{\"type\": \"login\"}"

		action := &Action{
			Type:    "builtin",
			Command: "metric",
			Name:    "user_events_total",
			Labels: FlexLabels{Map: map[string]string{
				"event_type": "login",
				"static":     "val",
			}},
		}

		err := b.Execute(context.Background(), d, action, tmplCtx)
		require.NoError(t, err)
	})

	t.Run("Template Error", func(t *testing.T) {
		reg := NewRegistry(ServerMode)
		engine := template.New(false)
		d := NewDispatcher(reg, nil, engine, false, nil, nil, false, nil, nil, nil, nil, nil, "")
		tmplCtx := template.NewContext()

		action := &Action{
			Type:    "builtin",
			Command: "metric",
			Name:    "metric_{{.Invalid}}",
		}

		err := b.Execute(context.Background(), d, action, tmplCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "template error")
	})
}
