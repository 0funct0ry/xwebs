package handler

import (
	"context"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockKafkaManager struct {
	mock.Mock
}

func (m *mockKafkaManager) Produce(ctx context.Context, brokers []string, topic, key, message string) error {
	args := m.Called(ctx, brokers, topic, key, message)
	return args.Error(0)
}

func (m *mockKafkaManager) Consume(ctx context.Context, brokers []string, topic, groupID, offset string, callback func(topic string, offset int64, key, payload []byte)) error {
	args := m.Called(ctx, brokers, topic, groupID, offset, callback)
	return args.Error(0)
}

func (m *mockKafkaManager) Close() error {
	return nil
}

func TestKafkaProduceBuiltin(t *testing.T) {
	builtin := &KafkaProduceBuiltin{}
	
	// Test Validate
	t.Run("Validate", func(t *testing.T) {
		assert.Error(t, builtin.Validate(Action{}))
		assert.Error(t, builtin.Validate(Action{Brokers: []string{"localhost:9092"}}))
		assert.Error(t, builtin.Validate(Action{Brokers: []string{"localhost:9092"}, Topic: "test"}))
		assert.NoError(t, builtin.Validate(Action{Brokers: []string{"localhost:9092"}, Topic: "test", Message: "hello"}))
	})

	// Test Execute
	t.Run("Execute", func(t *testing.T) {
		m := new(mockKafkaManager)
		reg := NewRegistry(ServerMode)
		conn := &mockConn{}
		engine := template.New(false)
		
		d := NewDispatcher(reg, conn, engine, true, nil, nil, false, nil, nil, nil, nil, nil, nil, nil, m, nil, "")
		
		a := &Action{
			Brokers: []string{"localhost:9092"},
			Topic:   "test-{{.Message}}",
			Message: "Payload: {{.Message}}",
			Key:     "key-{{.Message}}",
		}
		
		tmplCtx := template.NewContext()
		tmplCtx.Message = "foo"
		
		m.On("Produce", mock.Anything, []string{"localhost:9092"}, "test-foo", "key-foo", "Payload: foo").Return(nil)
		
		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		assert.NoError(t, err)
		m.AssertExpectations(t)
	})

	// Test Execute with default brokers
	t.Run("ExecuteDefaultBrokers", func(t *testing.T) {
		m := new(mockKafkaManager)
		reg := NewRegistry(ServerMode)
		conn := &mockConn{}
		engine := template.New(false)
		d := NewDispatcher(reg, conn, engine, true, nil, nil, false, nil, nil, nil, nil, nil, nil, nil, m, nil, "")
		
		a := &Action{
			Brokers: nil, // Should default to localhost:9092
			Topic:   "test",
			Message: "hello",
		}
		
		m.On("Produce", mock.Anything, []string{"localhost:9092"}, "test", "", "hello").Return(nil)
		
		err := builtin.Execute(context.Background(), d, a, template.NewContext())
		assert.NoError(t, err)
		m.AssertExpectations(t)
	})
}
