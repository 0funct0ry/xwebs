package server

import (
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockConnection struct {
	mock.Mock
	id string
}

func (m *mockConnection) Write(msg *ws.Message) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *mockConnection) CloseWithCode(code int, reason string) error { return nil }
func (m *mockConnection) Subscribe() <-chan *ws.Message               { return nil }
func (m *mockConnection) Unsubscribe(ch <-chan *ws.Message)           {}
func (m *mockConnection) Done() <-chan struct{}                       { return nil }
func (m *mockConnection) IsCompressionEnabled() bool                  { return false }
func (m *mockConnection) GetID() string                               { return m.id }
func (m *mockConnection) GetURL() string                              { return "" }
func (m *mockConnection) GetSubprotocol() string                      { return "" }
func (m *mockConnection) RemoteAddr() string                          { return "127.0.0.1:12345" }
func (m *mockConnection) LocalAddr() string                           { return "127.0.0.1:8080" }
func (m *mockConnection) ConnectedAt() time.Time                      { return time.Now() }
func (m *mockConnection) MessageCount() uint64                        { return 0 }
func (m *mockConnection) MsgsIn() uint64                              { return 0 }
func (m *mockConnection) MsgsOut() uint64                             { return 0 }
func (m *mockConnection) LastMsgReceivedAt() time.Time                { return time.Now() }
func (m *mockConnection) LastMsgSentAt() time.Time                    { return time.Now() }
func (m *mockConnection) RTT() time.Duration                          { return 0 }
func (m *mockConnection) AvgRTT() time.Duration                       { return 0 }

func TestTopicStore_StickyPublish(t *testing.T) {
	ts := newTopicStore()
	topic := "dashboard"

	conn1 := &mockConnection{id: "client1"}
	conn2 := &mockConnection{id: "client2"}

	msg1 := &ws.Message{Data: []byte("initial state")}

	// 1. Publish sticky to a topic with no subscribers
	delivered, err := ts.PublishSticky(topic, msg1)
	assert.NoError(t, err)
	assert.Equal(t, 0, delivered)

	// 2. Subscribe a client and check if they get the retained message
	conn1.On("Write", msg1).Return(nil).Once()
	ts.Subscribe(conn1.id, conn1, topic)
	conn1.AssertExpectations(t)

	// 3. Update sticky message
	msg2 := &ws.Message{Data: []byte("updated state")}
	conn1.On("Write", msg2).Return(nil).Once()
	delivered, err = ts.PublishSticky(topic, msg2)
	assert.NoError(t, err)
	assert.Equal(t, 1, delivered)
	conn1.AssertExpectations(t)

	// 4. Subscribe another client and check if they get the LATEST retained message
	conn2.On("Write", msg2).Return(nil).Once()
	ts.Subscribe(conn2.id, conn2, topic)
	conn2.AssertExpectations(t)

	// 5. Verify Unsubscribe doesn't delete topic if retained message exists
	ts.Unsubscribe(conn1.id, topic)
	ts.Unsubscribe(conn2.id, topic)

	info, ok := ts.GetTopic(topic)
	assert.True(t, ok)
	assert.Equal(t, 0, len(info.Subscribers))
	assert.Equal(t, "updated state", info.Retained)

	// 6. Clear retained and verify topic is deleted
	ts.ClearRetained(topic)
	_, ok = ts.GetTopic(topic)
	assert.False(t, ok, "Topic should be deleted after clearing retained message if no subscribers")
}
