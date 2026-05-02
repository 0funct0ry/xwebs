package handler

import (
	"context"
	"fmt"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mqttManager struct {
	clients map[string]mqtt.Client
	mu      sync.RWMutex
}

// NewMQTTManager creates a new MQTT manager with connection pooling.
func NewMQTTManager() MQTTManager {
	return &mqttManager{
		clients: make(map[string]mqtt.Client),
	}
}

func (m *mqttManager) Publish(ctx context.Context, brokerURL, topic, message string, qos byte, retain bool) error {
	client, err := m.getOrCreateClient(brokerURL)
	if err != nil {
		return err
	}

	token := client.Publish(topic, qos, retain, message)
	
	// Wait for publish to complete or context to be cancelled
	done := make(chan struct{})
	go func() {
		token.Wait()
		close(done)
	}()

	select {
	case <-done:
		return token.Error()
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *mqttManager) Subscribe(brokerURL, topic string, qos byte, callback func(topic string, payload []byte)) (func(), error) {
	client, err := m.getOrCreateClient(brokerURL)
	if err != nil {
		return nil, err
	}

	token := client.Subscribe(topic, qos, func(c mqtt.Client, msg mqtt.Message) {
		callback(msg.Topic(), msg.Payload())
	})

	if token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("subscribing to %s on %s: %w", topic, brokerURL, token.Error())
	}

	unsubscribe := func() {
		client.Unsubscribe(topic)
	}

	return unsubscribe, nil
}

func (m *mqttManager) getOrCreateClient(brokerURL string) (mqtt.Client, error) {
	m.mu.RLock()
	client, ok := m.clients[brokerURL]
	m.mu.RUnlock()

	if ok && client.IsConnected() {
		return client, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check after acquiring lock
	client, ok = m.clients[brokerURL]
	if ok && client.IsConnected() {
		return client, nil
	}

	// Create new client
	opts := mqtt.NewClientOptions().AddBroker(brokerURL)
	opts.SetAutoReconnect(true)
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetKeepAlive(60 * time.Second)
	opts.SetClientID(fmt.Sprintf("xwebs-%d", time.Now().UnixNano()))

	newClient := mqtt.NewClient(opts)
	if token := newClient.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("connecting to MQTT broker %s: %w", brokerURL, token.Error())
	}

	m.clients[brokerURL] = newClient
	return newClient, nil
}

func (m *mqttManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for url, client := range m.clients {
		if client.IsConnected() {
			client.Disconnect(250)
		}
		delete(m.clients, url)
	}
	return nil
}
