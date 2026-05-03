package handler

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/itchyny/gojq"

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
	MustRegister(&SSEForwardBuiltin{})
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
	MustRegister(&HttpGetBuiltin{})
	MustRegister(&OllamaGenerateBuiltin{})
	MustRegister(&OllamaChatBuiltin{})
	MustRegister(&OllamaEmbedBuiltin{})
	MustRegister(&OllamaClassifyBuiltin{})
	MustRegister(&OpenAIChatBuiltin{})
	MustRegister(&HttpGraphQLBuiltin{})
	MustRegister(&HttpMockRespondBuiltin{})
	MustRegister(&MetricBuiltin{})
	MustRegister(&ThrottleBroadcastBuiltin{})
	MustRegister(&MulticastBuiltin{})
	MustRegister(&StickyBroadcastBuiltin{})
	MustRegister(&RoundRobinBuiltin{})
	MustRegister(&SampleBuiltin{})
	MustRegister(&ABTestBuiltin{})
	MustRegister(&OnceBuiltin{})
	MustRegister(&DebounceBuiltin{})
	MustRegister(&RuleEngineBuiltin{})
	MustRegister(&ShadowBuiltin{})
	MustRegister(&RedisSetBuiltin{})
	MustRegister(&RedisGetBuiltin{})
	MustRegister(&RedisDelBuiltin{})
	MustRegister(&RedisPublishBuiltin{})
	MustRegister(&RedisSubscribeBuiltin{})
	MustRegister(&RedisLPushBuiltin{})
	MustRegister(&RedisRPopBuiltin{})
	MustRegister(&RedisIncrBuiltin{})
	MustRegister(&WebhookBuiltin{})
	MustRegister(&WebhookHMACBuiltin{})
	MustRegister(&MQTTPublishBuiltin{})
	MustRegister(&MQTTSubscribeBuiltin{})
	MustRegister(&NATSPublishBuiltin{})
	MustRegister(&NATSSubscribeBuiltin{})
	MustRegister(&KafkaProduceBuiltin{})
	MustRegister(&KafkaConsumeBuiltin{})
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

func (b *SubscribeBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Subscribe the current connection to a pub/sub topic.",
		Fields: []BuiltinField{
			{Name: "topic", Type: "string", Required: true, Description: "Topic name (supports templates)."},
		},
		YAMLReplExample: "builtin: subscribe\ntopic: updates",
		REPLAddExample:  ":handler add -m '*' --builtin subscribe --topic updates",
	}
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

func (b *UnsubscribeBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Unsubscribe the current connection from a pub/sub topic.",
		Fields: []BuiltinField{
			{Name: "topic", Type: "string", Required: true, Description: "Topic name (supports templates)."},
		},
		YAMLReplExample: "builtin: unsubscribe\ntopic: updates",
		REPLAddExample:  ":handler add --builtin unsubscribe --topic updates",
	}
}

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

func (b *PublishBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Publish a message to a pub/sub topic.",
		Fields: []BuiltinField{
			{Name: "topic", Type: "string", Required: true, Description: "Topic name (supports templates)."},
			{Name: "message", Type: "string", Required: true, Description: "Message to publish (supports templates)."},
		},
		YAMLReplExample: "builtin: publish\ntopic: updates\nmessage: 'New data: {{.Message}}'",
		REPLAddExample:  ":handler add -m 'broadcast *' --builtin publish --topic updates --message '{{.Message}}'",
	}
}

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

func (b *KVSetBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Store a value in the server's shared key-value store.",
		Fields: []BuiltinField{
			{Name: "key", Type: "string", Required: true, Description: "Key (supports templates)."},
			{Name: "value", Type: "string", Required: true, Description: "Value (supports templates)."},
			{Name: "ttl", Type: "string", Required: false, Description: "TTL duration (e.g. 1h, 30s)."},
		},
		YAMLReplExample: "builtin: kv-set\nkey: 'user:{{.ConnID}}'\nvalue: 'online'\nttl: 5m",
		REPLAddExample:  ":handler add -m '*' --builtin kv-set --key 'user:{{.ConnID}}' --value 'online' --ttl 5m",
	}
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

// RedisSetBuiltin stores a key-value pair in Redis.
type RedisSetBuiltin struct{}

func (b *RedisSetBuiltin) Name() string { return "redis-set" }
func (b *RedisSetBuiltin) Description() string {
	return "Store a key-value pair in Redis with optional TTL."
}
func (b *RedisSetBuiltin) Scope() BuiltinScope { return Shared }

func (b *RedisSetBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Store a key-value pair in Redis with optional TTL.",
		Fields: []BuiltinField{
			{Name: "key", Type: "string", Required: true, Description: "Redis key (supports templates)."},
			{Name: "value", Type: "string", Required: true, Description: "Value to store (supports templates)."},
			{Name: "ttl", Type: "duration", Required: false, Description: "Optional expiration time (e.g. 1h)."},
		},
		YAMLReplExample: "builtin: redis-set\nkey: 'user:{{.ConnectionID}}'\nvalue: '{{.Message}}'\nttl: 24h",
		REPLAddExample:  ":handler add -m 'save *' --builtin redis-set --key 'data' --value '{{.Message}}'",
	}
}

func (b *RedisSetBuiltin) Validate(a Action) error {
	if a.Key == "" {
		return fmt.Errorf("builtin redis-set missing key")
	}
	if a.Value == "" {
		return fmt.Errorf("builtin redis-set missing value")
	}
	return nil
}

func (b *RedisSetBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.redisManager == nil {
		return fmt.Errorf("builtin redis-set: redis manager not initialized (check --redis-url)")
	}

	key, err := d.templateEngine.Execute("redis-key", a.Key, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in redis key: %w", err)
	}
	val, err := d.templateEngine.Execute("redis-value", a.Value, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in redis value: %w", err)
	}

	ttl := time.Duration(0)
	if a.TTL != "" {
		ttlStr, err := d.templateEngine.Execute("redis-ttl", a.TTL, tmplCtx)
		if err == nil {
			if dur, err := time.ParseDuration(ttlStr); err == nil {
				ttl = dur
			}
		}
	}

	if err := d.redisManager.Set(ctx, key, val, ttl); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	if d.verbose {
		if ttl > 0 {
			d.errorf("  [handler] redis-set: %s = %v (ttl: %v)\n", key, val, ttl)
		} else {
			d.errorf("  [handler] redis-set: %s = %v\n", key, val)
		}
	}
	return nil
}

// RedisGetBuiltin retrieves a value from Redis.
type RedisGetBuiltin struct{}

func (b *RedisGetBuiltin) Name() string { return "redis-get" }
func (b *RedisGetBuiltin) Description() string {
	return "Fetch a value from Redis into .RedisValue."
}
func (b *RedisGetBuiltin) Scope() BuiltinScope { return Shared }

func (b *RedisGetBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Fetch a value from Redis and store it in .RedisValue.",
		Fields: []BuiltinField{
			{Name: "key", Type: "string", Required: true, Description: "Redis key (supports templates)."},
		},
		TemplateVars: map[string]string{
			".RedisValue": "The value fetched from Redis.",
		},
		YAMLReplExample: "builtin: redis-get\nkey: 'config:global'\nrespond: 'Current config: {{.RedisValue}}'",
		REPLAddExample:  ":handler add -m 'get *' --builtin redis-get --key '{{.Message}}' -R 'Value: {{.RedisValue}}'",
	}
}

func (b *RedisGetBuiltin) Validate(a Action) error {
	if a.Key == "" {
		return fmt.Errorf("builtin redis-get missing key")
	}
	return nil
}

func (b *RedisGetBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.redisManager == nil {
		return fmt.Errorf("builtin redis-get: redis manager not initialized (check --redis-url)")
	}

	key, err := d.templateEngine.Execute("redis-key", a.Key, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in redis key: %w", err)
	}

	val, err := d.redisManager.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("redis get error: %w", err)
	}

	if val != nil {
		tmplCtx.RedisValue = val
	} else {
		// Not found, use default
		if a.Default != "" {
			defaultVal, err := d.templateEngine.Execute("redis-default", a.Default, tmplCtx)
			if err != nil {
				d.errorf("  [handler] redis-get: error rendering default template: %v\n", err)
				tmplCtx.RedisValue = ""
			} else {
				tmplCtx.RedisValue = defaultVal
			}
		} else {
			tmplCtx.RedisValue = ""
		}
	}

	if d.verbose {
		d.errorf("  [handler] redis-get: %s = %v (default=%q)\n", key, tmplCtx.RedisValue, a.Default)
	}
	return nil
}

// RedisDelBuiltin deletes a key from Redis.
type RedisDelBuiltin struct{}

func (b *RedisDelBuiltin) Name() string        { return "redis-del" }
func (b *RedisDelBuiltin) Description() string { return "Delete a key from Redis." }
func (b *RedisDelBuiltin) Scope() BuiltinScope { return Shared }

func (b *RedisDelBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Delete a key from Redis.",
		Fields: []BuiltinField{
			{Name: "key", Type: "string", Required: true, Description: "Redis key (supports templates)."},
		},
		YAMLReplExample: "builtin: redis-del\nkey: 'session:{{.ConnectionID}}'",
		REPLAddExample:  ":handler add -m 'clear' --builtin redis-del --key 'cache'",
	}
}

func (b *RedisDelBuiltin) Validate(a Action) error {
	if a.Key == "" {
		return fmt.Errorf("builtin redis-del missing key")
	}
	return nil
}

func (b *RedisDelBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.redisManager == nil {
		return fmt.Errorf("builtin redis-del: redis manager not initialized (check --redis-url)")
	}

	key, err := d.templateEngine.Execute("redis-key", a.Key, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in redis key: %w", err)
	}

	if err := d.redisManager.Del(ctx, key); err != nil {
		return fmt.Errorf("redis del error: %w", err)
	}

	if d.verbose {
		d.errorf("  [handler] redis-del: %s\n", key)
	}
	return nil
}
 
// RedisPublishBuiltin publishes a message to a Redis Pub/Sub channel.
type RedisPublishBuiltin struct{}
 
func (b *RedisPublishBuiltin) Name() string { return "redis-publish" }
func (b *RedisPublishBuiltin) Description() string {
	return "Publish a message to a Redis Pub/Sub channel."
}
func (b *RedisPublishBuiltin) Scope() BuiltinScope { return Shared }

func (b *RedisPublishBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Publish a message to a Redis Pub/Sub channel.",
		Fields: []BuiltinField{
			{Name: "channel", Type: "string", Required: true, Description: "Redis channel (supports templates)."},
			{Name: "message", Type: "string", Required: true, Description: "Message to publish (supports templates)."},
		},
		YAMLReplExample: "builtin: redis-publish\nchannel: 'events'\nmessage: '{{.Message}}'",
		REPLAddExample:  ":handler add -m 'notify *' --builtin redis-publish --channel 'alerts' --message '{{.Message}}'",
	}
}
 
func (b *RedisPublishBuiltin) Validate(a Action) error {
	if a.Channel == "" {
		return fmt.Errorf("builtin redis-publish missing channel")
	}
	if a.Message == "" {
		return fmt.Errorf("builtin redis-publish missing message")
	}
	return nil
}
 
func (b *RedisPublishBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.redisManager == nil {
		return fmt.Errorf("builtin redis-publish: redis manager not initialized (check --redis-url)")
	}
 
	channel, err := d.templateEngine.Execute("redis-channel", a.Channel, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in redis channel: %w", err)
	}
	msg, err := d.templateEngine.Execute("redis-message", a.Message, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in redis message: %w", err)
	}
 
	if err := d.redisManager.Publish(ctx, channel, msg); err != nil {
		return fmt.Errorf("redis publish error: %w", err)
	}
 
	if d.verbose {
		d.errorf("  [handler] redis-publish: %s -> %s\n", channel, msg)
	}
	return nil
}

// RedisLPushBuiltin pushes a value onto the left of a Redis list.
type RedisLPushBuiltin struct{}

func (b *RedisLPushBuiltin) Name() string { return "redis-lpush" }
func (b *RedisLPushBuiltin) Description() string {
	return "Push a value onto the left of a Redis list."
}
func (b *RedisLPushBuiltin) Scope() BuiltinScope { return Shared }

func (b *RedisLPushBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Push a value onto the left of a Redis list.",
		Fields: []BuiltinField{
			{Name: "key", Type: "string", Required: true, Description: "Redis list key (supports templates)."},
			{Name: "value", Type: "string", Required: true, Description: "Value to push (supports templates)."},
		},
		YAMLReplExample: "builtin: redis-lpush\nkey: 'history'\nvalue: '{{.Message}}'",
		REPLAddExample:  ":handler add -m 'log *' --builtin redis-lpush --key 'audit' --value '{{.Message}}'",
	}
}

func (b *RedisLPushBuiltin) Validate(a Action) error {
	if a.Key == "" {
		return fmt.Errorf("builtin redis-lpush missing key")
	}
	if a.Value == "" {
		return fmt.Errorf("builtin redis-lpush missing value")
	}
	return nil
}

func (b *RedisLPushBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.redisManager == nil {
		return fmt.Errorf("builtin redis-lpush: redis manager not initialized (check --redis-url)")
	}

	key, err := d.templateEngine.Execute("redis-key", a.Key, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in redis key: %w", err)
	}
	val, err := d.templateEngine.Execute("redis-value", a.Value, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in redis value: %w", err)
	}

	if err := d.redisManager.LPush(ctx, key, val); err != nil {
		return fmt.Errorf("redis lpush error: %w", err)
	}

	if d.verbose {
		d.errorf("  [handler] redis-lpush: %s <- %v\n", key, val)
	}
	return nil
}

