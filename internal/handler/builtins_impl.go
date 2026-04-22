package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

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
	MustRegister(&HttpBuiltin{})
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
