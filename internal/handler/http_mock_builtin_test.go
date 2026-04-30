package handler

import (
	"context"
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockHTTPMockServer struct {
	mockPath string
	mockResp template.HTTPMockResponse
}

func (m *mockHTTPMockServer) GetClientCount() int                   { return 0 }
func (m *mockHTTPMockServer) GetUptime() time.Duration             { return 0 }
func (m *mockHTTPMockServer) GetClients() []template.ClientInfo   { return nil }
func (m *mockHTTPMockServer) IsPaused() bool                      { return false }
func (m *mockHTTPMockServer) WaitIfPaused()                       {}
func (m *mockHTTPMockServer) Broadcast(msg *ws.Message, excludeIDs ...string) int { return 0 }
func (m *mockHTTPMockServer) Send(id string, msg *ws.Message) error { return nil }
func (m *mockHTTPMockServer) SendToSSE(stream, event, data, id string) error { return nil }
func (m *mockHTTPMockServer) UpdateSSEStreamConfig(stream, onNoConsumers string, bufferSize int) error { return nil }
func (m *mockHTTPMockServer) RegisterHTTPMock(path string, mock template.HTTPMockResponse) error {
	m.mockPath = path
	m.mockResp = mock
	return nil
}

func TestHttpMockRespondBuiltin(t *testing.T) {
	builtin := &HttpMockRespondBuiltin{}

	t.Run("Name and Scope", func(t *testing.T) {
		assert.Equal(t, "http-mock-respond", builtin.Name())
		assert.Equal(t, ServerOnly, builtin.Scope())
	})

	t.Run("Validate", func(t *testing.T) {
		assert.Error(t, builtin.Validate(Action{}))
		assert.Error(t, builtin.Validate(Action{Path: "/test"}))
		assert.NoError(t, builtin.Validate(Action{Path: "/test", Status: "200"}))
	})

	t.Run("Execute", func(t *testing.T) {
		mockServer := &mockHTTPMockServer{}
		d := &Dispatcher{
			serverStats:    mockServer,
			templateEngine: template.New(false),
		}

		a := &Action{
			Path:   "/api/{{ .Vars.name }}",
			Status: "201",
			Headers: map[string]string{
				"X-Custom": "{{ .Vars.val }}",
			},
			Body: "Created {{ .Vars.name }}",
		}

		tmplCtx := template.NewContext()
		tmplCtx.Vars["name"] = "testuser"
		tmplCtx.Vars["val"] = "foo"

		err := builtin.Execute(context.Background(), d, a, tmplCtx)
		require.NoError(t, err)

		assert.Equal(t, "/api/testuser", mockServer.mockPath)
		assert.Equal(t, 201, mockServer.mockResp.Status)
		assert.Equal(t, "foo", mockServer.mockResp.Headers["X-Custom"])
		assert.Equal(t, "Created testuser", mockServer.mockResp.Body)
	})
}
