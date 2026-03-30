package ws

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MessageType represents the type of WebSocket message.
type MessageType int

const (
	// TextMessage is a text WebSocket message.
	TextMessage MessageType = iota
	// BinaryMessage is a binary WebSocket message.
	BinaryMessage
	// PingMessage is a ping WebSocket message.
	PingMessage
	// PongMessage is a pong WebSocket message.
	PongMessage
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

	_readCh  chan *Message
	_writeCh chan *Message
	_done    chan struct{}

	_mu       sync.Mutex
	_closed   bool
	_lastErr  error
	_closeErr error

	_pingInterval time.Duration
	_pongWait     time.Duration
	_verbose      bool
}

// Start launches the read and write loops for the connection.
func (c *Connection) Start() {
	go c.readLoop()
	go c.writeLoop()
}

// Close closes the underlying websocket connection and signals the loops to stop.
func (c *Connection) Close() error {
	c._mu.Lock()
	defer c._mu.Unlock()
	if c._closed {
		return nil
	}
	c._closed = true
	close(c._done)
	c._closeErr = c.Conn.Close()
	return c._closeErr
}

// IsClosed returns true if the connection is closed.
func (c *Connection) IsClosed() bool {
	c._mu.Lock()
	defer c._mu.Unlock()
	return c._closed
}

// Err returns the error that caused the connection to close, if any.
func (c *Connection) Err() error {
	c._mu.Lock()
	defer c._mu.Unlock()
	return c._lastErr
}

// Done returns a channel that is closed when the connection is closed.
func (c *Connection) Done() <-chan struct{} {
	return c._done
}

// Read returns the channel for incoming messages.
func (c *Connection) Read() <-chan *Message {
	return c._readCh
}

// Write sends a message to the write channel.
func (c *Connection) Write(msg *Message) error {
	select {
	case <-c._done:
		return fmt.Errorf("connection closed")
	default:
	}

	select {
	case c._writeCh <- msg:
		return nil
	case <-c._done:
		return fmt.Errorf("connection closed")
	}
}

func (c *Connection) readLoop() {
	defer c.Close()

	if c._pongWait > 0 {
		if err := c.Conn.SetReadDeadline(time.Now().Add(c._pongWait)); err != nil {
			c._mu.Lock()
			if c._lastErr == nil && !c._closed {
				c._lastErr = fmt.Errorf("setting read deadline: %w", err)
			}
			c._mu.Unlock()
			return
		}

		c.Conn.SetPongHandler(func(string) error {
			if err := c.Conn.SetReadDeadline(time.Now().Add(c._pongWait)); err != nil {
				if c._verbose {
					fmt.Fprintf(os.Stderr, "  [ws] error resetting read deadline on pong: %v\n", err)
				}
			}
			if c._verbose {
				fmt.Fprintf(os.Stderr, "  [ws] received pong from %s\n", c.URL)
			}
			return nil
		})
	}

	for {
		mt, data, err := c.Conn.ReadMessage()
		if err != nil {
			c._mu.Lock()
			if c._lastErr == nil && !c._closed {
				c._lastErr = err
			}
			c._mu.Unlock()
			return
		}

		var msgType MessageType
		switch mt {
		case websocket.TextMessage:
			msgType = TextMessage
		case websocket.BinaryMessage:
			msgType = BinaryMessage
		case websocket.PingMessage:
			msgType = PingMessage
		case websocket.PongMessage:
			msgType = PongMessage
		default:
			continue // Ignore other message types for now
		}

		select {
		case c._readCh <- &Message{Type: msgType, Data: data}:
		case <-c._done:
			return
		}
	}
}

func (c *Connection) writeLoop() {
	defer c.Close()

	var ticker *time.Ticker
	if c._pingInterval > 0 {
		ticker = time.NewTicker(c._pingInterval)
		defer ticker.Stop()
	}

	for {
		select {
		case msg := <-c._writeCh:
			var mt int
			switch msg.Type {
			case TextMessage:
				mt = websocket.TextMessage
			case BinaryMessage:
				mt = websocket.BinaryMessage
			case PingMessage:
				mt = websocket.PingMessage
			case PongMessage:
				mt = websocket.PongMessage
			default:
				continue
			}

			if err := c.Conn.WriteMessage(mt, msg.Data); err != nil {
				c._mu.Lock()
				if c._lastErr == nil && !c._closed {
					c._lastErr = err
				}
				c._mu.Unlock()
				return
			}
		case <-func() <-chan time.Time {
			if ticker == nil {
				return nil
			}
			return ticker.C
		}():
			if c._verbose {
				fmt.Fprintf(os.Stderr, "  [ws] sending ping to %s\n", c.URL)
			}
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c._mu.Lock()
				if c._lastErr == nil && !c._closed {
					c._lastErr = err
				}
				c._mu.Unlock()
				return
			}
		case <-c._done:
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
		_readCh:               make(chan *Message, readBuf),
		_writeCh:              make(chan *Message, writeBuf),
		_done:                 make(chan struct{}),
		_pingInterval:         opts.PingInterval,
		_pongWait:             opts.PongWait,
		_verbose:              opts.Verbose,
	}
}
