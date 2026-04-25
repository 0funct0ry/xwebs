package server

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0funct0ry/xwebs/internal/handler"
	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
)

// topicSubscription holds a single client's subscription state for a topic.
type topicSubscription struct {
	connID       string
	conn         handler.Connection
	remoteAddr   string
	subscribedAt time.Time
	msgsSent     uint64 // updated atomically
}

// topicEntry is the internal record for a single topic.
type topicEntry struct {
	mu            sync.RWMutex
	subscriptions map[string]*topicSubscription // keyed by connID
	lastActive    time.Time
	retained      *ws.Message
}

// TopicStore is the in-memory pub/sub engine.  It is safe for concurrent use.
type TopicStore struct {
	mu     sync.RWMutex
	topics map[string]*topicEntry
}

// newTopicStore creates an empty TopicStore.
func newTopicStore() *TopicStore {
	return &TopicStore{
		topics: make(map[string]*topicEntry),
	}
}

// Subscribe registers conn as a subscriber of topic, creating the topic if needed.
// Calling Subscribe again for the same (connID, topic) pair is a no-op.
func (ts *TopicStore) Subscribe(connID string, conn handler.Connection, topic string) {
	ts.mu.Lock()
	entry, ok := ts.topics[topic]
	if !ok {
		entry = &topicEntry{
			subscriptions: make(map[string]*topicSubscription),
			lastActive:    time.Now(),
		}
		ts.topics[topic] = entry
	}
	ts.mu.Unlock()

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if _, exists := entry.subscriptions[connID]; !exists {
		sub := &topicSubscription{
			connID:       connID,
			conn:         conn,
			remoteAddr:   conn.RemoteAddr(),
			subscribedAt: time.Now(),
		}
		entry.subscriptions[connID] = sub

		// Deliver retained message if present
		if entry.retained != nil {
			_ = conn.Write(entry.retained)
		}
	}
}

// Unsubscribe removes connID from topic.  Returns the number of remaining
// subscribers.  The topic entry is deleted when the last subscriber leaves.
func (ts *TopicStore) Unsubscribe(connID, topic string) int {
	ts.mu.Lock()
	entry, ok := ts.topics[topic]
	if !ok {
		ts.mu.Unlock()
		return 0
	}
	ts.mu.Unlock()

	entry.mu.Lock()
	delete(entry.subscriptions, connID)
	remaining := len(entry.subscriptions)
	entry.mu.Unlock()

	if remaining == 0 {
		ts.mu.Lock()
		// Re-check under write lock to avoid a race where another goroutine
		// re-subscribed between the two locks.
		entry.mu.RLock()
		stillEmpty := len(entry.subscriptions) == 0 && entry.retained == nil
		entry.mu.RUnlock()
		if stillEmpty {
			delete(ts.topics, topic)
		}
		ts.mu.Unlock()
	}

	return remaining
}

// UnsubscribeAll removes connID from every topic it is currently subscribed to.
// Returns the names of the topics that were affected (sorted).
func (ts *TopicStore) UnsubscribeAll(connID string) []string {
	ts.mu.RLock()
	// Collect topic names where connID appears.
	var affected []string
	for name, entry := range ts.topics {
		entry.mu.RLock()
		_, has := entry.subscriptions[connID]
		entry.mu.RUnlock()
		if has {
			affected = append(affected, name)
		}
	}
	ts.mu.RUnlock()

	for _, name := range affected {
		ts.Unsubscribe(connID, name)
	}

	sort.Strings(affected)
	return affected
}

// Publish fans out msg to all current subscribers of topic.
// Returns the number of clients the message was delivered to.
// Returns an error only when the topic does not exist (zero subscribers).
func (ts *TopicStore) Publish(topic string, msg *ws.Message) (int, error) {
	ts.mu.RLock()
	entry, ok := ts.topics[topic]
	ts.mu.RUnlock()

	if !ok {
		return 0, fmt.Errorf("topic %q has no subscribers. Message not sent", topic)
	}

	entry.mu.Lock()
	entry.lastActive = time.Now()
	subs := make([]*topicSubscription, 0, len(entry.subscriptions))
	for _, s := range entry.subscriptions {
		subs = append(subs, s)
	}
	entry.mu.Unlock()

	delivered := 0
	for _, sub := range subs {
		if err := sub.conn.Write(msg); err == nil {
			atomic.AddUint64(&sub.msgsSent, 1)
			delivered++
		}
	}

	return delivered, nil
}