// RedisRPopBuiltin pops a value from the right of a Redis list.
type RedisRPopBuiltin struct{}

func (b *RedisRPopBuiltin) Name() string { return "redis-rpop" }
func (b *RedisRPopBuiltin) Description() string {
	return "Pop a value from the right of a Redis list into .RedisValue."
}
func (b *RedisRPopBuiltin) Scope() BuiltinScope { return Shared }

func (b *RedisRPopBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Pop a value from the right of a Redis list into .RedisValue.",
		Fields: []BuiltinField{
			{Name: "key", Type: "string", Required: true, Description: "Redis list key (supports templates)."},
		},
		TemplateVars: map[string]string{
			".RedisValue": "The value popped from Redis.",
		},
		YAMLReplExample: "builtin: redis-rpop\nkey: 'queue'\nrespond: 'Processing: {{.RedisValue}}'",
		REPLAddExample:  ":handler add -m 'next' --builtin redis-rpop --key 'work' -R 'Task: {{.RedisValue}}'",
	}
}

func (b *RedisRPopBuiltin) Validate(a Action) error {
	if a.Key == "" {
		return fmt.Errorf("builtin redis-rpop missing key")
	}
	return nil
}

func (b *RedisRPopBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.redisManager == nil {
		return fmt.Errorf("builtin redis-rpop: redis manager not initialized (check --redis-url)")
	}

	key, err := d.templateEngine.Execute("redis-key", a.Key, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in redis key: %w", err)
	}

	val, err := d.redisManager.RPop(ctx, key)
	if err != nil {
		return fmt.Errorf("redis rpop error: %w", err)
	}

	if val != "" {
		tmplCtx.RedisValue = val
	} else {
		// Empty list, use default if provided
		if a.Default != "" {
			defaultVal, err := d.templateEngine.Execute("redis-default", a.Default, tmplCtx)
			if err != nil {
				d.errorf("  [handler] redis-rpop: error rendering default template: %v\n", err)
				tmplCtx.RedisValue = ""
			} else {
				tmplCtx.RedisValue = defaultVal
			}
		} else {
			tmplCtx.RedisValue = ""
		}
	}

	if d.verbose {
		d.errorf("  [handler] redis-rpop: %s -> %v (default=%q)\n", key, tmplCtx.RedisValue, a.Default)
	}
	return nil
}

// RedisIncrBuiltin increments a integer value in Redis.
type RedisIncrBuiltin struct{}

func (b *RedisIncrBuiltin) Name() string { return "redis-incr" }
func (b *RedisIncrBuiltin) Description() string {
	return "Atomically increment a Redis key by 1 or by a specified value."
}
func (b *RedisIncrBuiltin) Scope() BuiltinScope { return Shared }

func (b *RedisIncrBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Atomically increment a Redis key by 1 or by a specified value.",
		Fields: []BuiltinField{
			{Name: "key", Type: "string", Required: true, Description: "Redis key (supports templates)."},
			{Name: "by", Type: "int", Default: "1", Required: false, Description: "Increment amount."},
		},
		TemplateVars: map[string]string{
			".RedisValue": "The new value after increment.",
		},
		YAMLReplExample: "builtin: redis-incr\nkey: 'counter:{{.Message}}'\nby: 5",
		REPLAddExample:  ":handler add -m 'hits' --builtin redis-incr --key 'total_hits'",
	}
}

func (b *RedisIncrBuiltin) Validate(a Action) error {
	if a.Key == "" {
		return fmt.Errorf("builtin redis-incr missing key")
	}
	return nil
}

func (b *RedisIncrBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.redisManager == nil {
		return fmt.Errorf("builtin redis-incr: redis manager not initialized (check --redis-url)")
	}

	key, err := d.templateEngine.Execute("redis-key", a.Key, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in redis key: %w", err)
	}

	by := int64(1)
	if a.By != "" {
		byStr, err := d.templateEngine.Execute("redis-by", a.By, tmplCtx)
		if err != nil {
			return fmt.Errorf("template error in redis by: %w", err)
		}
		if bVal, err := strconv.ParseInt(strings.TrimSpace(byStr), 10, 64); err == nil {
			by = bVal
		} else {
			return fmt.Errorf("invalid increment value %q: %w", byStr, err)
		}
	}

	newVal, err := d.redisManager.Incr(ctx, key, by)
	if err != nil {
		return fmt.Errorf("redis incr error: %w", err)
	}

	tmplCtx.RedisValue = newVal

	if d.verbose {
		d.errorf("  [handler] redis-incr: %s incremented by %d -> %d\n", key, by, newVal)
	}
	return nil
}

// RedisSubscribeBuiltin subscribes to a Redis channel and broadcasts messages.
// This is a source builtin, meaning it is started by the server at load time.
type RedisSubscribeBuiltin struct{}

func (b *RedisSubscribeBuiltin) Name() string { return "redis-subscribe" }
func (b *RedisSubscribeBuiltin) Description() string {
	return "Subscribe to a Redis channel and deliver messages to WebSocket clients."
}
func (b *RedisSubscribeBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *RedisSubscribeBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Subscribe to a Redis channel and deliver messages to WebSocket clients.",
		Fields: []BuiltinField{
			{Name: "channel", Type: "string", Required: true, Description: "Redis channel (supports templates)."},
		},
		YAMLReplExample: "- builtin: redis-subscribe\n  channel: 'broadcast'\n  respond: '{\"source\":\"redis\",\"data\":{{.Message}}}'",
		REPLAddExample:  ":handler add --builtin redis-subscribe --channel 'live-updates'",
	}
}

func (b *RedisSubscribeBuiltin) Validate(a Action) error {
	if a.Channel == "" {
		return fmt.Errorf("builtin redis-subscribe missing channel")
	}
	if a.ReconnectInterval != "" {
		if _, err := time.ParseDuration(a.ReconnectInterval); err != nil {
			return fmt.Errorf("invalid reconnect_interval %q: %w", a.ReconnectInterval, err)
		}
	}
	return nil
}

func (b *RedisSubscribeBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	return fmt.Errorf("builtin %q is a source action and cannot be executed in a reactive flow", b.Name())
}


// KVGetBuiltin retrieves a value from the server's shared key-value store.
type KVGetBuiltin struct{}

func (b *KVGetBuiltin) Name() string { return "kv-get" }
func (b *KVGetBuiltin) Description() string {
	return "Retrieve a value from the server's shared key-value store into .KvValue."
}
func (b *KVGetBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *KVGetBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Retrieve a value from the server's shared key-value store.",
		Fields: []BuiltinField{
			{Name: "key", Type: "string", Required: true, Description: "KV key (supports templates)."},
		},
		TemplateVars: map[string]string{
			".KVValue": "The value retrieved from KV store.",
		},
		YAMLReplExample: "builtin: kv-get\nkey: 'user:{{.ConnectionID}}'\nrespond: 'Hello {{.KVValue}}'",
		REPLAddExample:  ":handler add -m 'whoami' --builtin kv-get --key 'user:{{.ConnectionID}}'",
	}
}

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

func (b *KVDelBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Delete a key from the server's shared key-value store.",
		Fields: []BuiltinField{
			{Name: "key", Type: "string", Required: true, Description: "KV key (supports templates)."},
		},
		YAMLReplExample: "builtin: kv-del\nkey: 'temp:{{.ConnectionID}}'",
		REPLAddExample:  ":handler add -m 'logout' --builtin kv-del --key 'session:{{.ConnectionID}}'",
	}
}

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

func (b *KVListBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Retrieve all keys from the server's shared key-value store.",
		TemplateVars: map[string]string{
			".KV": "Map of all key-value pairs.",
		},
		YAMLReplExample: "builtin: kv-list\nrespond: 'All keys: {{range $k, $v := .KV}}{{$k}}, {{end}}'",
		REPLAddExample:  ":handler add -m 'list' --builtin kv-list",
	}
}

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

func (b *GateBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Check a KV key against an expected value. Drops message if it doesn't match.",
		Fields: []BuiltinField{
			{Name: "key", Type: "string", Required: true, Description: "KV key to check."},
			{Name: "value", Type: "string", Required: true, Description: "Expected value (supports templates)."},
		},
		YAMLReplExample: "builtin: gate\nkey: 'maintenance_mode'\nvalue: 'false'",
		REPLAddExample:  ":handler add -m '*' --builtin gate --key 'allow_chat' --value 'true'",
	}
}

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

func (b *NoopBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "A shared builtin that does nothing (useful for testing or disabling actions).",
		YAMLReplExample: "builtin: noop",
		REPLAddExample:  ":handler add --builtin noop",
	}
}

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

func (b *EchoBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description:     "Reflect the current message back to the sender.",
		YAMLReplExample: "builtin: echo",
		REPLAddExample:  ":handler add -m '*' --builtin echo",
	}
}

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

func (b *BroadcastBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Fan-out a message to all connected clients.",
		Fields: []BuiltinField{
			{Name: "message", Type: "string", Required: true, Description: "Message content (supports templates)."},
		},
		YAMLReplExample: "builtin: broadcast\nmessage: 'Hello everyone!'",
		REPLAddExample:  ":handler add -m 'announce *' --builtin broadcast --message 'System: {{.Matches.0}}'",
	}
}

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

func (b *BroadcastOthersBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Fan-out a message to all connected clients except the sender.",
		Fields: []BuiltinField{
			{Name: "message", Type: "string", Required: true, Description: "Message content (supports templates)."},
		},
		YAMLReplExample: "builtin: broadcast-others\nmessage: '{{.ConnID}} joined the chat'",
		REPLAddExample:  ":handler add -m '*' --builtin broadcast-others --message 'User {{.ConnID}} says: {{.Message}}'",
	}
}

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

func (b *SequenceBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Cycle through a list of responses in order.",
		Fields: []BuiltinField{
			{Name: "responses", Type: "[]string", Required: true, Description: "List of response templates to cycle through."},
		},
		YAMLReplExample: "builtin: sequence\nresponses:\n  - 'First response'\n  - 'Second response'\n  - 'Third response'",
		REPLAddExample:  ":handler add -m 'ping' --builtin sequence --responses 'pong,pang,pung'",
	}
}

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

func (b *TemplateBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Render a response from an external template file.",
		Fields: []BuiltinField{
			{Name: "path", Type: "string", Required: true, Description: "Path to the .tmpl file."},
		},
		YAMLReplExample: "builtin: template\npath: 'templates/welcome.tmpl'",
		REPLAddExample:  ":handler add -m '*' --builtin template --path 'msg.tmpl'",
	}
}

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

func (b *FileSendBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Read a file from disk and send its contents as a message.",
		Fields: []BuiltinField{
			{Name: "path", Type: "string", Required: true, Description: "Path to the file (supports templates)."},
		},
		YAMLReplExample: "builtin: file-send\npath: 'responses/{{.Message}}.json'",
		REPLAddExample:  ":handler add -m 'read *' --builtin file-send --path 'data/{{.Message}}.txt'",
	}
}

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

func (b *FileWriteBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Write the message or a template-rendered variant to a file.",
		Fields: []BuiltinField{
			{Name: "path", Type: "string", Required: true, Description: "Path to the output file (supports templates)."},
			{Name: "append", Type: "bool", Default: "false", Required: false, Description: "Append to file instead of overwriting."},
			{Name: "body", Type: "string", Required: false, Description: "Content to write (defaults to raw message)."},
		},
		YAMLReplExample: "builtin: file-write\npath: 'logs/incoming.log'\nappend: true\nbody: '{{now}}: {{.Message}}\n'",
		REPLAddExample:  ":handler add -m '*' --builtin file-write --path 'data.txt' --append",
	}
}

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

func (b *RateLimitBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Enforce a per-client, global, or handler-level message rate.",
		Fields: []BuiltinField{
			{Name: "rate", Type: "string", Required: true, Description: "Token rate: N/s, N/m, N/h. Supports templates."},
			{Name: "burst", Type: "int", Required: false, Description: "Max burst size (default: N from rate)."},
			{Name: "scope", Type: "string", Default: "client", Required: false, Description: "client | global | handler"},
			{Name: "on_limit", Type: "tmpl", Required: false, Description: "Response on limit exceeded. Omit to drop silently."},
		},
		TemplateVars: map[string]string{
			".RetryAfter":   "seconds until next token (float64)",
			".RetryAfterMs": "milliseconds until next token (int)",
			".RateLimit":    "configured rate string",
			".LimitScope":   "effective scope",
		},
		YAMLReplExample: "- name: api-throttle\n  match: { jq: '.type == \"query\"' }\n  builtin: rate-limit\n  rate: 10/s\n  scope: client\n  on_limit: '{\"error\":\"rate_limited\",\"retry_after\":{{.RetryAfter}}}'",
		REPLAddExample:  ":handler add -m '.type == \"query\"' --builtin rate-limit --rate 10/s --scope client --on-limit '{\"error\":\"rate_limited\"}'",
	}
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

