package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/0funct0ry/xwebs/internal/observability"
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
	MustRegister(&KVListBuiltin{})
	MustRegister(&GateBuiltin{})
	MustRegister(&NoopBuiltin{})
	MustRegister(&EchoBuiltin{})
	MustRegister(&BroadcastBuiltin{})
	MustRegister(&BroadcastOthersBuiltin{})
	MustRegister(&ForwardBuiltin{conns: make(map[string]*ws.Connection)})
	MustRegister(&SequenceBuiltin{})
	MustRegister(&TemplateBuiltin{})
	MustRegister(&FileSendBuiltin{})
	MustRegister(&FileWriteBuiltin{})
	MustRegister(&RateLimitBuiltin{})
	MustRegister(&DelayBuiltin{})
	MustRegister(&DropBuiltin{})
	MustRegister(&CloseBuiltin{})
	MustRegister(&LogBuiltin{})
	MustRegister(&HttpBuiltin{})
	MustRegister(&MetricBuiltin{})
	MustRegister(&ThrottleBroadcastBuiltin{})
	MustRegister(&MulticastBuiltin{})
	MustRegister(&StickyBroadcastBuiltin{})
	MustRegister(&RoundRobinBuiltin{})
	MustRegister(&SampleBuiltin{})
}

// SubscribeBuiltin subscribes the current connection to a pub/sub topic.
type SubscribeBuiltin struct{}

func (b *SubscribeBuiltin) Name() string { return "subscribe" }
func (b *SubscribeBuiltin) Description() string {
	return "Subscribe the current connection to a pub/sub topic."
}
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

func (b *UnsubscribeBuiltin) Name() string { return "unsubscribe" }
func (b *UnsubscribeBuiltin) Description() string {
	return "Unsubscribe the current connection from a pub/sub topic."
}
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
	if a.Message == "" && a.Send == "" && a.Respond == "" {
		return fmt.Errorf("builtin publish missing message (provide message:, send:, or respond:)")
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
	if msgContent == "" {
		msgContent = a.Respond
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

func (b *KVSetBuiltin) Name() string { return "kv-set" }
func (b *KVSetBuiltin) Description() string {
	return "Store a value in the server's shared key-value store."
}
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

	ttl := time.Duration(0)
	if a.TTL != "" {
		ttlStr, err := d.templateEngine.Execute("kv-ttl", a.TTL, tmplCtx)
		if err == nil {
			if d, err := time.ParseDuration(ttlStr); err == nil {
				ttl = d
			}
		}
	}

	d.kvManager.SetKV(key, val, ttl)
	d.refreshKVSnapshot(tmplCtx)
	if d.verbose {
		if ttl > 0 {
			d.errorf("  [handler] kv-set: %s = %v (ttl: %v)\n", key, val, ttl)
		} else {
			d.errorf("  [handler] kv-set: %s = %v\n", key, val)
		}
	}
	return nil
}

// KVGetBuiltin retrieves a value from the server's shared key-value store.
type KVGetBuiltin struct{}

func (b *KVGetBuiltin) Name() string { return "kv-get" }
func (b *KVGetBuiltin) Description() string {
	return "Retrieve a value from the server's shared key-value store into .KvValue."
}
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

	defaultVal := ""
	if a.Default != "" {
		if dVal, err := d.templateEngine.Execute("kv-default", a.Default, tmplCtx); err == nil {
			defaultVal = dVal
		}
	}

	val, ok := d.kvManager.GetKV(key)
	if ok {
		tmplCtx.KvValue = val
	} else {
		tmplCtx.KvValue = defaultVal
	}
	if d.verbose {
		d.errorf("  [handler] kv-get: %s = %v (found=%v, default=%q)\n", key, tmplCtx.KvValue, ok, defaultVal)
	}
	return nil
}

// KVDelBuiltin deletes a key from the server's shared key-value store.
type KVDelBuiltin struct{}

func (b *KVDelBuiltin) Name() string { return "kv-del" }
func (b *KVDelBuiltin) Description() string {
	return "Delete a key from the server's shared key-value store."
}
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

// KVListBuiltin retrieves all keys from the server's shared key-value store.
type KVListBuiltin struct{}

func (b *KVListBuiltin) Name() string { return "kv-list" }
func (b *KVListBuiltin) Description() string {
	return "Retrieve all keys from the server's shared key-value store into .KvKeys."
}
func (b *KVListBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *KVListBuiltin) Validate(a Action) error { return nil }

func (b *KVListBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.kvManager == nil {
		return fmt.Errorf("builtin kv-list: kv manager not available")
	}

	kvMap := d.kvManager.ListKV()
	keys := make([]string, 0, len(kvMap))
	for k := range kvMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	tmplCtx.KvKeys = keys
	if d.verbose {
		d.errorf("  [handler] kv-list: found %d keys\n", len(keys))
	}
	return nil
}

// GateBuiltin checks a KV key before allowing a message to proceed.
type GateBuiltin struct{}

func (b *GateBuiltin) Name() string { return "gate" }
func (b *GateBuiltin) Description() string {
	return "Check a KV key against an expected value. Drops message if they don't match."
}
func (b *GateBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *GateBuiltin) Validate(a Action) error {
	if a.Key == "" {
		return fmt.Errorf("builtin gate missing key")
	}
	if a.Expect == "" {
		return fmt.Errorf("builtin gate missing expect")
	}
	return nil
}

func (b *GateBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.kvManager == nil {
		return fmt.Errorf("builtin gate: kv manager not available")
	}

	key, err := d.templateEngine.Execute("gate-key", a.Key, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in gate key: %w", err)
	}

	expect, err := d.templateEngine.Execute("gate-expect", a.Expect, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in gate expect: %w", err)
	}

	val, ok := d.kvManager.GetKV(key)
	valStr := fmt.Sprintf("%v", val)
	if !ok {
		valStr = ""
	}

	if valStr != expect {
		if d.verbose {
			d.errorf("  [handler] gate closed: %s (got: %q, expect: %q)\n", key, valStr, expect)
		}

		if a.OnClosed != "" {
			resp, err := d.templateEngine.Execute("on-closed", a.OnClosed, tmplCtx)
			if err != nil {
				d.errorf("  [handler] gate: error rendering on_closed template: %v\n", err)
			} else if resp != "" {
				_ = d.conn.Write(&ws.Message{
					Type: ws.TextMessage,
					Data: []byte(resp),
					Metadata: ws.MessageMetadata{
						Direction: "sent",
						Timestamp: time.Now(),
					},
				})
			}
		}

		return ErrDrop
	}

	if d.verbose {
		d.errorf("  [handler] gate open: %s (%q == %q)\n", key, valStr, expect)
	}

	return nil
}

