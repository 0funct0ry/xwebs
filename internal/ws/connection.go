package ws

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// MessageType represents the type of WebSocket message.
type MessageType int

const (
	// TextMessage is a text WebSocket message.
	TextMessage MessageType = iota
	// BinaryMessage is a binary WebSocket message.
	BinaryMessage
)

// Message represents a WebSocket message.
type Message struct {
	Type MessageType
	Data []byte
}

// Connection wraps a gorilla websocket connection with additional metadata and lifecycle management.
type Connection struct {
	Conn                  *websocket.Conn
	URL                   string
	NegotiatedSubprotocol string
	HandshakeResponse     *http.Response

	readCh  chan *Message
	writeCh chan *Message
	done    chan struct{}

	mu       sync.Mutex
	closed   bool
	lastErr  error
	closeErr error
}

// Start launches the read and write loops for the connection.
func (c *Connection) Start() {
	go c.readLoop()
	go c.writeLoop()
}

// Close closes the underlying websocket connection and signals the loops to stop.
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	close(c.done)
	c.closeErr = c.Conn.Close()
	return c.closeErr
}

// IsClosed returns true if the connection is closed.
func (c *Connection) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

// Err returns the error that caused the connection to close, if any.
func (c *Connection) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastErr
}

// Read returns the channel for incoming messages.
func (c *Connection) Read() <-chan *Message {
	return c.readCh
}

// Write sends a message to the write channel.
func (c *Connection) Write(msg *Message) error {
	select {
	case <-c.done:
		return fmt.Errorf("connection closed")
	default:
	}

	select {
	case c.writeCh <- msg:
		return nil
	case <-c.done:
		return fmt.Errorf("connection closed")
	}
}

func (c *Connection) readLoop() {
	defer c.Close()
	for {
		mt, data, err := c.Conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			if c.lastErr == nil && !c.closed {
				c.lastErr = err
			}
			c.mu.Unlock()
			return
		}

		var msgType MessageType
		switch mt {
		case websocket.TextMessage:
			msgType = TextMessage
		case websocket.BinaryMessage:
			msgType = BinaryMessage
		default:
			continue // Ignore other message types for now
		}

		select {
		case c.readCh <- &Message{Type: msgType, Data: data}:
		case <-c.done:
			return
		}
	}
}

func (c *Connection) writeLoop() {
	defer c.Close()
	for {
		select {
		case msg := <-c.writeCh:
			var mt int
			switch msg.Type {
			case TextMessage:
				mt = websocket.TextMessage
			case BinaryMessage:
				mt = websocket.BinaryMessage
			default:
				continue
			}

			if err := c.Conn.WriteMessage(mt, msg.Data); err != nil {
				c.mu.Lock()
				if c.lastErr == nil && !c.closed {
					c.lastErr = err
				}
				c.mu.Unlock()
				return
			}
		case <-c.done:
			return
		}
	}
}

// NewConnection creates a new Connection wrapper with initialized channels.
func NewConnection(conn *websocket.Conn, url string, resp *http.Response, opts *DialOptions) *Connection {
	readBuf := 1024
	writeBuf := 1024
	if opts.ReadBufferSize > 0 {
		readBuf = opts.ReadBufferSize
	}
	if opts.WriteBufferSize > 0 {
		writeBuf = opts.WriteBufferSize
	}

	return &Connection{
		Conn:                  conn,
		URL:                   url,
		NegotiatedSubprotocol: conn.Subprotocol(),
		HandshakeResponse:     resp,
		readCh:                make(chan *Message, readBuf),
		writeCh:               make(chan *Message, writeBuf),
		done:                  make(chan struct{}),
	}
}
