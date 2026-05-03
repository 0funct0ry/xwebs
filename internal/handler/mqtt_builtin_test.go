package handler

import (
	"context"
	"testing"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockMQTTManager struct {
	mock.Mock
}

func (m *MockMQTTManager) Publish(ctx context.Context, brokerURL, topic, message string, qos byte, retain bool) error {
	args := m.Called(ctx, brokerURL, topic, message, qos, retain)
	return args.Error(0)
}

func (m *MockMQTTManager) Subscribe(brokerURL, topic string, qos byte, callback func(topic string, payload []byte)) (func(), error) {
	args := m.Called(brokerURL, topic, qos, callback)
	return args.Get(0).(func()), args.Error(1)
}

func (m *MockMQTTManager) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestMQTTPublishBuiltin(t *testing.T) {
	mqttMock := new(MockMQTTManager)
	reg := NewRegistry(ServerMode)
	engine := template.New(false)
	
	d := NewDispatcher(reg, nil, engine, true, nil, nil, false, nil, nil, nil, nil, nil, mqttMock, nil, nil, nil, "")
	
	builtin := &MQTTPublishBuiltin{}
	
	t.Run("successful publish", func(t *testing.T) {
		action := &Action{
			BrokerURL: "tcp://localhost:1883",
			Topic:     "test/topic",
			Message:   "hello mqtt",
			QoS:       "1",
			Retain:    true,
		}
		
		mqttMock.On("Publish", mock.Anything, "tcp://localhost:1883", "test/topic", "hello mqtt", byte(1), true).Return(nil).Once()
		
		tmplCtx := template.NewContext()
		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		
		assert.NoError(t, err)
		mqttMock.AssertExpectations(t)
	})
	
	t.Run("template support", func(t *testing.T) {
		action := &Action{
			BrokerURL: "tcp://{{.Vars.host}}:1883",
			Topic:     "test/{{.Vars.subtopic}}",
			Message:   "val: {{.Vars.val}}",
			QoS:       "{{.Vars.qos}}",
		}
		
		mqttMock.On("Publish", mock.Anything, "tcp://mqtt.local:1883", "test/sensor", "val: 42", byte(2), false).Return(nil).Once()
		
		tmplCtx := template.NewContext()
		tmplCtx.Vars["host"] = "mqtt.local"
		tmplCtx.Vars["subtopic"] = "sensor"
		tmplCtx.Vars["val"] = 42
		tmplCtx.Vars["qos"] = "2"
		
		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		
		assert.NoError(t, err)
		mqttMock.AssertExpectations(t)
	})
	
	t.Run("missing broker_url", func(t *testing.T) {
		action := &Action{
			Topic:   "test",
			Message: "test",
		}
		err := builtin.Validate(*action)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing broker_url")
	})

	t.Run("invalid QoS", func(t *testing.T) {
		action := &Action{
			BrokerURL: "tcp://localhost:1883",
			Topic:     "test",
			Message:   "test",
			QoS:       "5",
		}
		tmplCtx := template.NewContext()
		err := builtin.Execute(context.Background(), d, action, tmplCtx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid mqtt qos")
	})
}
