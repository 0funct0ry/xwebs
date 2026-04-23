package server

import (
	"testing"
	"time"

	"github.com/0funct0ry/xwebs/internal/ws"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockConn is a mock implementation of handler.Connection
type mockConn struct {
	mock.Mock
}

func (m *mockConn) Write(msg *ws.Message) error {
	args := m.Called(msg)
	return args.Error(0)
}

func (m *mockConn) CloseWithCode(code int, reason string) error {
	return nil
}

func (m *mockConn) Subscribe() <-chan *ws.Message {
	return nil
}

func (m *mockConn) Unsubscribe(ch <-chan *ws.Message) {}

func (m *mockConn) Done() <-chan struct{} {
	return nil
}

func (m *mockConn) IsCompressionEnabled() bool {
	return false
}

func (m *mockConn) GetID() string {
	return m.Called().String(0)
}

func (m *mockConn) GetURL() string {
	return "ws://localhost"
}

func (m *mockConn) GetSubprotocol() string {
	return ""
}

func (m *mockConn) RemoteAddr() string {
	return "127.0.0.1:1234"
}

func (m *mockConn) LocalAddr() string {
	return "127.0.0.1:8080"
}

func (m *mockConn) ConnectedAt() time.Time {
	return time.Now()
}

func (m *mockConn) MessageCount() uint64 {
	return 0
}

func (m *mockConn) MsgsIn() uint64 {
	return 0
}

func (m *mockConn) MsgsOut() uint64 {
	return 0
}

func (m *mockConn) LastMsgReceivedAt() time.Time {
	return time.Time{}
}

func (m *mockConn) LastMsgSentAt() time.Time {
	return time.Time{}
}

func (m *mockConn) RTT() time.Duration {
	return 0
}

func (m *mockConn) AvgRTT() time.Duration {
	return 0
}

func TestTopicStore(t *testing.T) {
	ts := newTopicStore()

	conn1 := new(mockConn)
	conn1.On("GetID").Return("c1").Maybe()
	conn2 := new(mockConn)
	conn2.On("GetID").Return("c2").Maybe()

	// 1. Subscribe
	ts.Subscribe("c1", conn1, "news")
	ts.Subscribe("c2", conn2, "news")

	topics := ts.GetTopics()
	assert.Len(t, topics, 1)
	assert.Equal(t, "news", topics[0].Name)
	assert.Len(t, topics[0].Subscribers, 2)

	// 2. Publish
	msg := &ws.Message{Data: []byte("hello")}
	conn1.On("Write", msg).Return(nil)
	conn2.On("Write", msg).Return(nil)

	delivered, err := ts.Publish("news", msg)
	assert.NoError(t, err)
	assert.Equal(t, 2, delivered)

	conn1.AssertExpectations(t)
	conn2.AssertExpectations(t)

	// 3. Unsubscribe
	remaining := ts.Unsubscribe("c1", "news")
	assert.Equal(t, 1, remaining)

	info, ok := ts.GetTopic("news")
	assert.True(t, ok)
	assert.Len(t, info.Subscribers, 1)
	assert.Equal(t, "c2", info.Subscribers[0].ConnID)

	// 4. Unsubscribe last (topic should be removed)
	remaining = ts.Unsubscribe("c2", "news")
	assert.Equal(t, 0, remaining)

	_, ok = ts.GetTopic("news")
	assert.False(t, ok)
	assert.Empty(t, ts.GetTopics())

	// 5. UnsubscribeAll
	ts.Subscribe("c1", conn1, "t1")
	ts.Subscribe("c1", conn1, "t2")
	ts.Subscribe("c2", conn2, "t2")

	affected := ts.UnsubscribeAll("c1")
	assert.Equal(t, []string{"t1", "t2"}, affected)

	t2Info, _ := ts.GetTopic("t2")
	assert.Len(t, t2Info.Subscribers, 1)
	assert.Equal(t, "c2", t2Info.Subscribers[0].ConnID)

	_, t1Exists := ts.GetTopic("t1")
	assert.False(t, t1Exists)
}

func TestTopicStore_PublishNoSubscribers(t *testing.T) {
	ts := newTopicStore()
	msg := &ws.Message{Data: []byte("hello")}

	delivered, err := ts.Publish("empty", msg)
	assert.Error(t, err)
	assert.Equal(t, 0, delivered)
	assert.Contains(t, err.Error(), "no subscribers")
}