// NoopBuiltin is a shared builtin that does nothing.
type NoopBuiltin struct{}

func (b *NoopBuiltin) Name() string { return "noop" }
func (b *NoopBuiltin) Description() string {
	return "A shared builtin that does nothing (useful for testing)."
}
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

// BroadcastBuiltin fouts a message to all connected clients.
type BroadcastBuiltin struct{}

func (b *BroadcastBuiltin) Name() string        { return "broadcast" }
func (b *BroadcastBuiltin) Description() string { return "Send a message to all connected clients." }
func (b *BroadcastBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *BroadcastBuiltin) Validate(a Action) error {
	return nil // message or respond is optional; defaults to incoming message
}

func (b *BroadcastBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.serverStats == nil {
		return fmt.Errorf("broadcast is only available in server mode")
	}

	msgContent := a.Message
	if msgContent == "" {
		msgContent = a.Send
	}

	var data []byte
	var msgType ws.MessageType

	if msgContent != "" {
		res, err := d.templateEngine.Execute("broadcast", msgContent, tmplCtx)
		if err != nil {
			return fmt.Errorf("rendering broadcast template: %w", err)
		}
		data = []byte(res)
		msgType = ws.TextMessage
	} else {
		// Default to original message content
		data = tmplCtx.MessageBytes
		mt := ws.TextMessage
		if tmplCtx.MessageType == "binary" {
			mt = ws.BinaryMessage
		} else if tmplCtx.MessageType == "ping" {
			mt = ws.PingMessage
		} else if tmplCtx.MessageType == "pong" {
			mt = ws.PongMessage
		}
		msgType = mt
	}

	broadcastMsg := &ws.Message{
		Type: msgType,
		Data: data,
		Metadata: ws.MessageMetadata{
			Direction: "sent",
			Timestamp: time.Now(),
		},
	}

	count := d.serverStats.Broadcast(broadcastMsg)
	tmplCtx.Stdout = fmt.Sprintf("Broadcasted to %d clients", count)
	if d.verbose {
		d.log("  [builtin:broadcast] delivered to %d clients\n", count)
	}

	return nil
}

// BroadcastOthersBuiltin fouts a message to all connected clients except the sender.
type BroadcastOthersBuiltin struct{}

func (b *BroadcastOthersBuiltin) Name() string { return "broadcast-others" }
func (b *BroadcastOthersBuiltin) Description() string {
	return "Send a message to all connected clients except the sender."
}
func (b *BroadcastOthersBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *BroadcastOthersBuiltin) Validate(a Action) error { return nil }

func (b *BroadcastOthersBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.serverStats == nil {
		return fmt.Errorf("broadcast-others is only available in server mode")
	}

	msgContent := a.Message
	if msgContent == "" {
		msgContent = a.Send
	}

	var data []byte
	var msgType ws.MessageType

	if msgContent != "" {
		res, err := d.templateEngine.Execute("broadcast-others", msgContent, tmplCtx)
		if err != nil {
			return fmt.Errorf("rendering broadcast template: %w", err)
		}
		data = []byte(res)
		msgType = ws.TextMessage
	} else {
		data = tmplCtx.MessageBytes
		mt := ws.TextMessage
		if tmplCtx.MessageType == "binary" {
			mt = ws.BinaryMessage
		} else if tmplCtx.MessageType == "ping" {
			mt = ws.PingMessage
		} else if tmplCtx.MessageType == "pong" {
			mt = ws.PongMessage
		}
		msgType = mt
	}

	broadcastMsg := &ws.Message{
		Type: msgType,
		Data: data,
		Metadata: ws.MessageMetadata{
			Direction: "sent",
			Timestamp: time.Now(),
		},
	}

	excludeID := ""
	if d.conn != nil {
		excludeID = d.conn.GetID()
	}

	count := d.serverStats.Broadcast(broadcastMsg, excludeID)
	tmplCtx.Stdout = fmt.Sprintf("Broadcasted to %d other clients", count)
	if d.verbose {
		d.log("  [builtin:broadcast-others] delivered to %d clients\n", count)
	}

	return nil
}

// SequenceBuiltin cycles through a list of responses.
type SequenceBuiltin struct{}

func (b *SequenceBuiltin) Name() string        { return "sequence" }
func (b *SequenceBuiltin) Description() string { return "Cycle through a list of responses in order." }
func (b *SequenceBuiltin) Scope() BuiltinScope { return Shared }

func (b *SequenceBuiltin) Validate(a Action) error {
	if len(a.Responses) == 0 {
		return fmt.Errorf("builtin sequence: responses list is required and cannot be empty")
	}
	return nil
}

func (b *SequenceBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	key := a.HandlerName
	if key == "" {
		key = "anonymous-sequence"
	}

	idx := d.registry.GetNextSequenceIndex(key, d.conn.GetID(), len(a.Responses), a.Loop, a.PerClient)
	respTmpl := a.Responses[idx]

	resp, err := d.templateEngine.Execute("sequence", respTmpl, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering sequence response template: %w", err)
	}

	if d.verbose {
		d.errorf("  [handler] sequence: sending item %d/%d (handler=%q): %q\n", idx+1, len(a.Responses), key, resp)
	}

	return d.conn.Write(&ws.Message{
		Type: ws.TextMessage,
		Data: []byte(resp),
		Metadata: ws.MessageMetadata{
			Direction: "sent",
			Timestamp: time.Now(),
		},
	})
}

