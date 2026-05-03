package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/0funct0ry/xwebs/internal/template"
)

// KafkaProduceBuiltin produces a message to a Kafka topic.
type KafkaProduceBuiltin struct{}

func (b *KafkaProduceBuiltin) Name() string { return "kafka-produce" }
func (b *KafkaProduceBuiltin) Description() string {
	return "Produce a message to a Kafka topic."
}
func (b *KafkaProduceBuiltin) Scope() BuiltinScope { return Shared }

func (b *KafkaProduceBuiltin) Validate(a Action) error {
	if a.Topic == "" {
		return fmt.Errorf("builtin kafka-produce missing topic")
	}
	if a.Message == "" {
		return fmt.Errorf("builtin kafka-produce missing message")
	}
	return nil
}

func (b *KafkaProduceBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.kafkaManager == nil {
		return fmt.Errorf("builtin kafka-produce: kafka manager not available")
	}

	brokers := a.Brokers
	if len(brokers) == 0 {
		brokers = []string{"localhost:9092"}
	}

	// Render templates
	renderedBrokers := make([]string, len(brokers))
	for i, br := range brokers {
		rb, err := d.templateEngine.Execute("kafka-broker", br, tmplCtx)
		if err != nil {
			return fmt.Errorf("template error in broker %q: %w", br, err)
		}
		renderedBrokers[i] = strings.TrimSpace(rb)
	}

	topic, err := d.templateEngine.Execute("kafka-topic", a.Topic, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in topic: %w", err)
	}
	topic = strings.TrimSpace(topic)

	message, err := d.templateEngine.Execute("kafka-message", a.Message, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in message: %w", err)
	}

	var key string
	if a.Key != "" {
		key, err = d.templateEngine.Execute("kafka-key", a.Key, tmplCtx)
		if err != nil {
			return fmt.Errorf("template error in key: %w", err)
		}
	}

	// Apply timeout if specified
	execCtx := ctx
	if a.Timeout != "" {
		if dur, err := time.ParseDuration(a.Timeout); err == nil {
			var cancel context.CancelFunc
			execCtx, cancel = context.WithTimeout(ctx, dur)
			defer cancel()
		}
	}

	if d.verbose {
		d.log("  [handler] kafka-produce: brokers=%v topic=%q key=%q\n", renderedBrokers, topic, key)
	}

	err = d.kafkaManager.Produce(execCtx, renderedBrokers, topic, key, message)
	if err != nil {
		return fmt.Errorf("kafka produce error: %w", err)
	}

	return nil
}

// KafkaConsumeBuiltin consumes messages from a Kafka topic.
type KafkaConsumeBuiltin struct{}

func (b *KafkaConsumeBuiltin) Name() string { return "kafka-consume" }
func (b *KafkaConsumeBuiltin) Description() string {
	return "Consume messages from a Kafka topic and broadcast them to WebSocket clients."
}
func (b *KafkaConsumeBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *KafkaConsumeBuiltin) Validate(a Action) error {
	if a.Topic == "" {
		return fmt.Errorf("builtin kafka-consume missing topic")
	}
	if a.Offset != "" {
		o := strings.ToLower(a.Offset)
		if o != "earliest" && o != "latest" {
			return fmt.Errorf("builtin kafka-consume: invalid offset %q (must be 'earliest' or 'latest')", a.Offset)
		}
	}
	return nil
}

func (b *KafkaConsumeBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	return fmt.Errorf("builtin %q is a source action and cannot be executed in a reactive flow", b.Name())
}