func (b *DelayBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Pause handler execution for a configurable duration before sending a response.",
		Fields: []BuiltinField{
			{Name: "duration", Type: "string", Required: true, Description: "Duration to pause (supports templates, e.g. 500ms, 1s)."},
			{Name: "max", Type: "string", Required: false, Description: "Hard cap on duration (e.g. 5s)."},
		},
		YAMLReplExample: "builtin: delay\nduration: 1s",
		REPLAddExample:  ":handler add -m '*' --builtin delay --duration 1s",
	}
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

func (b *DropBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description:     "Silently discard a message and stop further handler execution.",
		YAMLReplExample: "builtin: drop",
		REPLAddExample:  ":handler add -m 'ignore me' --builtin drop",
	}
}

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

func (b *CloseBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Close the current WebSocket connection.",
		Fields: []BuiltinField{
			{Name: "code", Type: "int", Required: false, Description: "WebSocket close code (default: 1000)."},
			{Name: "reason", Type: "string", Required: false, Description: "Close reason (supports templates)."},
		},
		YAMLReplExample: "builtin: close\ncode: 1001\nreason: 'Going away'",
		REPLAddExample:  ":handler add -m 'bye' --builtin close --reason 'User requested exit'",
	}
}

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

func (b *HttpBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Make an outbound HTTP request.",
		Fields: []BuiltinField{
			{Name: "url", Type: "string", Required: true, Description: "Target URL (supports templates)."},
			{Name: "method", Type: "string", Default: "GET", Required: false, Description: "HTTP method (GET, POST, etc.)."},
			{Name: "headers", Type: "map", Required: false, Description: "Custom HTTP headers."},
			{Name: "body", Type: "string", Required: false, Description: "Request body (supports templates)."},
		},
		TemplateVars: map[string]string{
			".HttpResponse": "Raw body of the HTTP response.",
			".HttpStatus":   "HTTP status code (int).",
		},
		YAMLReplExample: "builtin: http\nurl: 'https://api.example.com/data'\nmethod: POST\nbody: '{\"input\":\"{{.Message}}\"}'\nrespond: 'API says: {{.HttpResponse}}'",
		REPLAddExample:  ":handler add -m 'check' --builtin http --url 'https://status.io/api' -R 'Status: {{.HttpResponse}}'",
	}
}

func (b *HttpBuiltin) Validate(a Action) error {
	if a.URL == "" {
		return fmt.Errorf("builtin http missing url")
	}
	return nil
}

func (b *HttpBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	return b.execute(ctx, d, a, tmplCtx, "")
}

func (b *HttpBuiltin) execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext, forceMethod string) error {
	// 1. Resolve URL
	urlStr, err := d.templateEngine.Execute("http-url", a.URL, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in url expression: %w", err)
	}
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return fmt.Errorf("builtin http: url evaluates to empty string")
	}

	// 2. Resolve Method
	method := "GET"
	if forceMethod != "" {
		method = forceMethod
	} else if a.Method != "" {
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

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	System string `json:"system,omitempty"`
	Format string `json:"format,omitempty"`
}

type ollamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

type OllamaChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatRequest struct {
	Model    string              `json:"model"`
	Messages []OllamaChatMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type ollamaChatResponse struct {
	Model     string            `json:"model"`
	CreatedAt string            `json:"created_at"`
	Message   OllamaChatMessage `json:"message"`
	Done      bool              `json:"done"`
}

type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Input  string `json:"input,omitempty"`
	Prompt string `json:"prompt,omitempty"`
}

type ollamaEmbedResponse struct {
	Embedding  []float64   `json:"embedding"`
	Embeddings [][]float64 `json:"embeddings"`
}

type OpenAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []OpenAIChatMessage `json:"messages"`
	Stream      bool                `json:"stream"`
	Temperature *float64            `json:"temperature,omitempty"`
	TopP        *float64            `json:"top_p,omitempty"`
}

type openAIChatResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message      OpenAIChatMessage `json:"message"`
		Delta        OpenAIChatMessage `json:"delta"`
		FinishReason string            `json:"finish_reason"`
	} `json:"choices"`
}

// OllamaGenerateBuiltin sends a prompt to a local Ollama model.
type OllamaGenerateBuiltin struct{}

func (b *OllamaGenerateBuiltin) Name() string { return "ollama-generate" }
func (b *OllamaGenerateBuiltin) Description() string {
	return "Send a prompt to a local Ollama model and return the generated text."
}
func (b *OllamaGenerateBuiltin) Scope() BuiltinScope { return Shared }

func (b *OllamaGenerateBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Send a prompt to a local Ollama model and return the generated text.",
		Fields: []BuiltinField{
			{Name: "model", Type: "string", Default: "llama2", Required: false, Description: "Ollama model name."},
			{Name: "prompt", Type: "string", Required: true, Description: "The prompt to send (supports templates)."},
			{Name: "system", Type: "string", Required: false, Description: "System prompt for the model."},
			{Name: "options", Type: "map", Required: false, Description: "Model parameters (temperature, num_predict, etc.)."},
		},
		TemplateVars: map[string]string{
			".OllamaResponse": "The generated text from Ollama.",
		},
		YAMLReplExample: "builtin: ollama-generate\nmodel: mistral\nprompt: 'Summarize this: {{.Message}}'",
		REPLAddExample:  ":handler add -m 'ask *' --builtin ollama-generate --model mistral --prompt '{{.Message}}' -R '{{.OllamaResponse}}'",
	}
}

func (b *OllamaGenerateBuiltin) Validate(a Action) error {
	if a.Model == "" {
		return fmt.Errorf("builtin ollama-generate missing model")
	}
	if a.Prompt == "" {
		return fmt.Errorf("builtin ollama-generate missing prompt")
	}
	return nil
}

func (b *OllamaGenerateBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Resolve URL
	urlStr := a.OllamaURL
	if urlStr == "" {
		urlStr = d.ollamaURL // Use dispatcher's default if action doesn't specify
	}
	if urlStr == "" {
		urlStr = "http://localhost:11434/api/generate"
	}

	// Resolve URL template if needed
	if strings.Contains(urlStr, "{{") {
		evalURL, err := d.templateEngine.Execute("ollama-url", urlStr, tmplCtx)
		if err == nil && evalURL != "" {
			urlStr = evalURL
		}
	}

	// 2. Resolve Model
	model, err := d.templateEngine.Execute("ollama-model", a.Model, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering ollama model template: %w", err)
	}

	// 3. Resolve Prompt
	prompt, err := d.templateEngine.Execute("ollama-prompt", a.Prompt, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering ollama prompt template: %w", err)
	}

	// 4. Resolve Stream
	stream := false
	if a.Stream != "" {
		evalStream, err := d.templateEngine.Execute("ollama-stream", a.Stream, tmplCtx)
		if err == nil {
			stream = (evalStream == "true")
		}
	}

	// 5. Prepare Request
	ollamaReq := ollamaRequest{
		Model:  model,
		Prompt: prompt,
		Stream: stream,
	}
	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return fmt.Errorf("marshaling ollama request: %w", err)
	}

	// 6. Execute Request
	timeout := 60 * time.Second
	if a.Timeout != "" {
		if t, err := time.ParseDuration(a.Timeout); err == nil {
			timeout = t
		}
	}

	httpClient := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("creating ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if d.verbose {
		d.errorf("  [handler] ollama-generate: model=%q, stream=%v, url=%s\n", model, stream, urlStr)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(body))
	}

	if stream {
		decoder := json.NewDecoder(resp.Body)
		var fullReply strings.Builder
		for {
			var chunk ollamaResponse
			if err := decoder.Decode(&chunk); err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("decoding ollama stream: %w", err)
			}

			fullReply.WriteString(chunk.Response)
			tmplCtx.OllamaReply = chunk.Response

			// If respond: is present, send the chunk
			if a.Respond != "" {
				msg, err := d.templateEngine.Execute("ollama-respond", a.Respond, tmplCtx)
				if err != nil {
					return fmt.Errorf("rendering ollama stream response: %w", err)
				}
				if err := d.conn.Write(&ws.Message{
					Type: ws.TextMessage,
					Data: []byte(msg),
					Metadata: ws.MessageMetadata{
						Direction: "sent",
						Timestamp: time.Now(),
					},
				}); err != nil {
					return fmt.Errorf("sending ollama stream chunk: %w", err)
				}
			}

			if chunk.Done {
				break
			}
		}
		// Store full reply at the end in case anyone else needs it
		tmplCtx.OllamaReply = fullReply.String()
		// Clear Respond so dispatcher doesn't send it again
		a.Respond = ""
	} else {
		var ollamaResp ollamaResponse
		if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
			return fmt.Errorf("decoding ollama response: %w", err)
		}
		tmplCtx.OllamaReply = ollamaResp.Response
		if d.verbose {
			d.errorf("  [handler] ollama-generate: received %d chars\n", len(ollamaResp.Response))
		}
	}

	return nil
}

// OllamaChatBuiltin maintains a per-connection chat history and sends the next message to Ollama.
type OllamaChatBuiltin struct {
	histories sync.Map // connID -> []OllamaChatMessage
}

func (b *OllamaChatBuiltin) Name() string { return "ollama-chat" }
func (b *OllamaChatBuiltin) Description() string {
	return "Maintain a per-connection chat history and send the next message to Ollama."
}
func (b *OllamaChatBuiltin) Scope() BuiltinScope { return Shared }

func (b *OllamaChatBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Maintain a per-connection chat history and send the next message to Ollama.",
		Fields: []BuiltinField{
			{Name: "model", Type: "string", Default: "llama2", Required: false, Description: "Ollama model name."},
			{Name: "system", Type: "string", Required: false, Description: "System prompt for the conversation."},
			{Name: "max_history", Type: "int", Default: "10", Required: false, Description: "Number of past messages to retain."},
		},
		TemplateVars: map[string]string{
			".OllamaResponse": "The AI's response message.",
		},
		YAMLReplExample: "builtin: ollama-chat\nmodel: gemma\nsystem: 'You are a helpful assistant.'\nmax_history: 20",
		REPLAddExample:  ":handler add -m '*' --builtin ollama-chat --model gemma -R '{{.OllamaResponse}}'",
	}
}

func (b *OllamaChatBuiltin) Validate(a Action) error {
	if a.Model == "" {
		return fmt.Errorf("builtin ollama-chat missing model")
	}
	return nil
}

func (b *OllamaChatBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Resolve URL
	urlStr := a.OllamaURL
	if urlStr == "" {
		urlStr = d.ollamaURL
	}
	if urlStr == "" {
		urlStr = "http://localhost:11434/api/chat"
	} else if strings.HasSuffix(urlStr, "/api/generate") {
		urlStr = strings.TrimSuffix(urlStr, "/api/generate") + "/api/chat"
	}

	// Resolve URL template if needed
	if strings.Contains(urlStr, "{{") {
		evalURL, err := d.templateEngine.Execute("ollama-url", urlStr, tmplCtx)
		if err == nil && evalURL != "" {
			urlStr = evalURL
		}
	}

	// 2. Resolve Model
	model, err := d.templateEngine.Execute("ollama-model", a.Model, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering ollama model template: %w", err)
	}

	// 3. Resolve Prompt (default to current message)
	prompt := a.Prompt
	if prompt == "" {
		prompt = "{{.Message}}"
	}
	userMsg, err := d.templateEngine.Execute("ollama-prompt", prompt, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering ollama prompt template: %w", err)
	}

	// 4. Resolve System Prompt
	systemPrompt := ""
	if a.System != "" {
		systemPrompt, err = d.templateEngine.Execute("ollama-system", a.System, tmplCtx)
		if err != nil {
			return fmt.Errorf("rendering ollama system template: %w", err)
		}
	}

	// 5. Get History
	connID := d.conn.GetID()
	var history []OllamaChatMessage
	if val, ok := b.histories.Load(connID); ok {
		history = val.([]OllamaChatMessage)
	}

	// 6. Append User Turn
	history = append(history, OllamaChatMessage{Role: "user", Content: userMsg})

	// 7. Limit History
	if a.MaxHistory > 0 && len(history) > a.MaxHistory {
		history = history[len(history)-a.MaxHistory:]
	}

	// 8. Prepare Messages (System + History)
	messages := make([]OllamaChatMessage, 0, len(history)+1)
	if systemPrompt != "" {
		messages = append(messages, OllamaChatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, history...)

	// 9. Prepare Request
	stream := false
	if a.Stream != "" {
		evalStream, err := d.templateEngine.Execute("ollama-stream", a.Stream, tmplCtx)
		if err == nil {
			stream = (evalStream == "true")
		}
	}

	ollamaReq := ollamaChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   stream,
	}
	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return fmt.Errorf("marshaling ollama chat request: %w", err)
	}

	// 10. Execute Request
	timeout := 60 * time.Second
	if a.Timeout != "" {
		if t, err := time.ParseDuration(a.Timeout); err == nil {
			timeout = t
		}
	}

	httpClient := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("creating ollama chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if d.verbose {
		d.errorf("  [handler] ollama-chat: model=%q, history_len=%d, stream=%v, url=%s\n", model, len(history), stream, urlStr)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama chat request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama chat error (status %d): %s", resp.StatusCode, string(body))
	}

	var assistantReply string
	if stream {
		decoder := json.NewDecoder(resp.Body)
		var fullReply strings.Builder
		for {
			var chunk ollamaChatResponse
			if err := decoder.Decode(&chunk); err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("decoding ollama chat stream: %w", err)
			}

			fullReply.WriteString(chunk.Message.Content)
			tmplCtx.OllamaReply = chunk.Message.Content

			// If respond: is present, send the chunk
			if a.Respond != "" {
				msg, err := d.templateEngine.Execute("ollama-respond", a.Respond, tmplCtx)
				if err != nil {
					return fmt.Errorf("rendering ollama chat stream response: %w", err)
				}
				if err := d.conn.Write(&ws.Message{
					Type: ws.TextMessage,
					Data: []byte(msg),
					Metadata: ws.MessageMetadata{
						Direction: "sent",
						Timestamp: time.Now(),
					},
				}); err != nil {
					return fmt.Errorf("sending ollama chat stream chunk: %w", err)
				}
			}

			if chunk.Done {
				break
			}
		}
		assistantReply = fullReply.String()
		tmplCtx.OllamaReply = assistantReply
		a.Respond = "" // Clear Respond so dispatcher doesn't send it again
	} else {
		var ollamaResp ollamaChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
			return fmt.Errorf("decoding ollama chat response: %w", err)
		}
		assistantReply = ollamaResp.Message.Content
		tmplCtx.OllamaReply = assistantReply
		if d.verbose {
			d.errorf("  [handler] ollama-chat: received %d chars\n", len(assistantReply))
		}
	}

	// 11. Append Assistant Turn and Update History
	history = append(history, OllamaChatMessage{Role: "assistant", Content: assistantReply})

	// Apply MaxHistory again to ensure assistant turn doesn't push us over if we want to be strict
	if a.MaxHistory > 0 && len(history) > a.MaxHistory {
		history = history[len(history)-a.MaxHistory:]
	}

	b.histories.Store(connID, history)

	return nil
}