// TemplateBuiltin renders a response from an external template file.
type TemplateBuiltin struct{}

func (b *TemplateBuiltin) Name() string { return "template" }
func (b *TemplateBuiltin) Description() string {
	return "Render a response from an external template file."
}
func (b *TemplateBuiltin) Scope() BuiltinScope { return Shared }

func (b *TemplateBuiltin) Validate(a Action) error {
	if a.File == "" {
		return fmt.Errorf("builtin template missing file path")
	}
	return nil
}

func (b *TemplateBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Resolve file path (render if template)
	filePath, err := d.templateEngine.Execute("file-path", a.File, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in file path expression: %w", err)
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return fmt.Errorf("builtin template: file path evaluates to empty string")
	}

	// 2. Resolve relative path using BaseDir from Action (calculated in Dispatcher)
	resolvedPath := filePath
	if !filepath.IsAbs(filePath) && a.BaseDir != "" {
		resolvedPath = filepath.Join(a.BaseDir, filePath)
	}

	// 3. Read file content at execution time
	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("builtin template: file not found: %s (resolved: %s)", filePath, resolvedPath)
		}
		return fmt.Errorf("builtin template: error reading file %s: %w", resolvedPath, err)
	}

	// 4. Render file content as a template
	resp, err := d.templateEngine.Execute(filePath, string(content), tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering template file %s: %w", filePath, err)
	}

	if d.verbose {
		d.errorf("  [handler] template: rendered %s (resolved: %s)\n", filePath, resolvedPath)
	}

	// 5. Send response
	return d.conn.Write(&ws.Message{
		Type: ws.TextMessage,
		Data: []byte(resp),
		Metadata: ws.MessageMetadata{
			Direction: "sent",
			Timestamp: time.Now(),
		},
	})
}

// FileSendBuiltin sends the contents of a local file as a WebSocket message.
type FileSendBuiltin struct{}

func (b *FileSendBuiltin) Name() string { return "file-send" }
func (b *FileSendBuiltin) Description() string {
	return "Send the contents of a local file as a WebSocket message."
}
func (b *FileSendBuiltin) Scope() BuiltinScope { return ClientOnly }

func (b *FileSendBuiltin) Validate(a Action) error {
	if a.File == "" {
		return fmt.Errorf("builtin file-send missing file path")
	}
	if a.Mode != "" {
		m := strings.ToLower(a.Mode)
		if m != "text" && m != "binary" {
			return fmt.Errorf("builtin file-send: invalid mode %q (must be 'text' or 'binary')", a.Mode)
		}
	}
	return nil
}

func (b *FileSendBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Resolve file path (render if template)
	filePath, err := d.templateEngine.Execute("file-path", a.File, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in file path expression: %w", err)
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return fmt.Errorf("builtin file-send: file path evaluates to empty string")
	}

	// 2. Resolve relative path using BaseDir from Action
	resolvedPath := filePath
	if !filepath.IsAbs(filePath) && a.BaseDir != "" {
		resolvedPath = filepath.Join(a.BaseDir, filePath)
	}

	// 3. Read file content
	content, err := os.ReadFile(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("builtin file-send: file not found: %s (resolved: %s)", filePath, resolvedPath)
		}
		return fmt.Errorf("builtin file-send: error reading file %s: %w", resolvedPath, err)
	}

	// 4. Determine message type
	mt := ws.TextMessage
	if strings.ToLower(a.Mode) == "binary" {
		mt = ws.BinaryMessage
	}

	if d.verbose {
		d.errorf("  [handler] file-send: sending %s as %s (resolved: %s, size: %d bytes)\n",
			filePath, strings.ToLower(a.Mode), resolvedPath, len(content))
	}

	// 5. Send message
	return d.conn.Write(&ws.Message{
		Type: mt,
		Data: content,
		Metadata: ws.MessageMetadata{
			Direction: "sent",
			Timestamp: time.Now(),
		},
	})
}

// FileWriteBuiltin writes a message (or a rendered template) to a local file.
type FileWriteBuiltin struct{}

func (b *FileWriteBuiltin) Name() string { return "file-write" }
func (b *FileWriteBuiltin) Description() string {
	return "Write the message or a template-rendered variant to a file."
}
func (b *FileWriteBuiltin) Scope() BuiltinScope { return Shared }

func (b *FileWriteBuiltin) Validate(a Action) error {
	if a.Path == "" {
		return fmt.Errorf("builtin file-write missing path")
	}
	if a.Mode != "" {
		m := strings.ToLower(a.Mode)
		if m != "overwrite" && m != "append" {
			return fmt.Errorf("builtin file-write: invalid mode %q (must be 'overwrite' or 'append')", a.Mode)
		}
	}
	return nil
}

