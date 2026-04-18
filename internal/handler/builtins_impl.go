package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/0funct0ry/xwebs/internal/template"
	"github.com/0funct0ry/xwebs/internal/ws"
)

func init() {
	MustRegister(&SubscribeBuiltin{})
	MustRegister(&UnsubscribeBuiltin{})
	MustRegister(&PublishBuiltin{})
	MustRegister(&KVSetBuiltin{})
	MustRegister(&KVGetBuiltin{})
	MustRegister(&KVDelBuiltin{})
	MustRegister(&NoopBuiltin{})
	MustRegister(&EchoBuiltin{})
}

// SubscribeBuiltin subscribes the current connection to a pub/sub topic.
type SubscribeBuiltin struct{}

func (b *SubscribeBuiltin) Name() string        { return "subscribe" }
func (b *SubscribeBuiltin) Description() string { return "Subscribe the current connection to a pub/sub topic." }
func (b *SubscribeBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *SubscribeBuiltin) Validate(a Action) error {
	if a.Topic == "" {
		return fmt.Errorf("builtin subscribe missing topic")
	}
	return nil
}

func (b *SubscribeBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.topicManager == nil {
		return fmt.Errorf("builtin subscribe: topic manager not available")
	}
	topic, err := d.templateEngine.Execute("topic", a.Topic, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in topic expression: %w", err)
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return fmt.Errorf("builtin subscribe: topic evaluates to empty string")
	}
	d.topicManager.Subscribe(d.conn.GetID(), d.conn, topic)
	if d.verbose {
		d.errorf("  [handler] subscribed %s to topic %q\n", d.conn.GetID(), topic)
	}
	return nil
}

// UnsubscribeBuiltin unsubscribes the current connection from a pub/sub topic.
type UnsubscribeBuiltin struct{}

func (b *UnsubscribeBuiltin) Name() string        { return "unsubscribe" }
func (b *UnsubscribeBuiltin) Description() string { return "Unsubscribe the current connection from a pub/sub topic." }
func (b *UnsubscribeBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *UnsubscribeBuiltin) Validate(a Action) error {
	if a.Topic == "" {
		return fmt.Errorf("builtin unsubscribe missing topic")
	}
	return nil
}

func (b *UnsubscribeBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.topicManager == nil {
		return fmt.Errorf("builtin unsubscribe: topic manager not available")
	}
	topic, err := d.templateEngine.Execute("topic", a.Topic, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in topic expression: %w", err)
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return fmt.Errorf("builtin unsubscribe: topic evaluates to empty string")
	}
	remaining := d.topicManager.Unsubscribe(d.conn.GetID(), topic)
	if d.verbose {
		d.errorf("  [handler] unsubscribed %s from topic %q (%d remaining)\n", d.conn.GetID(), topic, remaining)
	}
	return nil
}

// PublishBuiltin publishes a message to a pub/sub topic.
type PublishBuiltin struct{}

func (b *PublishBuiltin) Name() string        { return "publish" }
func (b *PublishBuiltin) Description() string { return "Publish a message to a pub/sub topic." }
func (b *PublishBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *PublishBuiltin) Validate(a Action) error {
	if a.Topic == "" {
		return fmt.Errorf("builtin publish missing topic")
	}
	if a.Message == "" && a.Send == "" {
		return fmt.Errorf("builtin publish missing message")
	}
	return nil
}

func (b *PublishBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.topicManager == nil {
		return fmt.Errorf("builtin publish: topic manager not available")
	}
	topic, err := d.templateEngine.Execute("topic", a.Topic, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in topic expression: %w", err)
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return fmt.Errorf("builtin publish: topic evaluates to empty string")
	}

	msgContent := a.Message
	if msgContent == "" {
		msgContent = a.Send
	}

	msgStr, err := d.templateEngine.Execute("publish-msg", msgContent, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in publish message: %w", err)
	}
	delivered, err := d.topicManager.Publish(topic, &ws.Message{
		Type: ws.TextMessage,
		Data: []byte(msgStr),
	})
	if err != nil {
		return err
	}
	if d.verbose {
		d.errorf("  [handler] published to topic %q → %d clients\n", topic, delivered)
	}
	return nil
}

// KVSetBuiltin stores a value in the server's shared key-value store.
type KVSetBuiltin struct{}

func (b *KVSetBuiltin) Name() string        { return "kv-set" }
func (b *KVSetBuiltin) Description() string { return "Store a value in the server's shared key-value store." }
func (b *KVSetBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *KVSetBuiltin) Validate(a Action) error {
	if a.Key == "" {
		return fmt.Errorf("builtin kv-set missing key")
	}
	if a.Value == "" {
		return fmt.Errorf("builtin kv-set missing value")
	}
	return nil
}