// PublishSticky stores msg as the retained value for topic and fans it out to current subscribers.
func (ts *TopicStore) PublishSticky(topic string, msg *ws.Message) (int, error) {
	ts.mu.Lock()
	entry, ok := ts.topics[topic]
	if !ok {
		entry = &topicEntry{
			subscriptions: make(map[string]*topicSubscription),
			lastActive:    time.Now(),
		}
		ts.topics[topic] = entry
	}
	ts.mu.Unlock()

	entry.mu.Lock()
	entry.retained = msg
	entry.lastActive = time.Now()
	subs := make([]*topicSubscription, 0, len(entry.subscriptions))
	for _, s := range entry.subscriptions {
		subs = append(subs, s)
	}
	entry.mu.Unlock()

	delivered := 0
	for _, sub := range subs {
		if err := sub.conn.Write(msg); err == nil {
			atomic.AddUint64(&sub.msgsSent, 1)
			delivered++
		}
	}

	return delivered, nil
}

// ClearRetained removes the retained message for a topic.
func (ts *TopicStore) ClearRetained(topic string) {
	ts.mu.RLock()
	entry, ok := ts.topics[topic]
	ts.mu.RUnlock()

	if !ok {
		return
	}

	entry.mu.Lock()
	entry.retained = nil
	empty := len(entry.subscriptions) == 0
	entry.mu.Unlock()

	if empty {
		ts.mu.Lock()
		// Re-check under lock
		entry.mu.RLock()
		if len(entry.subscriptions) == 0 && entry.retained == nil {
			delete(ts.topics, topic)
		}
		entry.mu.RUnlock()
		ts.mu.Unlock()
	}
}

// GetTopics returns metadata for all active topics (sorted by name).
func (ts *TopicStore) GetTopics() []template.TopicInfo {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	infos := make([]template.TopicInfo, 0, len(ts.topics))
	for name, entry := range ts.topics {
		entry.mu.RLock()
		subs := buildSubscriberInfos(entry)
		lastActive := entry.lastActive
		var retained interface{}
		if entry.retained != nil {
			retained = string(entry.retained.Data)
		}
		entry.mu.RUnlock()

		infos = append(infos, template.TopicInfo{
			Name:        name,
			Subscribers: subs,
			LastActive:  lastActive,
			Retained:    retained,
		})
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})
	return infos
}

// GetTopic returns metadata for a single topic by name.
func (ts *TopicStore) GetTopic(name string) (template.TopicInfo, bool) {
	ts.mu.RLock()
	entry, ok := ts.topics[name]
	ts.mu.RUnlock()

	if !ok {
		return template.TopicInfo{}, false
	}

	entry.mu.RLock()
	subs := buildSubscriberInfos(entry)
	lastActive := entry.lastActive
	var retained interface{}
	if entry.retained != nil {
		retained = string(entry.retained.Data)
	}
	entry.mu.RUnlock()

	return template.TopicInfo{
		Name:        name,
		Subscribers: subs,
		LastActive:  lastActive,
		Retained:    retained,
	}, true
}

// TopicSubscriberCount returns the number of subscribers for a topic.
func (ts *TopicStore) TopicSubscriberCount(name string) int {
	ts.mu.RLock()
	entry, ok := ts.topics[name]
	ts.mu.RUnlock()

	if !ok {
		return 0
	}

	entry.mu.RLock()
	n := len(entry.subscriptions)
	entry.mu.RUnlock()
	return n
}

// buildSubscriberInfos converts the internal subscription map to a sorted
// slice of TopicSubscriberInfo.  Must be called with entry.mu held (at least RLock).
func buildSubscriberInfos(entry *topicEntry) []template.TopicSubscriberInfo {
	subs := make([]template.TopicSubscriberInfo, 0, len(entry.subscriptions))
	for _, s := range entry.subscriptions {
		subs = append(subs, template.TopicSubscriberInfo{
			ConnID:       s.connID,
			RemoteAddr:   s.remoteAddr,
			SubscribedAt: s.subscribedAt,
			MsgsSent:     atomic.LoadUint64(&s.msgsSent),
		})
	}
	sort.Slice(subs, func(i, j int) bool {
		return subs[i].ConnID < subs[j].ConnID
	})
	return subs
}