func (b *FileWriteBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Resolve path (render if template)
	filePath, err := d.templateEngine.Execute("file-path", a.Path, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in file path expression: %w", err)
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return fmt.Errorf("builtin file-write: file path evaluates to empty string")
	}

	// 2. Resolve relative path using BaseDir from Action
	resolvedPath := filePath
	if !filepath.IsAbs(filePath) && a.BaseDir != "" {
		resolvedPath = filepath.Join(a.BaseDir, filePath)
	}

	// 3. Resolve content
	var content []byte
	if a.Content != "" {
		res, err := d.templateEngine.Execute("file-content", a.Content, tmplCtx)
		if err != nil {
			return fmt.Errorf("template error in file content expression: %w", err)
		}
		content = []byte(res)
	} else {
		// Use original message content
		content = tmplCtx.MessageBytes
	}

	// 4. Create parent directories
	dir := filepath.Dir(resolvedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// 5. Open file
	flags := os.O_CREATE | os.O_WRONLY
	if strings.ToLower(a.Mode) == "append" {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	f, err := os.OpenFile(resolvedPath, flags, 0644)
	if err != nil {
		return fmt.Errorf("opening file %s: %w", resolvedPath, err)
	}
	defer f.Close()

	// 6. Write content
	if _, err := f.Write(content); err != nil {
		return fmt.Errorf("writing to file %s: %w", resolvedPath, err)
	}

	if d.verbose {
		d.errorf("  [handler] file-write: wrote %d bytes to %s (mode=%s, resolved=%s)\n",
			len(content), filePath, strings.ToLower(a.Mode), resolvedPath)
	}

	return nil
}

// RateLimitBuiltin enforces a per-client, global, or handler-level message rate.
type RateLimitBuiltin struct{}

func (b *RateLimitBuiltin) Name() string { return "rate-limit" }
func (b *RateLimitBuiltin) Description() string {
	return "Enforce a per-client, global, or handler-level message rate."
}
func (b *RateLimitBuiltin) Scope() BuiltinScope { return Shared }

func (b *RateLimitBuiltin) Validate(a Action) error {
	if a.Rate == "" {
		return fmt.Errorf("builtin rate-limit: missing 'rate'")
	}
	if a.Scope != "" {
		s := strings.ToLower(a.Scope)
		if s != "client" && s != "global" && s != "handler" {
			return fmt.Errorf("builtin rate-limit: invalid scope %q (must be 'client', 'global', or 'handler')", a.Scope)
		}
	}
	return nil
}

func (b *RateLimitBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Evaluate rate template
	rateStr, err := d.templateEngine.Execute("rate", a.Rate, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in rate expression: %w", err)
	}
	rateStr = strings.TrimSpace(rateStr)
	if rateStr == "" {
		return fmt.Errorf("builtin rate-limit: rate evaluates to empty string")
	}

	// 2. Determine scope and key
	scope := strings.ToLower(a.Scope)
	if scope == "" {
		scope = "client"
	}

	var key string
	handlerName := a.HandlerName
	if handlerName == "" {
		handlerName = "anon"
	}

	switch scope {
	case "global":
		key = "global:ratelimit" // Shared by all rate-limit actions using "global" scope
	case "handler":
		key = "handler:" + handlerName
	case "client":
		key = "client:" + handlerName + ":" + d.conn.GetID()
	}

	// 3. Get or create limiter
	limiter := d.registry.GetScopedLimiter(key, rateStr, a.Burst)
	if limiter == nil {
		return fmt.Errorf("builtin rate-limit: failed to create limiter for %q", key)
	}

	// 4. Check limit
	if !limiter.Allow() {
		// Calculate retry after without consuming a token permanently
		reserve := limiter.Reserve()
		delay := reserve.Delay()
		reserve.Cancel()

		// Populate template variables
		retryAfter := delay.Seconds()
		if retryAfter <= 0 {
			// If not allowed but delay is 0, it means the rate is effectively 0 or something went wrong.
			// Default to a small value if we can't calculate a future slot.
			retryAfter = 1.0
		}

		tmplCtx.RetryAfter = retryAfter
		tmplCtx.RetryAfterMs = int64(retryAfter * 1000)
		tmplCtx.RateLimit = rateStr
		tmplCtx.LimitScope = scope

		if d.verbose {
			d.errorf("  [handler] rate-limit triggered: %s (scope=%s, key=%s, retry_after=%.2fs)\n",
				rateStr, scope, key, retryAfter)
		}

		// 5. Handle rejection response
		if a.OnLimit != "" {
			resp, err := d.templateEngine.Execute("on-limit", a.OnLimit, tmplCtx)
			if err != nil {
				d.errorf("  [handler] rate-limit: error rendering on_limit template: %v\n", err)
			} else {
				_ = d.conn.Write(&ws.Message{
					Type: ws.TextMessage,
					Data: []byte(resp),
					Metadata: ws.MessageMetadata{
						Direction: "sent",
						Timestamp: time.Now(),
					},
				})
			}
		}

		// 6. Short-circuit
		return ErrLimitExceeded
	}

	return nil
}

// DelayBuiltin pauses handler execution for a configurable duration, optionally
// capped by a max: value to guard against malicious or misconfigured inputs.
// After the delay, any respond: template is sent naturally by ExecuteAction.
type DelayBuiltin struct{}

func (b *DelayBuiltin) Name() string { return "delay" }
func (b *DelayBuiltin) Description() string {
	return "Pause handler execution for a configurable duration before sending a response."
}
func (b *DelayBuiltin) Scope() BuiltinScope { return Shared }

func (b *DelayBuiltin) Validate(a Action) error {
	if a.Duration == "" {
		return fmt.Errorf("builtin delay: missing 'duration'")
	}
	// Static duration strings are validated at config-load time.
	// Template expressions are only resolvable at runtime, so we skip them here.
	if !strings.Contains(a.Duration, "{{") {
		if _, err := time.ParseDuration(a.Duration); err != nil {
			return fmt.Errorf("builtin delay: invalid duration %q: %w", a.Duration, err)
		}
	}
	if a.Max != "" && !strings.Contains(a.Max, "{{") {
		if _, err := time.ParseDuration(a.Max); err != nil {
			return fmt.Errorf("builtin delay: invalid max %q: %w", a.Max, err)
		}
	}
	return nil
}

func (b *DelayBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Render the duration field (may be a template expression)
	durStr, err := d.templateEngine.Execute("delay-duration", a.Duration, tmplCtx)
	if err != nil {
		return fmt.Errorf("builtin delay: template error in duration: %w", err)
	}
	durStr = strings.TrimSpace(durStr)
	if durStr == "" {
		return fmt.Errorf("builtin delay: duration evaluates to empty string")
	}

	dur, err := time.ParseDuration(durStr)
	if err != nil {
		return fmt.Errorf("builtin delay: invalid duration %q: %w", durStr, err)
	}
	if dur < 0 {
		return fmt.Errorf("builtin delay: duration must be non-negative, got %v", dur)
	}

	// 2. Apply the max cap if specified
	if a.Max != "" {
		maxStr, err := d.templateEngine.Execute("delay-max", a.Max, tmplCtx)
		if err == nil {
			maxStr = strings.TrimSpace(maxStr)
			if maxDur, err := time.ParseDuration(maxStr); err == nil && maxDur > 0 && dur > maxDur {
				if d.verbose {
					d.errorf("  [handler] builtin delay: capping %v to max %v\n", dur, maxDur)
				}
				dur = maxDur
			}
		}
	}

	if d.verbose {
		d.errorf("  [handler] builtin delay: sleeping for %v\n", dur)
	}

	// 3. Sleep, honouring context cancellation so other connections are not affected
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(dur):
		// Delay elapsed — respond: (if any) is handled by ExecuteAction
	}

	return nil
}

// DropBuiltin silently discards the matched message and stops further processing.
type DropBuiltin struct{}

func (b *DropBuiltin) Name() string { return "drop" }
func (b *DropBuiltin) Description() string {
	return "Silently discard the message and stop further handlers."
}
func (b *DropBuiltin) Scope() BuiltinScope { return Shared }

func (b *DropBuiltin) Validate(a Action) error { return nil }

func (b *DropBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.verbose {
		d.errorf("  [handler] builtin drop: discarding message\n")
	}
	return ErrDrop
}

// CloseBuiltin terminates the current connection with an optional code and reason.
type CloseBuiltin struct{}

func (b *CloseBuiltin) Name() string { return "close" }
func (b *CloseBuiltin) Description() string {
	return "Terminate the connection with an optional code and reason."
}
func (b *CloseBuiltin) Scope() BuiltinScope { return Shared }

func (b *CloseBuiltin) Validate(a Action) error { return nil }

func (b *CloseBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	code := 1000
	if a.Code != "" {
		codeStr, err := d.templateEngine.Execute("close-code", a.Code, tmplCtx)
		if err == nil {
			if c, err := strconv.Atoi(strings.TrimSpace(codeStr)); err == nil {
				code = c
			} else {
				d.errorf("  [handler] warning: invalid close code %q: %v\n", codeStr, err)
			}
		}
	}

	reason := ""
	if a.Reason != "" {
		res, err := d.templateEngine.Execute("close-reason", a.Reason, tmplCtx)
		if err == nil {
			reason = res
		}
	}

	if d.verbose {
		d.errorf("  [handler] builtin close: closing connection (code=%d, reason=%q)\n", code, reason)
	}

	return d.conn.CloseWithCode(code, reason)
}

// HttpBuiltin makes an outbound HTTP request.
type HttpBuiltin struct{}

func (b *HttpBuiltin) Name() string { return "http" }
func (b *HttpBuiltin) Description() string {
	return "Make an outbound HTTP request."
}
func (b *HttpBuiltin) Scope() BuiltinScope { return Shared }

func (b *HttpBuiltin) Validate(a Action) error {
	if a.URL == "" {
		return fmt.Errorf("builtin http missing url")
	}
	return nil
}

func (b *HttpBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Resolve URL
	urlStr, err := d.templateEngine.Execute("http-url", a.URL, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in url expression: %w", err)
	}
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return fmt.Errorf("builtin http: url evaluates to empty string")
	}

	// 2. Resolve Method (default GET)
	method := "GET"
	if a.Method != "" {
		m, err := d.templateEngine.Execute("http-method", a.Method, tmplCtx)
		if err == nil && m != "" {
			method = strings.ToUpper(strings.TrimSpace(m))
		}
	}

	// 3. Resolve Body
	var bodyReader io.Reader
	if a.Body != "" {
		bodyStr, err := d.templateEngine.Execute("http-body", a.Body, tmplCtx)
		if err != nil {
			return fmt.Errorf("template error in body expression: %w", err)
		}
		bodyReader = strings.NewReader(bodyStr)
	}

	// 4. Create Request
	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	// 5. Add Headers
	if a.Headers != nil {
		for k, v := range a.Headers {
			evalK, _ := d.templateEngine.Execute("http-header-key", k, tmplCtx)
			evalV, _ := d.templateEngine.Execute("http-header-value", v, tmplCtx)
			if evalK != "" {
				req.Header.Set(evalK, evalV)
			}
		}
	}

	// 6. Set Timeout
	timeout := 10 * time.Second
	if a.Timeout != "" {
		if t, err := time.ParseDuration(a.Timeout); err == nil {
			timeout = t
		}
	}

	client := &http.Client{
		Timeout: timeout,
	}

	// 7. Execute Request
	if d.verbose {
		d.errorf("  [handler] http: %s %s (timeout: %v)\n", method, urlStr, timeout)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Network errors trigger on_error (by returning error here)
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// 8. Read Response Body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read http response body: %w", err)
	}

	// 9. Update Context
	tmplCtx.HttpStatus = resp.StatusCode
	tmplCtx.HttpBody = string(respBody)

	if d.verbose {
		d.errorf("  [handler] http: received status %d (%d bytes)\n", resp.StatusCode, len(respBody))
	}

	return nil
}