// OpenAIChatBuiltin maintains a per-connection chat history and sends the next message to an OpenAI-compatible API.
type OpenAIChatBuiltin struct {
	histories sync.Map // connID -> []OpenAIChatMessage
}

func (b *OpenAIChatBuiltin) Name() string { return "openai-chat" }
func (b *OpenAIChatBuiltin) Description() string {
	return "Maintain a per-connection chat history and send the next message to an OpenAI-compatible API."
}
func (b *OpenAIChatBuiltin) Scope() BuiltinScope { return Shared }

func (b *OpenAIChatBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Maintain a per-connection chat history and send the next message to an OpenAI-compatible API.",
		Fields: []BuiltinField{
			{Name: "model", Type: "string", Default: "gpt-3.5-turbo", Required: false, Description: "Model ID."},
			{Name: "api_key", Type: "string", Required: true, Description: "OpenAI API Key (or env var OPENAI_API_KEY)."},
			{Name: "api_url", Type: "string", Default: "https://api.openai.com/v1", Required: false, Description: "Base URL for the API."},
			{Name: "system", Type: "string", Required: false, Description: "System prompt."},
			{Name: "max_history", Type: "int", Default: "10", Required: false, Description: "Number of past messages to retain."},
		},
		TemplateVars: map[string]string{
			".AIResponse": "The AI's response message.",
		},
		YAMLReplExample: "builtin: openai-chat\napi_key: '${OPENAI_API_KEY}'\nsystem: 'You are a sarcastic bot.'",
		REPLAddExample:  ":handler add -m '*' --builtin openai-chat --api-key 'sk-...' -R '{{.AIResponse}}'",
	}
}

func (b *OpenAIChatBuiltin) Validate(a Action) error {
	if a.Model == "" {
		return fmt.Errorf("builtin openai-chat missing model")
	}
	return nil
}

func (b *OpenAIChatBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Resolve URL
	urlStr := a.APIURL
	if urlStr == "" {
		urlStr = "https://api.openai.com/v1/chat/completions"
	}

	// Resolve URL template if needed
	if strings.Contains(urlStr, "{{") {
		evalURL, err := d.templateEngine.Execute("openai-url", urlStr, tmplCtx)
		if err == nil && evalURL != "" {
			urlStr = evalURL
		}
	}

	// 2. Resolve API Key
	apiKey := ""
	if a.APIKey != "" {
		var err error
		apiKey, err = d.templateEngine.Execute("openai-key", a.APIKey, tmplCtx)
		if err != nil {
			return fmt.Errorf("rendering openai api key template: %w", err)
		}
	}
	apiKey = strings.TrimSpace(apiKey)

	// 3. Resolve Model
	model, err := d.templateEngine.Execute("openai-model", a.Model, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering openai model template: %w", err)
	}

	// 4. Resolve Prompt (default to current message)
	prompt := a.Prompt
	if prompt == "" {
		prompt = "{{.Message}}"
	}
	userMsg, err := d.templateEngine.Execute("openai-prompt", prompt, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering openai prompt template: %w", err)
	}

	// 5. Resolve System Prompt
	systemPrompt := ""
	if a.System != "" {
		systemPrompt, err = d.templateEngine.Execute("openai-system", a.System, tmplCtx)
		if err != nil {
			return fmt.Errorf("rendering openai system template: %w", err)
		}
	}

	// 6. Get History
	connID := d.conn.GetID()
	var history []OpenAIChatMessage
	if val, ok := b.histories.Load(connID); ok {
		history = val.([]OpenAIChatMessage)
	}

	// 7. Append User Turn
	history = append(history, OpenAIChatMessage{Role: "user", Content: userMsg})

	// 8. Limit History
	if a.MaxHistory > 0 && len(history) > a.MaxHistory {
		history = history[len(history)-a.MaxHistory:]
	}

	// 9. Prepare Messages (System + History)
	messages := make([]OpenAIChatMessage, 0, len(history)+1)
	if systemPrompt != "" {
		messages = append(messages, OpenAIChatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, history...)

	// 10. Prepare Request
	stream := false
	if a.Stream != "" {
		evalStream, err := d.templateEngine.Execute("openai-stream", a.Stream, tmplCtx)
		if err == nil {
			stream = (evalStream == "true")
		}
	}

	openaiReq := openAIChatRequest{
		Model:       model,
		Messages:    messages,
		Stream:      stream,
		Temperature: a.Temperature,
		TopP:        a.TopP,
	}
	reqBody, err := json.Marshal(openaiReq)
	if err != nil {
		return fmt.Errorf("marshaling openai chat request: %w", err)
	}

	// 11. Execute Request
	timeout := 60 * time.Second
	if a.Timeout != "" {
		if t, err := time.ParseDuration(a.Timeout); err == nil {
			timeout = t
		}
	}

	httpClient := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("creating openai chat request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	if d.verbose {
		d.errorf("  [handler] openai-chat: model=%q, history_len=%d, stream=%v, url=%s\n", model, len(history), stream, urlStr)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("openai chat request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai chat error (status %d): %s", resp.StatusCode, string(body))
	}

	var assistantReply string
	if stream {
		reader := bufio.NewReader(resp.Body)
		var fullReply strings.Builder
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("reading openai chat stream: %w", err)
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if line == "data: [DONE]" {
				break
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			var chunk openAIChatResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				return fmt.Errorf("unmarshaling openai chat stream chunk: %w", err)
			}

			if len(chunk.Choices) > 0 {
				content := chunk.Choices[0].Delta.Content
				fullReply.WriteString(content)
				tmplCtx.OpenAIReply = content
				tmplCtx.OllamaReply = content // Compatibility

				// If respond: is present, send the chunk
				if a.Respond != "" {
					msg, err := d.templateEngine.Execute("openai-respond", a.Respond, tmplCtx)
					if err != nil {
						return fmt.Errorf("rendering openai chat stream response: %w", err)
					}
					if err := d.conn.Write(&ws.Message{
						Type: ws.TextMessage,
						Data: []byte(msg),
						Metadata: ws.MessageMetadata{
							Direction: "sent",
							Timestamp: time.Now(),
						},
					}); err != nil {
						return fmt.Errorf("sending openai chat stream chunk: %w", err)
					}
				}
			}
		}
		assistantReply = fullReply.String()
		tmplCtx.OpenAIReply = assistantReply
		tmplCtx.OllamaReply = assistantReply // Compatibility
		a.Respond = ""                        // Clear Respond so dispatcher doesn't send it again
	} else {
		var openaiResp openAIChatResponse
		if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
			return fmt.Errorf("decoding openai chat response: %w", err)
		}
		if len(openaiResp.Choices) > 0 {
			assistantReply = openaiResp.Choices[0].Message.Content
		}
		tmplCtx.OpenAIReply = assistantReply
		tmplCtx.OllamaReply = assistantReply // Compatibility
		if d.verbose {
			d.errorf("  [handler] openai-chat: received %d chars\n", len(assistantReply))
		}
	}

	// 12. Append Assistant Turn and Update History
	history = append(history, OpenAIChatMessage{Role: "assistant", Content: assistantReply})

	// Apply MaxHistory again to ensure assistant turn doesn't push us over if we want to be strict
	if a.MaxHistory > 0 && len(history) > a.MaxHistory {
		history = history[len(history)-a.MaxHistory:]
	}

	b.histories.Store(connID, history)

	return nil
}

// OllamaEmbedBuiltin generates an embedding vector for a message using Ollama.
type OllamaEmbedBuiltin struct{}

func (b *OllamaEmbedBuiltin) Name() string { return "ollama-embed" }
func (b *OllamaEmbedBuiltin) Description() string {
	return "Generate an embedding vector for a message using Ollama."
}
func (b *OllamaEmbedBuiltin) Scope() BuiltinScope { return Shared }

func (b *OllamaEmbedBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Generate an embedding vector for a message using Ollama.",
		Fields: []BuiltinField{
			{Name: "model", Type: "string", Default: "llama2", Required: false, Description: "Embedding model name."},
			{Name: "input", Type: "string", Required: true, Description: "Text to embed (supports templates)."},
		},
		TemplateVars: map[string]string{
			".OllamaEmbedding": "The embedding vector (JSON float array).",
		},
		YAMLReplExample: "builtin: ollama-embed\ninput: '{{.Message}}'\nrespond: '{\"vector\": {{.OllamaEmbedding}}}'",
		REPLAddExample:  ":handler add -m 'vectorize *' --builtin ollama-embed --input '{{.Message}}'",
	}
}

func (b *OllamaEmbedBuiltin) Validate(a Action) error {
	if a.Model == "" {
		return fmt.Errorf("builtin ollama-embed missing model")
	}
	return nil
}

func (b *OllamaEmbedBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Resolve URL
	urlStr := a.OllamaURL
	if urlStr == "" {
		urlStr = d.ollamaURL
	}
	if urlStr == "" {
		urlStr = "http://localhost:11434/api/embeddings"
	} else if strings.HasSuffix(urlStr, "/api/generate") {
		urlStr = strings.TrimSuffix(urlStr, "/api/generate") + "/api/embeddings"
	} else if strings.HasSuffix(urlStr, "/api/chat") {
		urlStr = strings.TrimSuffix(urlStr, "/api/chat") + "/api/embeddings"
	}

	// Resolve URL template if needed
	if strings.Contains(urlStr, "{{") {
		evalURL, err := d.templateEngine.Execute("ollama-url", urlStr, tmplCtx)
		if err == nil && evalURL != "" {
			urlStr = evalURL
		}
	}

	// 2. Resolve Model
	model, err := d.templateEngine.Execute("ollama-model", a.Model, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering ollama model template: %w", err)
	}

	// 3. Resolve Input (default to current message)
	input := a.Input
	if input == "" {
		input = "{{.Message}}"
	}
	resolvedInput, err := d.templateEngine.Execute("ollama-input", input, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering ollama input template: %w", err)
	}

	// 4. Prepare Request
	ollamaReq := ollamaEmbedRequest{
		Model:  model,
		Input:  resolvedInput,
		Prompt: resolvedInput,
	}
	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return fmt.Errorf("marshaling ollama embed request: %w", err)
	}

	// 5. Execute Request
	timeout := 60 * time.Second
	if a.Timeout != "" {
		if t, err := time.ParseDuration(a.Timeout); err == nil {
			timeout = t
		}
	}

	httpClient := &http.Client{
		Timeout: timeout,
	}

	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("creating ollama embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if d.verbose {
		d.errorf("  [handler] ollama-embed: model=%q, input_len=%d, url=%s\n", model, len(resolvedInput), urlStr)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama embed request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama embed error (status %d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading ollama embed response: %w", err)
	}

	if d.verbose {
		d.errorf("  [handler] ollama-embed: raw response: %s\n", string(body))
	}

	var ollamaResp ollamaEmbedResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return fmt.Errorf("decoding ollama embed response: %w", err)
	}

	tmplCtx.Embedding = ollamaResp.Embedding
	if len(ollamaResp.Embeddings) > 0 && len(tmplCtx.Embedding) == 0 {
		tmplCtx.Embedding = ollamaResp.Embeddings[0]
	}

	if d.verbose {
		d.errorf("  [handler] ollama-embed: received vector of length %d\n", len(tmplCtx.Embedding))
	}

	return nil
}

// OllamaClassifyBuiltin categorizes a message into a set of labels using Ollama.
type OllamaClassifyBuiltin struct{}

func (b *OllamaClassifyBuiltin) Name() string { return "ollama-classify" }
func (b *OllamaClassifyBuiltin) Description() string {
	return "Classify a message into one of the provided labels using an Ollama model."
}
func (b *OllamaClassifyBuiltin) Scope() BuiltinScope { return Shared }