func (b *KVSetBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.kvManager == nil {
		return fmt.Errorf("builtin kv-set: kv manager not available")
	}
	key, err := d.templateEngine.Execute("kv-key", a.Key, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in kv key: %w", err)
	}
	val, err := d.templateEngine.Execute("kv-value", a.Value, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in kv value: %w", err)
	}

	d.kvManager.SetKV(key, val)
	d.refreshKVSnapshot(tmplCtx)
	if d.verbose {
		d.errorf("  [handler] kv-set: %s = %v\n", key, val)
	}
	return nil
}

// KVGetBuiltin retrieves a value from the server's shared key-value store.
type KVGetBuiltin struct{}

func (b *KVGetBuiltin) Name() string        { return "kv-get" }
func (b *KVGetBuiltin) Description() string { return "Retrieve a value from the server's shared key-value store into .KvValue." }
func (b *KVGetBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *KVGetBuiltin) Validate(a Action) error {
	if a.Key == "" {
		return fmt.Errorf("builtin kv-get missing key")
	}
	return nil
}

func (b *KVGetBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.kvManager == nil {
		return fmt.Errorf("builtin kv-get: kv manager not available")
	}
	key, err := d.templateEngine.Execute("kv-key", a.Key, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in kv key: %w", err)
	}
	val, ok := d.kvManager.GetKV(key)
	if ok {
		tmplCtx.KvValue = val
	} else {
		tmplCtx.KvValue = nil
	}
	if d.verbose {
		d.errorf("  [handler] kv-get: %s = %v (found=%v)\n", key, val, ok)
	}
	return nil
}

// KVDelBuiltin deletes a key from the server's shared key-value store.
type KVDelBuiltin struct{}

func (b *KVDelBuiltin) Name() string        { return "kv-del" }
func (b *KVDelBuiltin) Description() string { return "Delete a key from the server's shared key-value store." }
func (b *KVDelBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *KVDelBuiltin) Validate(a Action) error {
	if a.Key == "" {
		return fmt.Errorf("builtin kv-del missing key")
	}
	return nil
}

func (b *KVDelBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.kvManager == nil {
		return fmt.Errorf("builtin kv-del: kv manager not available")
	}
	key, err := d.templateEngine.Execute("kv-key", a.Key, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in kv key: %w", err)
	}
	d.kvManager.DeleteKV(key)
	d.refreshKVSnapshot(tmplCtx)
	if d.verbose {
		d.errorf("  [handler] kv-del: %s\n", key)
	}
	return nil
}

// NoopBuiltin is a shared builtin that does nothing.
type NoopBuiltin struct{}

func (b *NoopBuiltin) Name() string        { return "noop" }
func (b *NoopBuiltin) Description() string { return "A shared builtin that does nothing (useful for testing)." }
func (b *NoopBuiltin) Scope() BuiltinScope { return Shared }

func (b *NoopBuiltin) Validate(a Action) error { return nil }

func (b *NoopBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.verbose {
		d.errorf("  [handler] builtin noop: doing nothing\n")
	}
	return nil
}

// EchoBuiltin reflects the incoming message back to the sender.
type EchoBuiltin struct{}

func (b *EchoBuiltin) Name() string        { return "echo" }
func (b *EchoBuiltin) Description() string { return "Reflect the incoming message back to the sender." }
func (b *EchoBuiltin) Scope() BuiltinScope { return Shared }

func (b *EchoBuiltin) Validate(a Action) error { return nil }

func (b *EchoBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// If a.Respond is set, ExecuteAction will handle sending the transformed message.
	// We only send the verbatim message here if no override is provided.
	if a.Respond != "" {
		if d.verbose {
			d.errorf("  [handler] builtin echo: respond override present, skipping verbatim echo\n")
		}
		return nil
	}

	if d.verbose {
		d.errorf("  [handler] builtin echo: reflecting original message\n")
	}

	// Determine original message type from context
	mt := ws.TextMessage
	if tmplCtx.MessageType == "binary" {
		mt = ws.BinaryMessage
	} else if tmplCtx.MessageType == "ping" {
		mt = ws.PingMessage
	} else if tmplCtx.MessageType == "pong" {
		mt = ws.PongMessage
	}

	return d.conn.Write(&ws.Message{
		Type: mt,
		Data: tmplCtx.MessageBytes,
	})
}