// LogBuiltin writes a structured log entry to stdout, a file, or both.
type LogBuiltin struct{}

func (b *LogBuiltin) Name() string { return "log" }
func (b *LogBuiltin) Description() string {
	return "Write a structured JSONL log entry to stdout, a file, or both."
}
func (b *LogBuiltin) Scope() BuiltinScope { return Shared }

func (b *LogBuiltin) Validate(a Action) error {
	if a.Message == "" {
		return fmt.Errorf("builtin log: missing 'message'")
	}
	target := strings.ToLower(a.Target)
	if target == "" {
		target = "stdout" // default
	}
	if target != "stdout" && target != "file" && target != "both" {
		return fmt.Errorf("builtin log: invalid target %q (must be 'stdout', 'file', or 'both')", a.Target)
	}
	if (target == "file" || target == "both") && a.Path == "" {
		return fmt.Errorf("builtin log: missing 'path' for file-based logging")
	}
	return nil
}

func (b *LogBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Render message
	msgStr, err := d.templateEngine.Execute("log-message", a.Message, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering log message: %w", err)
	}

	// 2. Prepare log entry
	connID := tmplCtx.ConnectionID
	if connID == "" && d.conn != nil {
		connID = d.conn.GetID()
	}

	entry := struct {
		Timestamp    string      `json:"timestamp"`
		ConnectionID string      `json:"conn_id"`
		Message      string      `json:"message"`
		Metadata     interface{} `json:"metadata,omitempty"`
	}{
		Timestamp:    time.Now().Format(time.RFC3339),
		ConnectionID: connID,
		Message:      msgStr,
	}

	// Optional: add some metadata if available
	if tmplCtx.Msg != nil {
		entry.Metadata = map[string]interface{}{
			"type":      tmplCtx.MessageType,
			"direction": tmplCtx.Direction,
		}
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling log entry: %w", err)
	}
	jsonBytes = append(jsonBytes, '\n')

	// 3. Write to targets
	target := strings.ToLower(a.Target)
	if target == "" {
		target = "stdout"
	}

	if target == "stdout" || target == "both" {
		d.log("%s", string(jsonBytes))
	}

	if target == "file" || target == "both" {
		// Resolve path (render if template)
		filePath, err := d.templateEngine.Execute("log-path", a.Path, tmplCtx)
		if err != nil {
			return fmt.Errorf("rendering log path: %w", err)
		}
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			return fmt.Errorf("builtin log: path evaluates to empty string")
		}

		// Resolve relative path
		resolvedPath := filePath
		if !filepath.IsAbs(filePath) && a.BaseDir != "" {
			resolvedPath = filepath.Join(a.BaseDir, filePath)
		}

		// Ensure directory exists
		dir := filepath.Dir(resolvedPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating log directory %s: %w", dir, err)
		}

		// Append to file
		f, err := os.OpenFile(resolvedPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("opening log file %s: %w", resolvedPath, err)
		}
		defer f.Close()

		if _, err := f.Write(jsonBytes); err != nil {
			return fmt.Errorf("writing to log file %s: %w", resolvedPath, err)
		}
	}

	return nil
}