func (b *OllamaClassifyBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Classify a message into one of the provided labels using Ollama.",
		Fields: []BuiltinField{
			{Name: "model", Type: "string", Default: "llama2", Required: false, Description: "Model name."},
			{Name: "labels", Type: "[]string", Required: true, Description: "List of allowed classification labels."},
			{Name: "input", Type: "string", Required: true, Description: "Text to classify (supports templates)."},
		},
		TemplateVars: map[string]string{
			".OllamaLabel": "The predicted label from the list.",
		},
		YAMLReplExample: "builtin: ollama-classify\nlabels: [\"spam\", \"ham\"]\ninput: '{{.Message}}'",
		REPLAddExample:  ":handler add -m '*' --builtin ollama-classify --labels 'spam,ham' -R 'Class: {{.OllamaLabel}}'",
	}
}

func (b *OllamaClassifyBuiltin) Validate(a Action) error {
	if len(a.Labels.List) == 0 {
		return fmt.Errorf("builtin ollama-classify missing labels list")
	}
	return nil
}

func (b *OllamaClassifyBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Resolve URL
	urlStr := a.OllamaURL
	if urlStr == "" {
		urlStr = d.ollamaURL
	}
	if urlStr == "" {
		urlStr = "http://localhost:11434/api/generate"
	} else if !strings.HasSuffix(urlStr, "/api/generate") && !strings.HasSuffix(urlStr, "/api/chat") {
		urlStr = strings.TrimSuffix(urlStr, "/") + "/api/generate"
	}

	if strings.Contains(urlStr, "{{") {
		evalURL, err := d.templateEngine.Execute("ollama-url", urlStr, tmplCtx)
		if err == nil && evalURL != "" {
			urlStr = evalURL
		}
	}

	// 2. Resolve Model
	model := a.Model
	if model == "" {
		model = "llama3" // Sensible default
	}
	resolvedModel, err := d.templateEngine.Execute("ollama-model", model, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering ollama model template: %w", err)
	}

	// 3. Resolve Labels
	labels := a.Labels.List
	resolvedLabels := make([]string, 0, len(labels))
	for i, lTmpl := range labels {
		val, err := d.templateEngine.Execute(fmt.Sprintf("ollama-label-%d", i), lTmpl, tmplCtx)
		if err != nil {
			return fmt.Errorf("rendering ollama label %d template: %w", i, err)
		}
		resolvedLabels = append(resolvedLabels, val)
	}

	// 4. Construct Classification Prompt
	labelListStr := strings.Join(resolvedLabels, ", ")
	prompt := fmt.Sprintf(`Classify the following message into exactly one of these labels: [%s].
Message: %s

Respond ONLY with a JSON object containing "label" and "confidence" fields.
Example: {"label": "%s", "confidence": 0.95}`,
		labelListStr, tmplCtx.Message, resolvedLabels[0])

	// 5. Prepare Request
	ollamaReq := ollamaRequest{
		Model:  resolvedModel,
		Prompt: prompt,
		Stream: false,
		Format: "json",
	}
	if a.System != "" {
		sys, err := d.templateEngine.Execute("ollama-system", a.System, tmplCtx)
		if err == nil {
			ollamaReq.System = sys
		}
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return fmt.Errorf("marshaling ollama classify request: %w", err)
	}

	// 6. Execute Request
	timeout := 60 * time.Second
	if a.Timeout != "" {
		if t, err := time.ParseDuration(a.Timeout); err == nil {
			timeout = t
		}
	}

	httpClient := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("creating ollama classify request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if d.verbose {
		d.errorf("  [handler] ollama-classify: model=%q, labels=[%s], url=%s\n", resolvedModel, labelListStr, urlStr)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama classify request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama classify error (status %d): %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return fmt.Errorf("decoding ollama classify response: %w", err)
	}

	// 7. Parse Result JSON
	var result struct {
		Label      string  `json:"label"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(ollamaResp.Response), &result); err != nil {
		// Fallback: try to find label name in response if JSON parsing fails
		for _, l := range resolvedLabels {
			if strings.Contains(strings.ToLower(ollamaResp.Response), strings.ToLower(l)) {
				result.Label = l
				result.Confidence = 0.5 // Low confidence fallback
				break
			}
		}
		if result.Label == "" {
			return fmt.Errorf("failed to parse ollama classification result: %s", ollamaResp.Response)
		}
	}

	tmplCtx.Label = result.Label
	tmplCtx.Confidence = result.Confidence

	if d.verbose {
		d.errorf("  [handler] ollama-classify: result=%q (conf: %.2f)\n", result.Label, result.Confidence)
	}

	return nil
}

// HttpGetBuiltin makes an outbound HTTP GET request.
type HttpGetBuiltin struct {
	HttpBuiltin
}

func (b *HttpGetBuiltin) Name() string { return "http-get" }
func (b *HttpGetBuiltin) Description() string {
	return "Make an outbound HTTP GET request."
}
func (b *HttpGetBuiltin) Scope() BuiltinScope { return Shared }

func (b *HttpGetBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Make an outbound HTTP GET request (shorthand for http builtin).",
		Fields: []BuiltinField{
			{Name: "url", Type: "string", Required: true, Description: "Target URL (supports templates)."},
			{Name: "headers", Type: "map", Required: false, Description: "Custom HTTP headers."},
		},
		TemplateVars: map[string]string{
			".HttpResponse": "Raw body of the HTTP response.",
			".HttpStatus":   "HTTP status code (int).",
		},
		YAMLReplExample: "builtin: http-get\nurl: 'https://wttr.in/London?format=3'",
		REPLAddExample:  ":handler add -m 'weather' --builtin http-get --url 'https://wttr.in/London?format=3'",
	}
}

func (b *HttpGetBuiltin) Validate(a Action) error {
	if a.URL == "" {
		return fmt.Errorf("builtin http-get missing url")
	}
	return nil
}

func (b *HttpGetBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	return b.execute(ctx, d, a, tmplCtx, "GET")
}

// HttpGraphQLBuiltin makes an outbound GraphQL POST request.
type HttpGraphQLBuiltin struct {
	HttpBuiltin
}

func (b *HttpGraphQLBuiltin) Name() string { return "http-graphql" }
func (b *HttpGraphQLBuiltin) Description() string {
	return "Make an outbound GraphQL POST request."
}
func (b *HttpGraphQLBuiltin) Scope() BuiltinScope { return Shared }

func (b *HttpGraphQLBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Make an outbound GraphQL POST request.",
		Fields: []BuiltinField{
			{Name: "url", Type: "string", Required: true, Description: "GraphQL endpoint URL."},
			{Name: "query", Type: "string", Required: true, Description: "GraphQL query string (supports templates)."},
			{Name: "variables", Type: "map", Required: false, Description: "GraphQL variables (values support templates)."},
			{Name: "headers", Type: "map", Required: false, Description: "Custom HTTP headers."},
		},
		TemplateVars: map[string]string{
			".HttpResponse": "Raw JSON response from the GraphQL server.",
		},
		YAMLReplExample: "builtin: http-graphql\nurl: 'https://countries.trevorblades.com/'\nquery: 'query($code: ID!) { country(code: $code) { name emoji } }'\nvariables: { code: \"{{.Message}}\" }",
		REPLAddExample:  ":handler add -m 'country *' --builtin http-graphql --url 'https://countries.trevorblades.com/' --query '...' --variables '{\"code\":\"{{.Message}}\"}'",
	}
}

func (b *HttpGraphQLBuiltin) Validate(a Action) error {
	if a.URL == "" {
		return fmt.Errorf("builtin http-graphql missing url")
	}
	if a.Query == "" {
		return fmt.Errorf("builtin http-graphql missing query")
	}
	return nil
}

func (b *HttpGraphQLBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Render URL
	urlStr, err := d.templateEngine.Execute("http-url", a.URL, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in url expression: %w", err)
	}

	// 2. Render Query
	query, err := d.templateEngine.Execute("graphql-query", a.Query, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in query expression: %w", err)
	}

	// 3. Render Variables
	var variables map[string]interface{}
	if a.Variables != "" {
		varsStr, err := d.templateEngine.Execute("graphql-variables", a.Variables, tmplCtx)
		if err != nil {
			return fmt.Errorf("template error in variables expression: %w", err)
		}
		if varsStr != "" {
			if err := json.Unmarshal([]byte(varsStr), &variables); err != nil {
				return fmt.Errorf("invalid variables JSON: %w", err)
			}
		}
	}

	// 4. Construct Payload
	payload := map[string]interface{}{
		"query": query,
	}
	if variables != nil {
		payload["variables"] = variables
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal graphql payload: %w", err)
	}

	// 5. Create Request
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 6. Set Custom Headers
	if a.Headers != nil {
		for k, v := range a.Headers {
			evalK, err := d.templateEngine.Execute("http-header-key", k, tmplCtx)
			if err == nil {
				evalV, err := d.templateEngine.Execute("http-header-val", v, tmplCtx)
				if err == nil {
					req.Header.Set(evalK, evalV)
				}
			}
		}
	}

	// 7. Set Timeout
	timeout := 10 * time.Second
	if a.Timeout != "" {
		if t, err := time.ParseDuration(a.Timeout); err == nil {
			timeout = t
		}
	}

	client := &http.Client{
		Timeout: timeout,
	}

	// 8. Execute Request
	if d.verbose {
		d.errorf("  [handler] http-graphql: POST %s (timeout: %v)\n", urlStr, timeout)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("graphql request failed: %w", err)
	}
	defer resp.Body.Close()

	// 9. Read Response Body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read graphql response body: %w", err)
	}

	// 10. Update Context
	tmplCtx.HttpStatus = resp.StatusCode

	var gqlResponse struct {
		Data   interface{} `json:"data"`
		Errors interface{} `json:"errors"`
	}

	if err := json.Unmarshal(respBody, &gqlResponse); err == nil {
		if gqlResponse.Data != nil {
			dataJSON, _ := json.Marshal(gqlResponse.Data)
			tmplCtx.HttpBody = string(dataJSON)
		} else {
			tmplCtx.HttpBody = string(respBody)
		}
		tmplCtx.GraphQLErrors = gqlResponse.Errors
	} else {
		tmplCtx.HttpBody = string(respBody)
	}

	if d.verbose {
		d.errorf("  [handler] http-graphql: received status %d (%d bytes)\n", resp.StatusCode, len(respBody))
	}

	return nil
}

// HttpMockRespondBuiltin registers a canned HTTP response at a specific path.
type HttpMockRespondBuiltin struct{}

func (b *HttpMockRespondBuiltin) Name() string { return "http-mock-respond" }
func (b *HttpMockRespondBuiltin) Description() string {
	return "Register a canned HTTP response at a specific path."
}
func (b *HttpMockRespondBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *HttpMockRespondBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Register a canned HTTP response at a specific path (for mock servers).",
		Fields: []BuiltinField{
			{Name: "path", Type: "string", Required: true, Description: "HTTP path to mock (e.g. /api/health)."},
			{Name: "status", Type: "int", Default: "200", Required: false, Description: "HTTP status code."},
			{Name: "body", Type: "string", Required: false, Description: "Response body (supports templates)."},
			{Name: "content_type", Type: "string", Default: "application/json", Required: false, Description: "Content-Type header."},
		},
		YAMLReplExample: "- builtin: http-mock-respond\n  path: '/v1/status'\n  status: 200\n  body: '{\"status\":\"ok\",\"time\":\"{{now}}\"}'",
		REPLAddExample:  ":handler add --builtin http-mock-respond --path '/ping' --body 'pong'",
	}
}

func (b *HttpMockRespondBuiltin) Validate(a Action) error {
	if a.Path == "" {
		return fmt.Errorf("builtin http-mock-respond missing path")
	}
	if a.Status == "" {
		return fmt.Errorf("builtin http-mock-respond missing status")
	}
	return nil
}

func (b *HttpMockRespondBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.serverStats == nil {
		return fmt.Errorf("builtin http-mock-respond: server stats provider not available")
	}

	// 1. Resolve Path
	path, err := d.templateEngine.Execute("http-mock-path", a.Path, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in path: %w", err)
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("builtin http-mock-respond: path evaluates to empty string")
	}

	// 2. Resolve Status
	statusStr, err := d.templateEngine.Execute("http-mock-status", a.Status, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in status: %w", err)
	}
	status, err := strconv.Atoi(strings.TrimSpace(statusStr))
	if err != nil {
		return fmt.Errorf("invalid status code %q: %w", statusStr, err)
	}

	// 3. Resolve Headers
	headers := make(map[string]string)
	if a.Headers != nil {
		for k, v := range a.Headers {
			evalK, _ := d.templateEngine.Execute("http-mock-header-key", k, tmplCtx)
			evalV, _ := d.templateEngine.Execute("http-mock-header-value", v, tmplCtx)
			if evalK != "" {
				headers[evalK] = evalV
			}
		}
	}

	// 4. Resolve Body
	body, err := d.templateEngine.Execute("http-mock-body", a.Body, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in body: %w", err)
	}

	mock := template.HTTPMockResponse{
		Status:  status,
		Headers: headers,
		Body:    body,
	}

	if d.verbose {
		d.errorf("  [handler] registering http-mock at %s (status=%d)\n", path, status)
	}

	return d.serverStats.RegisterHTTPMock(path, mock)
}

// LogBuiltin writes a structured log entry to stdout, a file, or both.
type LogBuiltin struct{}

func (b *LogBuiltin) Name() string { return "log" }
func (b *LogBuiltin) Description() string {
	return "Write a structured JSONL log entry to stdout, a file, or both."
}
func (b *LogBuiltin) Scope() BuiltinScope { return Shared }

func (b *LogBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Print a message to the server/client console or a file.",
		Fields: []BuiltinField{
			{Name: "message", Type: "string", Required: true, Description: "Log message (supports templates)."},
			{Name: "target", Type: "string", Default: "stderr", Required: false, Description: "Output target (stdout, stderr, or filename)."},
		},
		YAMLReplExample: "builtin: log\nmessage: 'Received message from {{.ConnID}}'\ntarget: 'activity.log'",
		REPLAddExample:  ":handler add -m '*' --builtin log --message 'DEBUG: {{.Message}}'",
	}
}

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

func (b *MetricBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Increment a Prometheus counter.",
		Fields: []BuiltinField{
			{Name: "name", Type: "string", Required: true, Description: "Metric name."},
			{Name: "help", Type: "string", Required: false, Description: "Help text for the metric."},
			{Name: "labels", Type: "map", Required: false, Description: "Label names and template values."},
		},
		YAMLReplExample: "builtin: metric\nname: 'messages_total'\nhelp: 'Total messages received'\nlabels: { type: \"{{.Type}}\" }",
		REPLAddExample:  ":handler add -m '*' --builtin metric --name 'hits'",
	}
}

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
	for k, vTmpl := range a.Labels.ToMap() {
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

func (b *ThrottleBroadcastBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Deliver a message to all clients except those who received it recently via this handler.",
		Fields: []BuiltinField{
			{Name: "message", Type: "string", Required: true, Description: "Message to broadcast."},
			{Name: "rate", Type: "string", Required: true, Description: "Max rate per client (e.g. 1/s)."},
		},
		YAMLReplExample: "builtin: throttle-broadcast\nmessage: '{{.Message}}'\nrate: 1/s",
		REPLAddExample:  ":handler add --builtin throttle-broadcast --message '...' --rate '1/s'",
	}
}

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

func (b *MulticastBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Send a message to a specific list of client IDs.",
		Fields: []BuiltinField{
			{Name: "clients", Type: "[]string", Required: true, Description: "List of client IDs (supports templates)."},
			{Name: "message", Type: "string", Required: true, Description: "Message to send."},
		},
		YAMLReplExample: "builtin: multicast\nclients: ['{{.Session.target_id}}']\nmessage: 'Hello!'",
		REPLAddExample:  ":handler add --builtin multicast --clients 'ID1,ID2' --message 'Hello'",
	}
}

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

func (b *StickyBroadcastBuiltin) Name() string { return "sticky-broadcast" }
func (b *StickyBroadcastBuiltin) Description() string {
	return "Broadcast and retain a message for a topic."
}
func (b *StickyBroadcastBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *StickyBroadcastBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Broadcast and retain a message for a topic. New clients joining the topic will receive it immediately.",
		Fields: []BuiltinField{
			{Name: "topic", Type: "string", Required: true, Description: "Topic name."},
			{Name: "message", Type: "string", Required: true, Description: "Message to retain."},
		},
		YAMLReplExample: "builtin: sticky-broadcast\ntopic: 'system-status'\nmessage: 'ONLINE'",
		REPLAddExample:  ":handler add --builtin sticky-broadcast --topic 'motd' --message 'Welcome!'",
	}
}

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

func (b *RoundRobinBuiltin) Name() string { return "round-robin" }
func (b *RoundRobinBuiltin) Description() string {
	return "Cycle messages across a pool of client IDs."
}
func (b *RoundRobinBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *RoundRobinBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Cycle messages across a pool of client IDs.",
		Fields: []BuiltinField{
			{Name: "clients", Type: "[]string", Required: true, Description: "List of target client IDs."},
			{Name: "message", Type: "string", Required: true, Description: "Message to send."},
		},
		YAMLReplExample: "builtin: round-robin\nclients: ['worker-1', 'worker-2']\nmessage: '{{.Message}}'",
		REPLAddExample:  ":handler add --builtin round-robin --clients 'w1,w2' --message 'task'",
	}
}

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

// ABTestBuiltin routes messages to one of two handlers based on a deterministic hash of a field.
type ABTestBuiltin struct{}

func (b *ABTestBuiltin) Name() string { return "ab-test" }
func (b *ABTestBuiltin) Description() string {
	return "Route messages to one of two handlers based on a deterministic hash of a field."
}
func (b *ABTestBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *ABTestBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Route messages to one of two handlers based on a deterministic hash of the connection ID.",
		Fields: []BuiltinField{
			{Name: "a", Type: "string", Required: true, Description: "Handler for Group A."},
			{Name: "b", Type: "string", Required: true, Description: "Handler for Group B."},
			{Name: "split", Type: "float", Default: "0.5", Required: false, Description: "Probability of Group A (0.0 to 1.0)."},
		},
		YAMLReplExample: "builtin: ab-test\na: 'new-ui-handler'\nb: 'legacy-handler'\nsplit: 0.2",
		REPLAddExample:  ":handler add --builtin ab-test --a 'v1' --b 'v2'",
	}
}

func (b *ABTestBuiltin) Validate(a Action) error {
	if a.Field == "" {
		return fmt.Errorf("builtin ab-test: missing 'field' (jq expression)")
	}
	if a.HandlerA == "" {
		return fmt.Errorf("builtin ab-test: missing 'handler_a'")
	}
	if a.HandlerB == "" {
		return fmt.Errorf("builtin ab-test: missing 'handler_b'")
	}
	if a.Split != nil && (*a.Split < 0 || *a.Split > 100) {
		return fmt.Errorf("builtin ab-test: 'split' must be between 0 and 100")
	}
	return nil
}

func (b *ABTestBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Extract field value using jq
	var data interface{}
	if err := json.Unmarshal(tmplCtx.MessageBytes, &data); err != nil {
		// Not JSON, can't use jq field meaningfully if it expects structure.
		// Fallback to raw message as string for simple values.
		data = string(tmplCtx.MessageBytes)
	}

	query, err := gojq.Parse(a.Field)
	if err != nil {
		return fmt.Errorf("builtin ab-test: invalid jq expression %q: %w", a.Field, err)
	}

	iter := query.Run(data)
	v, ok := iter.Next()
	if !ok {
		v = "" // No result
	}
	if err, ok := v.(error); ok {
		return fmt.Errorf("builtin ab-test: jq evaluation error: %w", err)
	}

	// 2. Deterministic hash
	h := fnv.New32a()
	h.Write([]byte(fmt.Sprintf("%v", v)))
	hashVal := h.Sum32()
	bucket := int(hashVal % 100)

	// 3. Routing decision
	split := 50
	if a.Split != nil {
		split = *a.Split
	}

	var chosen string
	if bucket < split {
		chosen = a.HandlerA
	} else {
		chosen = a.HandlerB
	}

	if d.verbose {
		d.errorf("  [handler] ab-test: routing to handler %q (bucket %d, split %d)\n", chosen, bucket, split)
	}

	// 4. Retrieve and execute chosen handler
	targetHandler, ok := d.registry.GetHandler(chosen)
	if !ok {
		return fmt.Errorf("builtin ab-test: chosen handler %q not found", chosen)
	}

	msg := d.connToMessage(tmplCtx)
	return d.Execute(ctx, &targetHandler, msg, nil)
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

func (b *SampleBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Pass every Nth message and drop the rest.",
		Fields: []BuiltinField{
			{Name: "rate", Type: "int", Required: true, Description: "Sampling rate (e.g. 10 means 1 in 10 messages)."},
		},
		YAMLReplExample: "builtin: sample\nrate: 100",
		REPLAddExample:  ":handler add --builtin sample --rate 5",
	}
}

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

// OnceBuiltin executes its respond template once and then disables the handler.
type OnceBuiltin struct{}

func (b *OnceBuiltin) Name() string { return "once" }
func (b *OnceBuiltin) Description() string {
	return "Executes its respond template once and then permanently disables the handler."
}
func (b *OnceBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *OnceBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Executes its respond template once and then permanently disables itself for that connection.",
		Fields: []BuiltinField{
			{Name: "respond", Type: "tmpl", Required: true, Description: "Response to send once."},
		},
		YAMLReplExample: "builtin: once\nrespond: '{\"welcome\": true}'",
		REPLAddExample:  ":handler add --builtin once --respond 'Welcome!'",
	}
}

func (b *OnceBuiltin) Validate(a Action) error {
	return nil
}

func (b *OnceBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if a.HandlerName == "" {
		return fmt.Errorf("builtin once: handler name not available")
	}

	if d.verbose {
		d.errorf("  [handler] once: firing for handler %q\n", a.HandlerName)
	}

	// Disable the handler first to avoid race conditions if multiple messages arrive
	if err := d.registry.DisableHandlerWithReason(a.HandlerName, "once"); err != nil {
		return fmt.Errorf("builtin once: disabling handler %q: %w", a.HandlerName, err)
	}

	// If a.Respond is set, ExecuteAction will handle sending the transformed message.
	// But since we want to be sure it's sent as part of the 'once' execution,
	// and we are disabling the handler now, we should handle it here or let ExecuteAction do it.
	// Actually, ExecuteAction calls this and then handles a.Respond.
	// However, if we want to ensure it's sent ONLY on the first match, we are good because
	// subsequent matches won't even reach here (handler will be disabled).

	return nil
}

// DebounceBuiltin suppresses repeated matching messages within a time window and only processes the last one.
type DebounceBuiltin struct{}

func (b *DebounceBuiltin) Name() string { return "debounce" }
func (b *DebounceBuiltin) Description() string {
	return "Suppress repeated matching messages within a time window and only process the last one."
}
func (b *DebounceBuiltin) Scope() BuiltinScope { return Shared }

func (b *DebounceBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Suppress repeated matching messages within a time window. Only the first message is passed.",
		Fields: []BuiltinField{
			{Name: "duration", Type: "duration", Required: true, Description: "Quiet period (e.g. 500ms)."},
			{Name: "key", Type: "string", Required: false, Description: "Grouping key (supports templates)."},
		},
		YAMLReplExample: "builtin: debounce\nduration: 1s\nkey: '{{.Message}}'",
		REPLAddExample:  ":handler add --builtin debounce --duration 1s",
	}
}

func (b *DebounceBuiltin) Validate(a Action) error {
	if a.Window == "" && a.Duration == "" && a.Delay == "" {
		return fmt.Errorf("builtin debounce: missing 'window', 'duration', or 'delay'")
	}
	if a.Scope != "" {
		s := strings.ToLower(a.Scope)
		if s != "client" && s != "global" {
			return fmt.Errorf("builtin debounce: invalid scope %q (must be 'client' or 'global')", a.Scope)
		}
	}
	return nil
}

func (b *DebounceBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Resolve window duration
	windowStr := a.Window
	if windowStr == "" {
		windowStr = a.Duration
	}
	if windowStr == "" {
		windowStr = a.Delay
	}

	evaluatedWindow, err := d.templateEngine.Execute("debounce-window", windowStr, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering debounce window template: %w", err)
	}
	dur, err := time.ParseDuration(evaluatedWindow)
	if err != nil {
		return fmt.Errorf("invalid debounce duration %q: %w", evaluatedWindow, err)
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

	if scope == "client" {
		key = "debounce:client:" + handlerName + ":" + d.conn.GetID()
	} else {
		key = "debounce:global:" + handlerName
	}

	if d.verbose {
		d.errorf("  [handler] debounce: starting window of %v for key %q\n", dur, key)
	}

	// 3. Debounce
	// We capture the current dispatcher and action in a closure.
	// Note: msg is provided by handleMessage to ExecuteAction.
	// But ExecuteAction takes msg as an argument.
	// The msg variable is available in the scope of Dispatcher.ExecuteAction call.
	// However, we are in DebounceBuiltin.Execute, which also receives msg via ExecuteAction?
	// Wait, DebounceBuiltin.Execute does NOT receive msg.
	// Ah, I need to pass msg to Execute.
	// Let's check the signature of Execute in builtins_impl.go.
	// It is: func (b *SomeBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error
	// It does NOT have msg!
	// But tmplCtx has the message content.
	// Wait, I can reconstruct the message or I can change the signature if needed.
	// Actually, most builtins use tmplCtx.MessageBytes or tmplCtx.Message.

	// I'll check if I can get the raw message from d.
	// Dispatcher doesn't seem to store the "current" message in a field, it's passed around.

	// I'll reconstruct a *ws.Message from tmplCtx for the debouncer.
	msg := &ws.Message{
		Type: ws.TextMessage,
		Data: tmplCtx.MessageBytes,
		Metadata: ws.MessageMetadata{
			Direction:    tmplCtx.Direction,
			Timestamp:    tmplCtx.Timestamp,
			MessageIndex: tmplCtx.MessageIndex,
		},
	}
	if tmplCtx.MessageType == "binary" {
		msg.Type = ws.BinaryMessage
	}

	d.registry.Debounce(key, dur, msg, func(m *ws.Message) {
		if d.verbose {
			d.errorf("  [handler] debounce: window expired for %q, executing response\n", key)
		}

		// Re-populate context for the debounced message
		callbackCtx := template.NewContext()
		d.populateTemplateContext(callbackCtx, m)

		// Execute respond if present
		if a.Respond != "" {
			resp, err := d.templateEngine.Execute("debounce-respond", a.Respond, callbackCtx)
			if err != nil {
				d.errorf("  [handler] debounce: error rendering respond template: %v\n", err)
				return
			}
			if resp != "" {
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
	})

	return ErrDrop
}

// ShadowBuiltin forwards messages to another handler asynchronously and silently.
type ShadowBuiltin struct{}

func (b *ShadowBuiltin) Name() string { return "shadow" }
func (b *ShadowBuiltin) Description() string {
	return "Forward messages to another handler asynchronously and silently (server-only)."
}
func (b *ShadowBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *ShadowBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Forward messages to another handler asynchronously and silently (fire-and-forget).",
		Fields: []BuiltinField{
			{Name: "handler", Type: "string", Required: true, Description: "Name of the target handler."},
		},
		YAMLReplExample: "builtin: shadow\nhandler: 'logger-pipeline'",
		REPLAddExample:  ":handler add -m '*' --builtin shadow --handler 'analytics'",
	}
}

func (b *ShadowBuiltin) Validate(a Action) error {
	if a.Target == "" {
		return fmt.Errorf("builtin shadow: missing 'target' (handler name)")
	}
	return nil
}

func (b *ShadowBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	target, err := d.templateEngine.Execute("shadow-target", a.Target, tmplCtx)
	if err != nil {
		return fmt.Errorf("rendering shadow target template: %w", err)
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("builtin shadow: target evaluates to empty string")
	}

	targetHandler, ok := d.registry.GetHandler(target)
	if !ok {
		return fmt.Errorf("builtin shadow: handler %q not found", target)
	}

	// Reconstruct message from context
	msg := d.connToMessage(tmplCtx)

	// Execute asynchronously
	go func() {
		// Create a silent connection that discards all writes
		silent := &silentConn{Connection: d.conn}

		// Use a fresh context for the shadow execution
		shadowCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Execute the target handler using a cloned dispatcher with the silent connection
		shadowD := d.cloneWithConn(silent)

		if err := shadowD.Execute(shadowCtx, &targetHandler, msg, nil); err != nil {
			if d.verbose {
				d.errorf("  [handler] shadow error for handler %q: %v\n", target, err)
			}
		}
	}()

	if d.verbose {
		d.errorf("  [handler] shadow: dispatched message to handler %q asynchronously\n", target)
	}

	return nil
}

// RuleEngineBuiltin evaluates a list of conditions and executes the first matching one.
type RuleEngineBuiltin struct{}

func (b *RuleEngineBuiltin) Name() string { return "rule-engine" }
func (b *RuleEngineBuiltin) Description() string {
	return "Evaluate a list of rules in order and execute the first match."
}
func (b *RuleEngineBuiltin) Scope() BuiltinScope { return Shared }

func (b *RuleEngineBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Evaluate a list of rules in order and execute the first matching rule's actions.",
		Fields: []BuiltinField{
			{Name: "rules", Type: "[]Rule", Required: true, Description: "List of rules with 'match' and 'actions'."},
		},
		YAMLReplExample: "builtin: rule-engine\nrules:\n  - match: { jq: '.type == \"alert\"' }\n    actions: [{ builtin: log, message: \"Alert!\" }]\n  - match: { glob: \"*\" }\n    actions: [{ builtin: echo }]",
		REPLAddExample:  ":handler add --builtin rule-engine",
	}
}

func (b *RuleEngineBuiltin) Validate(a Action) error {
	if len(a.Rules) == 0 {
		return fmt.Errorf("builtin rule-engine: rules list is required and cannot be empty")
	}
	return nil
}

func (b *RuleEngineBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	for i, rule := range a.Rules {
		matched, matches, err := d.registry.matchMatcher(&rule.When, a.BaseDir, d.connToMessage(tmplCtx), d.templateEngine, tmplCtx)
		if err != nil {
			if d.verbose {
				d.errorf("  [handler] rule-engine: error matching rule %d: %v\n", i, err)
			}
			continue
		}

		if matched {
			if d.verbose {
				d.errorf("  [handler] rule-engine: rule %d matched\n", i)
			}

			// Capture matches from the matching rule's condition
			originalMatches := tmplCtx.Matches
			tmplCtx.Matches = matches
			defer func() { tmplCtx.Matches = originalMatches }()

			if rule.Respond != "" {
				resp, err := d.templateEngine.Execute("rule-respond", rule.Respond, tmplCtx)
				if err != nil {
					return fmt.Errorf("rule %d: rendering response template: %w", i, err)
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
			return nil
		}
	}

	// Fallback to default if no rules matched
	if a.Default != "" {
		if d.verbose {
			d.errorf("  [handler] rule-engine: no rules matched, using default\n")
		}
		resp, err := d.templateEngine.Execute("rule-default", a.Default, tmplCtx)
		if err != nil {
			return fmt.Errorf("rendering default template: %w", err)
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

	if d.verbose {
		d.errorf("  [handler] rule-engine: no rules matched and no default provided\n")
	}

	return nil
}

// WebhookBuiltin POSTs a message to an HTTP endpoint.
type WebhookBuiltin struct{}

func (b *WebhookBuiltin) Name() string { return "webhook" }
func (b *WebhookBuiltin) Description() string {
	return "POST a message to an HTTP endpoint (defaults to raw message body)."
}
func (b *WebhookBuiltin) Scope() BuiltinScope { return Shared }

func (b *WebhookBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "POST a message to an HTTP endpoint (shorthand for http POST).",
		Fields: []BuiltinField{
			{Name: "url", Type: "string", Required: true, Description: "Webhook URL (supports templates)."},
			{Name: "body", Type: "string", Required: false, Description: "Custom payload (defaults to raw message)."},
		},
		YAMLReplExample: "builtin: webhook\nurl: 'https://hooks.slack.com/services/...' \nbody: '{\"text\":\"{{.Message}}\"}'",
		REPLAddExample:  ":handler add -m 'alert *' --builtin webhook --url '...' --body '{\"alert\":\"{{.Message}}\"}'",
	}
}

func (b *WebhookBuiltin) Validate(a Action) error {
	if a.URL == "" {
		return fmt.Errorf("builtin webhook missing url")
	}
	return nil
}

func (b *WebhookBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Resolve URL
	urlStr, err := d.templateEngine.Execute("webhook-url", a.URL, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in url expression: %w", err)
	}
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return fmt.Errorf("builtin webhook: url evaluates to empty string")
	}

	// 2. Resolve Body (defaults to raw message if not provided)
	bodyContent := a.Body
	if bodyContent == "" {
		bodyContent = tmplCtx.Message
	}

	bodyStr, err := d.templateEngine.Execute("webhook-body", bodyContent, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in body expression: %w", err)
	}
	bodyReader := strings.NewReader(bodyStr)

	// 3. Create Request (always POST)
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	// 4. Add Headers
	if a.Headers != nil {
		for k, v := range a.Headers {
			evalK, _ := d.templateEngine.Execute("webhook-header-key", k, tmplCtx)
			evalV, _ := d.templateEngine.Execute("webhook-header-value", v, tmplCtx)
			if evalK != "" {
				req.Header.Set(evalK, evalV)
			}
		}
	}
	// Set default Content-Type if not provided and body looks like JSON
	if req.Header.Get("Content-Type") == "" {
		trimmedBody := strings.TrimSpace(bodyStr)
		if (strings.HasPrefix(trimmedBody, "{") && strings.HasSuffix(trimmedBody, "}")) ||
			(strings.HasPrefix(trimmedBody, "[") && strings.HasSuffix(trimmedBody, "]")) {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	// 5. Set Timeout
	timeout := 10 * time.Second
	if a.Timeout != "" {
		if t, err := time.ParseDuration(a.Timeout); err == nil {
			timeout = t
		}
	}

	client := &http.Client{
		Timeout: timeout,
	}

	// 6. Execute Request
	if d.verbose {
		d.errorf("  [handler] webhook: POST %s (timeout: %v)\n", urlStr, timeout)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Network errors trigger on_error (by returning error here)
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	// 7. Read Response Body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read webhook response body: %w", err)
	}

	// 8. Update Context
	tmplCtx.HttpStatus = resp.StatusCode
	tmplCtx.HttpBody = string(respBody)

	if d.verbose {
		d.errorf("  [handler] webhook: received status %d (%d bytes)\n", resp.StatusCode, len(respBody))
	}

	return nil
}

// WebhookHMACBuiltin POSTs a message to an HTTP endpoint with an HMAC-SHA256 signature.
type WebhookHMACBuiltin struct{}

func (b *WebhookHMACBuiltin) Name() string { return "webhook-hmac" }
func (b *WebhookHMACBuiltin) Description() string {
	return "POST a message to an HTTP endpoint with an HMAC-SHA256 signature (X-Hub-Signature-256)."
}
func (b *WebhookHMACBuiltin) Scope() BuiltinScope { return Shared }

func (b *WebhookHMACBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "POST a message to an HTTP endpoint with an HMAC-SHA256 signature header.",
		Fields: []BuiltinField{
			{Name: "url", Type: "string", Required: true, Description: "Webhook URL."},
			{Name: "secret", Type: "string", Required: true, Description: "HMAC secret key."},
			{Name: "header", Type: "string", Default: "X-Hub-Signature-256", Required: false, Description: "Header name for signature."},
			{Name: "body", Type: "string", Required: false, Description: "Custom payload."},
		},
		YAMLReplExample: "builtin: webhook-hmac\nurl: 'https://receiver.com/webhook'\nsecret: 'top-secret'\nbody: '{{.Message}}'",
		REPLAddExample:  ":handler add -m '*' --builtin webhook-hmac --url '...' --secret '...' --body '{{.Message}}'",
	}
}

func (b *WebhookHMACBuiltin) Validate(a Action) error {
	if a.URL == "" {
		return fmt.Errorf("builtin webhook-hmac missing url")
	}
	if a.Secret == "" {
		return fmt.Errorf("builtin webhook-hmac missing secret")
	}
	return nil
}

func (b *WebhookHMACBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	// 1. Resolve URL
	urlStr, err := d.templateEngine.Execute("webhook-hmac-url", a.URL, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in url expression: %w", err)
	}
	urlStr = strings.TrimSpace(urlStr)
	if urlStr == "" {
		return fmt.Errorf("builtin webhook-hmac: url evaluates to empty string")
	}

	// 2. Resolve Body (defaults to raw message if not provided)
	bodyContent := a.Body
	if bodyContent == "" {
		bodyContent = tmplCtx.Message
	}

	bodyStr, err := d.templateEngine.Execute("webhook-hmac-body", bodyContent, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in body expression: %w", err)
	}
	bodyBytes := []byte(bodyStr)
	bodyReader := io.NopCloser(strings.NewReader(bodyStr))

	// 3. Resolve Secret
	secretStr, err := d.templateEngine.Execute("webhook-hmac-secret", a.Secret, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in secret expression: %w", err)
	}

	// 4. Calculate HMAC
	mac := hmac.New(sha256.New, []byte(secretStr))
	mac.Write(bodyBytes)
	signature := hex.EncodeToString(mac.Sum(nil))

	// 5. Create Request (always POST)
	req, err := http.NewRequestWithContext(ctx, "POST", urlStr, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create webhook-hmac request: %w", err)
	}

	// 6. Add Headers
	if a.Headers != nil {
		for k, v := range a.Headers {
			evalK, _ := d.templateEngine.Execute("webhook-hmac-header-key", k, tmplCtx)
			evalV, _ := d.templateEngine.Execute("webhook-hmac-header-value", v, tmplCtx)
			if evalK != "" {
				req.Header.Set(evalK, evalV)
			}
		}
	}

	// Add HMAC signature header
	req.Header.Set("X-Hub-Signature-256", "sha256="+signature)

	// Set default Content-Type if not provided and body looks like JSON
	if req.Header.Get("Content-Type") == "" {
		trimmedBody := strings.TrimSpace(bodyStr)
		if (strings.HasPrefix(trimmedBody, "{") && strings.HasSuffix(trimmedBody, "}")) ||
			(strings.HasPrefix(trimmedBody, "[") && strings.HasSuffix(trimmedBody, "]")) {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	// 7. Set Timeout
	timeout := 10 * time.Second
	if a.Timeout != "" {
		if t, err := time.ParseDuration(a.Timeout); err == nil {
			timeout = t
		}
	}

	client := &http.Client{
		Timeout: timeout,
	}

	// 8. Execute Request
	if d.verbose {
		d.errorf("  [handler] webhook-hmac: POST %s (timeout: %v)\n", urlStr, timeout)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook-hmac request failed: %w", err)
	}
	defer resp.Body.Close()

	// 9. Read Response Body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read webhook-hmac response body: %w", err)
	}

	// 10. Update Context
	tmplCtx.HttpStatus = resp.StatusCode
	tmplCtx.HttpBody = string(respBody)

	if d.verbose {
		d.errorf("  [handler] webhook-hmac: received status %d (%d bytes)\n", resp.StatusCode, len(respBody))
	}

	return nil
}
// SSEForwardBuiltin forwards WebSocket messages to an SSE stream.
type SSEForwardBuiltin struct{}

func (b *SSEForwardBuiltin) Name() string { return "sse-forward" }
func (b *SSEForwardBuiltin) Description() string {
	return "Forward messages to a named Server-Sent Events (SSE) stream."
}
func (b *SSEForwardBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *SSEForwardBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Forward messages to a named Server-Sent Events (SSE) stream.",
		Fields: []BuiltinField{
			{Name: "stream", Type: "string", Required: true, Description: "Stream name (e.g. /events)."},
			{Name: "event", Type: "string", Required: false, Description: "SSE event type."},
		},
		YAMLReplExample: "builtin: sse-forward\nstream: '/api/v1/stream'\nevent: 'message'",
		REPLAddExample:  ":handler add --builtin sse-forward --stream '/live'",
	}
}

func (b *SSEForwardBuiltin) Validate(a Action) error {
	if a.Stream == "" {
		return fmt.Errorf("builtin sse-forward missing stream")
	}
	return nil
}

func (b *SSEForwardBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.serverStats == nil {
		return fmt.Errorf("builtin sse-forward: server stats provider not available")
	}

	stream, err := d.templateEngine.Execute("sse-stream", a.Stream, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in stream expression: %w", err)
	}
	stream = strings.TrimSpace(stream)
	if stream == "" {
		return fmt.Errorf("builtin sse-forward: stream evaluates to empty string")
	}

	event := "message"
	if a.Event != "" {
		ev, err := d.templateEngine.Execute("sse-event", a.Event, tmplCtx)
		if err != nil {
			return fmt.Errorf("template error in event expression: %w", err)
		}
		if e := strings.TrimSpace(ev); e != "" {
			event = e
		}
	}

	msgContent := a.Message
	if msgContent == "" {
		// Default to raw message
		msgContent = "{{.Message}}"
	}

	data, err := d.templateEngine.Execute("sse-data", msgContent, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in message expression: %w", err)
	}

	id := ""
	if a.ID != "" {
		evID, err := d.templateEngine.Execute("sse-id", a.ID, tmplCtx)
		if err != nil {
			return fmt.Errorf("template error in id expression: %w", err)
		}
		id = strings.TrimSpace(evID)
	}

	// Update stream config if provided in the handler
	if a.OnNoConsumers != "" || a.BufferSize > 0 {
		onNoConsumers := a.OnNoConsumers
		if onNoConsumers == "" {
			onNoConsumers = "drop" // Default
		}
		bufferSize := a.BufferSize
		if bufferSize <= 0 {
			bufferSize = 100 // Default
		}
		_ = d.serverStats.UpdateSSEStreamConfig(stream, onNoConsumers, bufferSize)
	}

	err = d.serverStats.SendToSSE(stream, event, data, id)
	if err != nil {
		return fmt.Errorf("sse-forward error: %w", err)
	}

	if d.verbose {
		d.errorf("  [handler] sse-forward: stream=%q event=%q id=%q\n", stream, event, id)
	}

	return nil
}

// MQTTPublishBuiltin publishes a message to an MQTT broker.
type MQTTPublishBuiltin struct{}

func (b *MQTTPublishBuiltin) Name() string { return "mqtt-publish" }
func (b *MQTTPublishBuiltin) Description() string {
	return "Publish a message to an MQTT broker topic."
}
func (b *MQTTPublishBuiltin) Scope() BuiltinScope { return Shared }

func (b *MQTTPublishBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Publish a message to an MQTT topic.",
		Fields: []BuiltinField{
			{Name: "broker_url", Type: "string", Required: true, Description: "MQTT broker URL (supports templates)."},
			{Name: "topic", Type: "string", Required: true, Description: "MQTT topic (supports templates)."},
			{Name: "message", Type: "string", Required: true, Description: "Message content (supports templates)."},
			{Name: "qos", Type: "string", Default: "0", Required: false, Description: "QoS level (0, 1, 2)."},
			{Name: "retain", Type: "bool", Default: "false", Required: false, Description: "Retain message."},
		},
		YAMLReplExample: "builtin: mqtt-publish\nbroker_url: 'tcp://localhost:1883'\ntopic: 'home/living-room/light'\nmessage: 'ON'",
		REPLAddExample:  ":handler add -m 'light on' --builtin mqtt-publish --broker-url 'tcp://localhost:1883' --topic 'home/light' --message 'ON'",
	}
}

func (b *MQTTPublishBuiltin) Validate(a Action) error {
	if a.BrokerURL == "" {
		return fmt.Errorf("builtin mqtt-publish missing broker_url")
	}
	if a.Topic == "" {
		return fmt.Errorf("builtin mqtt-publish missing topic")
	}
	if a.Message == "" && a.Send == "" && a.Respond == "" {
		return fmt.Errorf("builtin mqtt-publish missing message (provide message:, send:, or respond:)")
	}
	return nil
}

func (b *MQTTPublishBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.mqttManager == nil {
		return fmt.Errorf("builtin mqtt-publish: mqtt manager not initialized")
	}

	brokerURL, err := d.templateEngine.Execute("mqtt-broker", a.BrokerURL, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in mqtt broker_url: %w", err)
	}
	brokerURL = strings.TrimSpace(brokerURL)
	if brokerURL == "" {
		return fmt.Errorf("builtin mqtt-publish: broker_url evaluates to empty string")
	}

	topic, err := d.templateEngine.Execute("mqtt-topic", a.Topic, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in mqtt topic: %w", err)
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return fmt.Errorf("builtin mqtt-publish: topic evaluates to empty string")
	}

	msgContent := a.Message
	if msgContent == "" {
		msgContent = a.Send
	}
	if msgContent == "" {
		msgContent = a.Respond
	}

	msgStr, err := d.templateEngine.Execute("mqtt-message", msgContent, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in mqtt message: %w", err)
	}

	qos := byte(0)
	if a.QoS != "" {
		qosStr, err := d.templateEngine.Execute("mqtt-qos", a.QoS, tmplCtx)
		if err == nil {
			if q, err := strconv.Atoi(strings.TrimSpace(qosStr)); err == nil {
				if q < 0 || q > 2 {
					return fmt.Errorf("invalid mqtt qos: %d (must be 0, 1, or 2)", q)
				}
				qos = byte(q)
			}
		}
	}

	if err := d.mqttManager.Publish(ctx, brokerURL, topic, msgStr, qos, a.Retain); err != nil {
		return fmt.Errorf("mqtt publish error: %w", err)
	}

	tmplCtx.MqttTopic = topic
	tmplCtx.MqttMessage = msgStr

	if d.verbose {
		d.errorf("  [handler] mqtt-publish: %s -> %s (topic: %q, qos: %d, retain: %v)\n", brokerURL, msgStr, topic, qos, a.Retain)
	}
	return nil
}

// MQTTSubscribeBuiltin subscribes to an MQTT topic and broadcasts messages.
// This is a source builtin, meaning it is started by the server at load time.
type MQTTSubscribeBuiltin struct{}

func (b *MQTTSubscribeBuiltin) Name() string { return "mqtt-subscribe" }
func (b *MQTTSubscribeBuiltin) Description() string {
	return "Subscribe to an MQTT topic and deliver messages to WebSocket clients."
}
func (b *MQTTSubscribeBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *MQTTSubscribeBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Subscribe to an MQTT topic and deliver messages to WebSocket clients.",
		Fields: []BuiltinField{
			{Name: "broker_url", Type: "string", Required: true, Description: "MQTT broker URL (supports templates)."},
			{Name: "topic", Type: "string", Required: true, Description: "MQTT topic (supports templates)."},
			{Name: "qos", Type: "string", Default: "0", Required: false, Description: "QoS level (0, 1, 2)."},
		},
		YAMLReplExample: "- builtin: mqtt-subscribe\n  broker_url: 'tcp://localhost:1883'\n  topic: 'sensors/#'\n  respond: '{\"mqtt_topic\":\"{{.Topic}}\",\"value\":{{.Message}}}'",
		REPLAddExample:  ":handler add --builtin mqtt-subscribe --broker-url 'tcp://localhost:1883' --topic 'sensors/#'",
	}
}

func (b *MQTTSubscribeBuiltin) Validate(a Action) error {
	if a.BrokerURL == "" {
		return fmt.Errorf("builtin mqtt-subscribe missing broker_url")
	}
	if a.Topic == "" {
		return fmt.Errorf("builtin mqtt-subscribe missing topic")
	}
	if a.ReconnectInterval != "" {
		if _, err := time.ParseDuration(a.ReconnectInterval); err != nil {
			return fmt.Errorf("invalid reconnect_interval %q: %w", a.ReconnectInterval, err)
		}
	}
	return nil
}

func (b *MQTTSubscribeBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	return fmt.Errorf("builtin %q is a source action and cannot be executed in a reactive flow", b.Name())
}

// NATSPublishBuiltin publishes a message to a NATS subject.
type NATSPublishBuiltin struct{}

func (b *NATSPublishBuiltin) Name() string { return "nats-publish" }
func (b *NATSPublishBuiltin) Description() string {
	return "Publish a message to a NATS subject."
}
func (b *NATSPublishBuiltin) Scope() BuiltinScope { return Shared }

func (b *NATSPublishBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Publish a message to a NATS subject.",
		Fields: []BuiltinField{
			{Name: "nats_url", Type: "string", Required: true, Description: "NATS server URL (supports templates)."},
			{Name: "subject", Type: "string", Required: true, Description: "NATS subject (supports templates)."},
			{Name: "message", Type: "string", Required: true, Description: "Message content (supports templates)."},
		},
		YAMLReplExample: "builtin: nats-publish\nnats_url: 'nats://localhost:4222'\nsubject: 'events.chat'\nmessage: '{{.Message}}'",
		REPLAddExample:  ":handler add -m '*' --builtin nats-publish --nats-url 'nats://localhost:4222' --subject 'chat' --message '{{.Message}}'",
	}
}

func (b *NATSPublishBuiltin) Validate(a Action) error {
	if a.NatsURL == "" {
		return fmt.Errorf("builtin nats-publish missing nats_url")
	}
	if a.Subject == "" {
		return fmt.Errorf("builtin nats-publish missing subject")
	}
	if a.Message == "" && a.Send == "" && a.Respond == "" {
		return fmt.Errorf("builtin nats-publish missing message (provide message:, send:, or respond:)")
	}
	return nil
}

func (b *NATSPublishBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	if d.natsManager == nil {
		return fmt.Errorf("builtin nats-publish: nats manager not initialized")
	}

	natsURL, err := d.templateEngine.Execute("nats-url", a.NatsURL, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in nats_url: %w", err)
	}
	natsURL = strings.TrimSpace(natsURL)
	if natsURL == "" {
		return fmt.Errorf("builtin nats-publish: nats_url evaluates to empty string")
	}

	subject, err := d.templateEngine.Execute("nats-subject", a.Subject, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in nats subject: %w", err)
	}
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return fmt.Errorf("builtin nats-publish: subject evaluates to empty string")
	}

	msgContent := a.Message
	if msgContent == "" {
		msgContent = a.Send
	}
	if msgContent == "" {
		msgContent = a.Respond
	}

	msgStr, err := d.templateEngine.Execute("nats-message", msgContent, tmplCtx)
	if err != nil {
		return fmt.Errorf("template error in nats message: %w", err)
	}

	if err := d.natsManager.Publish(ctx, natsURL, subject, msgStr); err != nil {
		return fmt.Errorf("nats publish error: %w", err)
	}

	tmplCtx.NatsSubject = subject
	tmplCtx.NatsMessage = msgStr

	if d.verbose {
		d.errorf("  [handler] nats-publish: %s -> %s (subject: %q)\n", natsURL, msgStr, subject)
	}
	return nil
}

// NATSSubscribeBuiltin subscribes to a NATS subject and delivers messages to WebSocket clients.
type NATSSubscribeBuiltin struct{}

func (b *NATSSubscribeBuiltin) Name() string { return "nats-subscribe" }
func (b *NATSSubscribeBuiltin) Description() string {
	return "Subscribe to a NATS subject and deliver messages to WebSocket clients."
}
func (b *NATSSubscribeBuiltin) Scope() BuiltinScope { return ServerOnly }

func (b *NATSSubscribeBuiltin) Help() BuiltinHelp {
	return BuiltinHelp{
		Description: "Subscribe to a NATS subject and deliver messages to WebSocket clients.",
		Fields: []BuiltinField{
			{Name: "nats_url", Type: "string", Required: true, Description: "NATS server URL (supports templates)."},
			{Name: "subject", Type: "string", Required: true, Description: "NATS subject (supports templates)."},
		},
		YAMLReplExample: "- builtin: nats-subscribe\n  nats_url: 'nats://localhost:4222'\n  subject: 'broadcast.#'\n  respond: '{\"source\":\"nats\",\"data\":{{.Message}}}'",
		REPLAddExample:  ":handler add --builtin nats-subscribe --nats-url 'nats://localhost:4222' --subject 'chat.*'",
	}
}

func (b *NATSSubscribeBuiltin) Validate(a Action) error {
	if a.NatsURL == "" {
		return fmt.Errorf("builtin nats-subscribe missing nats_url")
	}
	if a.Subject == "" {
		return fmt.Errorf("builtin nats-subscribe missing subject")
	}
	if a.ReconnectInterval != "" {
		if _, err := time.ParseDuration(a.ReconnectInterval); err != nil {
			return fmt.Errorf("invalid reconnect_interval %q: %w", a.ReconnectInterval, err)
		}
	}
	return nil
}

func (b *NATSSubscribeBuiltin) Execute(ctx context.Context, d *Dispatcher, a *Action, tmplCtx *template.TemplateContext) error {
	return fmt.Errorf("builtin %q is a source action and cannot be executed in a reactive flow", b.Name())
}
