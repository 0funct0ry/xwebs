package handler

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/segmentio/kafka-go"
)

type kafkaManager struct {
	writers sync.Map // map[string]*kafka.Writer (key is "brokers|topic")
}

// NewKafkaManager creates a new thread-safe Kafka manager.
func NewKafkaManager() KafkaManager {
	return &kafkaManager{}
}

func (m *kafkaManager) Produce(ctx context.Context, brokers []string, topic, key, message string) error {
	if len(brokers) == 0 {
		return fmt.Errorf("no kafka brokers specified")
	}

	brokerStr := strings.Join(brokers, ",")
	cacheKey := fmt.Sprintf("%s|%s", brokerStr, topic)

	var writer *kafka.Writer
	if v, ok := m.writers.Load(cacheKey); ok {
		writer = v.(*kafka.Writer)
	} else {
		writer = &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: true,
		}
		m.writers.Store(cacheKey, writer)
	}

	msg := kafka.Message{
		Value: []byte(message),
	}
	if key != "" {
		msg.Key = []byte(key)
	}

	return writer.WriteMessages(ctx, msg)
}

func (m *kafkaManager) Consume(ctx context.Context, brokers []string, topic, groupID, offset string, callback func(topic string, offset int64, key, message []byte)) error {
	if len(brokers) == 0 {
		return fmt.Errorf("no kafka brokers specified")
	}

	startOffset := kafka.LastOffset
	if strings.ToLower(offset) == "earliest" {
		startOffset = kafka.FirstOffset
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		StartOffset: startOffset,
		MinBytes:    1,    // Snappy delivery
		MaxBytes:    10e6, // 10MB
	})

	if groupID == "" {
		// If no group ID, we need to set the offset manually if it's not a consumer group
		if err := reader.SetOffset(startOffset); err != nil {
			reader.Close()
			return fmt.Errorf("setting kafka offset: %w", err)
		}
	}

	defer reader.Close()

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("reading kafka message: %w", err)
		}

		callback(msg.Topic, msg.Offset, msg.Key, msg.Value)
	}
}

func (m *kafkaManager) Close() error {
	var lastErr error
	m.writers.Range(func(key, value interface{}) bool {
		writer := value.(*kafka.Writer)
		if err := writer.Close(); err != nil {
			lastErr = err
		}
		return true
	})
	return lastErr
}