// MetricBuiltin increments a Prometheus counter with dynamic name and labels.
type MetricBuiltin struct{}

func (b *MetricBuiltin) Name() string        { return "metric" }
func (b *MetricBuiltin) Description() string { return "Increment a Prometheus counter." }
func (b *MetricBuiltin) Scope() BuiltinScope { return Shared }

func (b *MetricBuiltin) Validate(a Action) error {
	if a.Name == "" {
		return fmt.Errorf("builtin metric: missing 'name'")
	}
	return nil
}

func (b *MetricBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Resolve metric name (render if template)
	metricName, err := d.templateEngine.Execute("metric-name", a.Name, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in metric name expression: %w", err)
	}
	metricName = strings.TrimSpace(metricName)
	if metricName == "" {
		return fmt.Errorf("builtin metric: name evaluates to empty string")
	}

	// 2. Resolve labels
	labels := make(map[string]string)
	for k, vTmpl := range a.Labels {
		val, err := d.templateEngine.Execute("metric-label-"+k, vTmpl, tmplCtx)
		if err != nil {
			return fmt.Errorf("template error in label %q expression: %w", k, err)
		}
		labels[k] = val
	}

	// 3. Increment counter
	observability.IncrementCounter(metricName, labels)

	if d.verbose {
		labelStr := ""
		if len(labels) > 0 {
			labelParts := make([]string, 0, len(labels))
			for k, v := range labels {
				labelParts = append(labelParts, fmt.Sprintf("%s=%q", k, v))
			}
			sort.Strings(labelParts)
			labelStr = " {" + strings.Join(labelParts, ", ") + "}"
		}
		d.errorf("  [handler] metric: incremented %s%s\n", metricName, labelStr)
	}

	return nil
}

// ThrottleBroadcastBuiltin delivers messages to all clients except those who
// received a message from this handler within the last window: duration.
type ThrottleBroadcastBuiltin struct{}

func (b *ThrottleBroadcastBuiltin) Name() string { return "throttle-broadcast" }
func (b *ThrottleBroadcastBuiltin) Description() string {
	return "Deliver a message to all clients except those who received one from this handler too recently."
}
func (b *ThrottleBroadcastBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *ThrottleBroadcastBuiltin) Validate(a Action) error {
	if a.Window == "" {
		return fmt.Errorf("builtin throttle-broadcast: missing 'window'")
	}
	return nil
}

func (b *ThrottleBroadcastBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.serverStats == nil {
		return fmt.Errorf("throttle-broadcast is only available in server mode")
	}

	// 1. Resolve window duration
	windowStr, err := d.templateEngine.Execute("window", a.Window, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering window template: %w", err)
	}
	window, err := time.ParseDuration(windowStr)
	if err != nil {
		return fmt.Errorf("invalid window duration %q: %w", windowStr, err)
	}

	// 2. Resolve message content
	msgContent := a.Message
	if msgContent == "" {
		msgContent = a.Send
	}

	var data []byte
	var msgType ws.MessageType

	if msgContent != "" {
		res, err := d.templateEngine.Execute("broadcast", msgContent, tmplCtx)
		if err != nil {
			return fmt.Errorf("rendering broadcast template: %w", err)
		}
		data = []byte(res)
		msgType = ws.TextMessage
	} else {
		// Default to original message content
		data = tmplCtx.MessageBytes
		mt := ws.TextMessage
		if tmplCtx.MessageType == "binary" {
			mt = ws.BinaryMessage
		} else if tmplCtx.MessageType == "ping" {
			mt = ws.PingMessage
		} else if tmplCtx.MessageType == "pong" {
			mt = ws.PongMessage
		}
		msgType = mt
	}

	broadcastMsg := &ws.Message{
		Type: msgType,
		Data: data,
		Metadata: ws.MessageMetadata{
			Direction: "sent",
			Timestamp: time.Now(),
		},
	}

	// 3. Filter clients and broadcast
	clients := d.serverStats.GetClients()
	deliveredCount := 0
	skippedCount := 0
	now := time.Now()

	for _, client := range clients {
		lastSent := d.registry.GetLastThrottleBroadcast(a.HandlerName, client.ID)
		if !lastSent.IsZero() && now.Sub(lastSent) < window {
			skippedCount++
			continue
		}

		if err := d.serverStats.Send(client.ID, broadcastMsg); err == nil {
			deliveredCount++
			d.registry.SetLastThrottleBroadcast(a.HandlerName, client.ID, now)
		} else {
			// Delivery failure, we don't count it as skipped but we don't update timestamp
			if d.verbose {
				d.errorf("  [builtin:throttle-broadcast] failed to deliver to %s: %v\n", client.ID, err)
			}
		}
	}

	tmplCtx.DeliveredCount = deliveredCount
	tmplCtx.SkippedCount = skippedCount
	tmplCtx.Stdout = fmt.Sprintf("Broadcasted to %d clients, skipped %d", deliveredCount, skippedCount)

	if d.verbose {
		d.log("  [builtin:throttle-broadcast] delivered to %d clients, skipped %d\n", deliveredCount, skippedCount)
	}

	return nil
}

