package handler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

type natsManager struct {
	conns map[string]*nats.Conn
	mu    sync.RWMutex
}

// NewNATSManager creates a new NATS manager with connection pooling.
func NewNATSManager() NATSManager {
	return &natsManager{
		conns: make(map[string]*nats.Conn),
	}
}

func (m *natsManager) Publish(ctx context.Context, natsURL, subject, message string) error {
	nc, err := m.getOrCreateConn(natsURL)
	if err != nil {
		return err
	}

	return nc.Publish(subject, []byte(message))
}

func (m *natsManager) getOrCreateConn(natsURL string) (*nats.Conn, error) {
	m.mu.RLock()
	nc, ok := m.conns[natsURL]
	m.mu.RUnlock()

	if ok && nc.Status() == nats.CONNECTED {
		return nc, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check after acquiring lock
	nc, ok = m.conns[natsURL]
	if ok && nc.Status() == nats.CONNECTED {
		return nc, nil
	}

	// Create new connection
	opts := []nats.Option{
		nats.Name("xwebs"),
		nats.Timeout(5 * time.Second),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(5),
	}

	newNC, err := nats.Connect(natsURL, opts...)
	if err != nil {
		return nil, fmt.Errorf("connecting to NATS server %s: %w", natsURL, err)
	}

	m.conns[natsURL] = newNC
	return newNC, nil
}

func (m *natsManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for url, nc := range m.conns {
		if nc.Status() == nats.CONNECTED {
			nc.Close()
		}
		delete(m.conns, url)
	}
	return nil
}