// MulticastBuiltin sends a message to a specific list of client IDs.
type MulticastBuiltin struct{}

func (b *MulticastBuiltin) Name() string { return "multicast" }
func (b *MulticastBuiltin) Description() string {
	return "Send a message to a specific list of client IDs."
}
func (b *MulticastBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *MulticastBuiltin) Validate(a Action) error {
	if a.Targets == "" {
		return fmt.Errorf("builtin multicast: missing 'targets'")
	}
	return nil
}

func (b *MulticastBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.serverStats == nil {
		return fmt.Errorf("multicast is only available in server mode")
	}

	// 1. Resolve targets (template expression evaluating to JSON array)
	targetsStr, err := d.templateEngine.Execute("multicast-targets", a.Targets, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering targets template: %w", err)
	}

	var targetIDs []string
	if err := json.Unmarshal([]byte(targetsStr), &targetIDs); err != nil {
		// Fallback: if it's not a JSON array, maybe it's a single ID or comma-separated
		targetsStr = strings.TrimSpace(targetsStr)
		if strings.HasPrefix(targetsStr, "[") {
			return fmt.Errorf("invalid JSON array in targets: %w", err)
		}
		if targetsStr != "" {
			for _, id := range strings.Split(targetsStr, ",") {
				if trimmed := strings.TrimSpace(id); trimmed != "" {
					targetIDs = append(targetIDs, trimmed)
				}
			}
		}
	}

	if len(targetIDs) == 0 {
		tmplCtx.DeliveredCount = 0
		tmplCtx.SkippedCount = 0
		if d.verbose {
			d.log("  [builtin:multicast] no targets specified\n")
		}
		return nil
	}

	// 2. Resolve message content
	msgContent := a.Message
	if msgContent == "" {
		msgContent = a.Send
	}

	var data []byte
	var msgType ws.MessageType

	if msgContent != "" {
		res, err := d.templateEngine.Execute("multicast-msg", msgContent, tmplCtx)
		if err != nil {
			return fmt.Errorf("rendering multicast template: %w", err)
		}
		data = []byte(res)
		msgType = ws.TextMessage
	} else {
		data = tmplCtx.MessageBytes
		mt := ws.TextMessage
		if tmplCtx.MessageType == "binary" {
			mt = ws.BinaryMessage
		} else if tmplCtx.MessageType == "ping" {
			mt = ws.PingMessage
		} else if tmplCtx.MessageType == "pong" {
			mt = ws.PongMessage
		}
		msgType = mt
	}

	multicastMsg := &ws.Message{
		Type: msgType,
		Data: data,
		Metadata: ws.MessageMetadata{
			Direction: "sent",
			Timestamp: time.Now(),
		},
	}

	// 3. Deliver to targets
	deliveredCount := 0
	skippedCount := 0

	for _, id := range targetIDs {
		err := d.serverStats.Send(id, multicastMsg)
		if err == nil {
			deliveredCount++
		} else {
			// In server mode, Send usually fails if the ID is not found in registry
			skippedCount++
			if d.verbose {
				d.errorf("  [builtin:multicast] failed to deliver to %s: %v\n", id, err)
			}
		}
	}

	tmplCtx.DeliveredCount = deliveredCount
	tmplCtx.SkippedCount = skippedCount
	tmplCtx.Stdout = fmt.Sprintf("Multicast to %d clients, skipped %d", deliveredCount, skippedCount)

	if d.verbose {
		d.log("  [builtin:multicast] delivered to %d clients, skipped %d\n", deliveredCount, skippedCount)
	}

	return nil
}

// StickyBroadcastBuiltin broadcasts a message to all current subscribers of a topic
// and stores it as the retained value for that topic.
type StickyBroadcastBuiltin struct{}

func (b *StickyBroadcastBuiltin) Name() string        { return "sticky-broadcast" }
func (b *StickyBroadcastBuiltin) Description() string { return "Broadcast and retain a message for a topic." }
func (b *StickyBroadcastBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *StickyBroadcastBuiltin) Validate(a Action) error {
	if a.Topic == "" {
		return fmt.Errorf("builtin sticky-broadcast missing topic")
	}
	if a.Message == "" && a.Send == "" && a.Respond == "" {
		return fmt.Errorf("builtin sticky-broadcast missing message (provide message:, send:, or respond:)")
	}
	return nil
}

func (b *StickyBroadcastBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.topicManager == nil {
		return fmt.Errorf("builtin sticky-broadcast: topic manager not available")
	}
	topic, err := d.templateEngine.Execute("topic", a.Topic, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in topic expression: %w", err)
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return fmt.Errorf("builtin sticky-broadcast: topic evaluates to empty string")
	}

	msgContent := a.Message
	if msgContent == "" {
		msgContent = a.Send
	}
	if msgContent == "" {
		msgContent = a.Respond
	}

	msgStr, err := d.templateEngine.Execute("sticky-broadcast-msg", msgContent, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in sticky-broadcast message: %w", err)
	}

	msg := &ws.Message{
		Type: ws.TextMessage,
		Data: []byte(msgStr),
		Metadata: ws.MessageMetadata{
			Direction: "sent",
			Timestamp: time.Now(),
		},
	}

	delivered, err := d.topicManager.PublishSticky(topic, msg)
	if err != nil {
		return err
	}

	tmplCtx.Retained = msgStr
	if d.verbose {
		d.errorf("  [handler] sticky-broadcast to topic %q → %d clients (retained value set)\n", topic, delivered)
	}
	return nil
}

// RoundRobinBuiltin distributes messages across a pool of client IDs.
type RoundRobinBuiltin struct{}

func (b *RoundRobinBuiltin) Name() string        { return "round-robin" }
func (b *RoundRobinBuiltin) Description() string { return "Cycle messages across a pool of client IDs." }
func (b *RoundRobinBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *RoundRobinBuiltin) Validate(a Action) error {
	if a.Pool == "" {
		return fmt.Errorf("builtin round-robin: missing 'pool'")
	}
	return nil
}

func (b *RoundRobinBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.serverStats == nil {
		return fmt.Errorf("builtin round-robin: only available in server mode")
	}

	// 1. Evaluate Pool template
	poolStr, err := d.templateEngine.Execute("round-robin-pool", a.Pool, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in pool expression: %w", err)
	}

	// 2. Parse Pool as JSON array
	var pool []string
	if err := json.Unmarshal([]byte(poolStr), &pool); err != nil {
		// Fallback: try comma-separated if not JSON array
		if strings.Contains(poolStr, ",") {
			parts := strings.Split(poolStr, ",")
			for _, p := range parts {
				pool = append(pool, strings.TrimSpace(p))
			}
		} else if strings.TrimSpace(poolStr) != "" {
			pool = []string{strings.TrimSpace(poolStr)}
		}
	}

	if len(pool) == 0 {
		return b.handleEmpty(d, a, tmplCtx)
	}

	// 3. Get next starting index from Registry
	key := a.HandlerName + ":round-robin:" + a.Pool
	idx := d.registry.GetRoundRobinIndex(key, len(pool))

	// 4. Resolve message content
	msgContent := a.Message
	if msgContent == "" {
		msgContent = a.Send
	}
	if msgContent == "" {
		msgContent = a.Respond
	}

	var data []byte
	var msgType ws.MessageType
	if msgContent != "" {
		res, err := d.templateEngine.Execute("round-robin-msg", msgContent, tmplCtx)
		if err != nil {
			return fmt.Errorf("rendering round-robin message template: %w", err)
		}
		data = []byte(res)
		msgType = ws.TextMessage
	} else {
		data = tmplCtx.MessageBytes
		msgType = ws.TextMessage
		if tmplCtx.MessageType == "binary" {
			msgType = ws.BinaryMessage
		}
	}

	rrMsg := &ws.Message{
		Type: msgType,
		Data: data,
		Metadata: ws.MessageMetadata{
			Direction: "sent",
			Timestamp: time.Now(),
		},
	}

	// 5. Try clients in the pool starting from idx
	clients := d.serverStats.GetClients()
	connectedMap := make(map[string]bool)
	for _, c := range clients {
		connectedMap[c.ID] = true
	}

	for i := 0; i < len(pool); i++ {
		tryIdx := (idx + i) % len(pool)
		clientID := pool[tryIdx]

		if connectedMap[clientID] {
			if d.verbose {
				d.errorf("  [handler] round-robin: sending to client %q (pool index %d)\n", clientID, tryIdx)
			}
			if err := d.serverStats.Send(clientID, rrMsg); err != nil {
				d.errorf("  [handler] round-robin error sending to %q: %v\n", clientID, err)
				continue // Try next one if send fails
			}
			// Update index to the one *after* the successful one for next time
			d.registry.SetRoundRobinIndex(key, (tryIdx+1)%len(pool))
			return nil
		}
	}

	// 6. All failed
	return b.handleEmpty(d, a, tmplCtx)
}

func (b *RoundRobinBuiltin) handleEmpty(d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if a.OnEmpty == "" {
		if d.verbose {
			d.errorf("  [handler] round-robin: pool is empty or all clients disconnected, no on_empty provided\n")
		}
		return nil
	}

	res, err := d.templateEngine.Execute("round-robin-empty", a.OnEmpty, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering on_empty template: %w", err)
	}

	if res == "" {
		return nil
	}

	if d.verbose {
		d.errorf("  [handler] round-robin: all disconnected, sending on_empty response to sender\n")
	}

	return d.conn.Write(&ws.Message{
		Type: ws.TextMessage,
		Data: []byte(res),
		Metadata: ws.MessageMetadata{
			Direction: "sent",
			Timestamp: time.Now(),
		},
	})
}

// SampleBuiltin passes every Nth message and drops the rest.
type SampleBuiltin struct{}

func (b *SampleBuiltin) Name() string { return "sample" }
func (b *SampleBuiltin) Description() string {
	return "Pass every Nth message and drop the rest."
}
func (b *SampleBuiltin) Scope() BuiltinScope { return Shared }

func (b *SampleBuiltin) Validate(a Action) error {
	if a.Rate == "" {
		return fmt.Errorf("builtin sample: missing 'rate' (number of messages)")
	}
	return nil
}

func (b *SampleBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Evaluate rate template
	rateStr, err := d.templateEngine.Execute("sample-rate", a.Rate, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in rate expression: %w", err)
	}
	rateStr = strings.TrimSpace(rateStr)
	if rateStr == "" {
		return fmt.Errorf("builtin sample: rate evaluates to empty string")
	}

	// 2. Parse rate as integer
	rate, err := strconv.Atoi(rateStr)
	if err != nil {
		return fmt.Errorf("builtin sample: invalid rate %q (must be an integer): %w", rateStr, err)
	}
	if rate <= 0 {
		return fmt.Errorf("builtin sample: rate must be positive (got %d)", rate)
	}

	// 3. Get next count from registry
	key := a.HandlerName + ":" + a.Command
	if a.HandlerName == "" {
		key = "anon:" + a.Command
	}

	count := d.registry.GetNextSampleCount(key)

	// 4. Check if we should pass or drop
	if count%rate != 0 {
		if d.verbose {
			d.errorf("  [handler] sample: dropping message %d (rate: %d)\n", count, rate)
		}
		return ErrDrop
	}

	if d.verbose {
		d.errorf("  [handler] sample: passing message %d (rate: %d)\n", count, rate)
	}

	return nil
}
