package ws

import (
	"fmt"
	"net/http"
	"os"
	"strings"
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

	_pingInterval  time.Duration
	_pongWait      time.Duration
	_verbose       bool
	_maxMessageSize       int64
	_maxFrameSize         int
	_compressionRequested bool

	_closeCode    int
	_closeReason  string
	_onDisconnect func(code int, reason string)
	_closing      chan struct{}
}

// Start launches the read and write loops for the connection.
func (c *Connection) Start() {
	go c.readLoop()
	go c.writeLoop()
}

// Close closes the underlying websocket connection and signals the loops to stop.
// CloseWithCode closes the connection with a specific code and reason gracefully.
func (c *Connection) CloseWithCode(code int, reason string) error {
	c._mu.Lock()
	if c._closed {
		c._mu.Unlock()
		return nil
	}

	// Check if already closing
	select {
	case <-c._closing:
		c._mu.Unlock()
		return nil
	default:
	}

	c._closeCode = code
	c._closeReason = reason
	close(c._closing)
	
	if c._verbose {
		fmt.Fprintf(os.Stderr, "  [ws] initiating graceful close: %d %s\n", code, reason)
	}
	c._mu.Unlock()

	// Wait for the done channel to be closed by loops
	select {
	case <-c._done:
	case <-time.After(2 * time.Second): // Timeout for graceful flush
		if c._verbose {
			fmt.Fprintf(os.Stderr, "  [ws] graceful close timed out, forcing closure\n")
		}
		c.forceClose()
	}
	
	return c._closeErr
}

// CloseStatus returns the close code and reason for the connection.
func (c *Connection) CloseStatus() (int, string) {
	c._mu.Lock()
	defer c._mu.Unlock()
	return c._closeCode, c._closeReason
}

// Close closes the underlying websocket connection gracefully (1000 Normal Closure).
func (c *Connection) Close() error {
	return c.CloseWithCode(websocket.CloseNormalClosure, "")
}

// forceClose performs an immediate, non-graceful closure.
func (c *Connection) forceClose() {
	c._mu.Lock()
	if c._closed {
		c._mu.Unlock()
		return
	}
	c._closed = true
	select {
	case <-c._done:
	default:
		close(c._done)
	}
	c._closeErr = c.Conn.Close()
	c._mu.Unlock()
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

// IsCompressionEnabled returns true if per-message-deflate compression was negotiated.
func (c *Connection) IsCompressionEnabled() bool {
	if c.HandshakeResponse == nil {
		return false
	}
	// Check all values of the Sec-WebSocket-Extensions header
	for _, ext := range c.HandshakeResponse.Header.Values("Sec-WebSocket-Extensions") {
		if strings.Contains(strings.ToLower(ext), "permessage-deflate") {
			return true
		}
	}
	return false
}

// CompressionRequested returns true if compression was requested during handshake.
func (c *Connection) CompressionRequested() bool {
	return c._compressionRequested
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

	if c._maxMessageSize > 0 && int64(len(msg.Data)) > c._maxMessageSize {
		return fmt.Errorf("message size %d exceeds limit of %d", len(msg.Data), c._maxMessageSize)
	}

	select {
	case c._writeCh <- msg:
		return nil
	case <-c._done:
		return fmt.Errorf("connection closed")
	}
}

func (c *Connection) readLoop() {
	defer func() {
		c.forceClose()
		if c._onDisconnect != nil {
			c._mu.Lock()
			code := c._closeCode
			reason := c._closeReason
			c._mu.Unlock()
			c._onDisconnect(code, reason)
		}
	}()

	c.Conn.SetCloseHandler(func(code int, text string) error {
		c._mu.Lock()
		c._closeCode = code
		c._closeReason = text
		if c._verbose {
			fmt.Fprintf(os.Stderr, "  [ws] received close frame from %s: %d %s\n", c.URL, code, text)
		}
		c._mu.Unlock()
		
		// The default handler sends a close message and returns.
		// We want to ensure the readLoop gets the error.
		message := websocket.FormatCloseMessage(code, "")
		_ = c.Conn.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		return nil
	})

	if c._pongWait > 0 {
		if err := c.Conn.SetReadDeadline(time.Now().Add(c._pongWait)); err != nil {
			c._mu.Lock()
			if c._lastErr == nil && !c._closed {
				c._lastErr = fmt.Errorf("setting read deadline: %w", err)
			}
			c._mu.Unlock()
			return
		}

		c.Conn.SetPingHandler(func(data string) error {
			if c._verbose {
				fmt.Fprintf(os.Stderr, "  [ws] received ping message from %s (%d bytes)\n", c.URL, len(data))
			}
			// Gorilla handles the pong reply by default
			return nil
		})

		c.Conn.SetPongHandler(func(data string) error {
			if err := c.Conn.SetReadDeadline(time.Now().Add(c._pongWait)); err != nil {
				if c._verbose {
					fmt.Fprintf(os.Stderr, "  [ws] error resetting read deadline on pong: %v\n", err)
				}
			}
			if c._verbose {
				fmt.Fprintf(os.Stderr, "  [ws] received pong message from %s (%d bytes)\n", c.URL, len(data))
			}
			return nil
		})
	}

	for {
		mt, data, err := c.Conn.ReadMessage()
		if err != nil {
			c._mu.Lock()
			if c._lastErr == nil {
				c._lastErr = err
			}
			
			// If we haven't captured a specific close code yet, do it now
			if ce, ok := err.(*websocket.CloseError); ok {
				c._closeCode = ce.Code
				c._closeReason = ce.Text
			} else if c._closeCode == 1000 && !c._closed {
				// If it's not a clean close and we didn't initiate closure, it's abnormal
				c._closeCode = websocket.CloseAbnormalClosure
				c._closeReason = err.Error()
			}
			c._mu.Unlock()
			return
		}

		var msgType MessageType
		var typeStr string
		switch mt {
		case websocket.TextMessage:
			msgType = TextMessage
			typeStr = "text"
		case websocket.BinaryMessage:
			msgType = BinaryMessage
			typeStr = "binary"
		default:
			continue // Ignore other message types as they are handled by handlers
		}

		if c._verbose {
			fmt.Fprintf(os.Stderr, "  [ws] received %s message from %s (%d bytes)\n", typeStr, c.URL, len(data))
		}

		select {
		case c._readCh <- &Message{Type: msgType, Data: data}:
		case <-c._done:
			return
		}
	}
}

func (c *Connection) writeLoop() {
	defer c.forceClose()

	var ticker *time.Ticker
	if c._pingInterval > 0 {
		ticker = time.NewTicker(c._pingInterval)
		defer ticker.Stop()
	}

	for {
		select {
		case msg := <-c._writeCh:
			if err := c.sendMessage(msg); err != nil {
				return
			}
		case <-func() <-chan time.Time {
			if ticker == nil {
				return nil
			}
			return ticker.C
		}():
			if c._verbose {
				fmt.Fprintf(os.Stderr, "  [ws] sending ping message to %s (0 bytes)\n", c.URL)
			}
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c._mu.Lock()
				if c._lastErr == nil && !c._closed {
					c._lastErr = err
				}
				c._mu.Unlock()
				return
			}
		case <-c._closing:
			// Flush pending messages
			for {
				select {
				case msg := <-c._writeCh:
					if err := c.sendMessage(msg); err != nil {
						return
					}
				default:
					// All messages flushed, send close frame
					c._mu.Lock()
					code := c._closeCode
					reason := c._closeReason
					c._mu.Unlock()

					if c._verbose {
						fmt.Fprintf(os.Stderr, "  [ws] sending close frame to %s: %d %s\n", c.URL, code, reason)
					}
					_ = c.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason))
					return
				}
			}
		case <-c._done:
			return
		}
	}
}

func (c *Connection) sendMessage(msg *Message) error {
	var mt int
	var typeStr string
	switch msg.Type {
	case TextMessage:
		mt = websocket.TextMessage
		typeStr = "text"
	case BinaryMessage:
		mt = websocket.BinaryMessage
		typeStr = "binary"
	case PingMessage:
		mt = websocket.PingMessage
		typeStr = "ping"
	case PongMessage:
		mt = websocket.PongMessage
		typeStr = "pong"
	default:
		return nil
	}

	if c._verbose {
		fmt.Fprintf(os.Stderr, "  [ws] sending %s message to %s (%d bytes)\n", typeStr, c.URL, len(msg.Data))
	}

	var err error
	if c._maxFrameSize > 0 && (msg.Type == TextMessage || msg.Type == BinaryMessage) && len(msg.Data) > c._maxFrameSize {
		err = c.writeFragmented(mt, msg.Data)
	} else {
		err = c.Conn.WriteMessage(mt, msg.Data)
	}

	if err != nil {
		c._mu.Lock()
		if c._lastErr == nil && !c._closed {
			c._lastErr = err
		}
		c._mu.Unlock()
		return err
	}
	return nil
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

	conn.SetReadLimit(opts.MaxMessageSize)
	c := &Connection{
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
		_maxMessageSize:       opts.MaxMessageSize,
		_maxFrameSize:         opts.MaxFrameSize,
		_compressionRequested: opts.Compress,
		_onDisconnect:         opts.OnDisconnect,
		_closing:              make(chan struct{}),
		_closeCode:            1000, // Default to normal closure
	}

	if c.IsCompressionEnabled() {
		conn.EnableWriteCompression(true)
	}

	if opts.Verbose && resp != nil {
		if ext := resp.Header.Values("Sec-WebSocket-Extensions"); len(ext) > 0 {
			fmt.Fprintf(os.Stderr, "  [ws] extensions negotiated: %s\n", strings.Join(ext, ", "))
		}
	}

	return c
}

func (c *Connection) writeFragmented(messageType int, data []byte) error {
	w, err := c.Conn.NextWriter(messageType)
	if err != nil {
		return err
	}

	for i := 0; i < len(data); i += c._maxFrameSize {
		end := i + c._maxFrameSize
		if end > len(data) {
			end = len(data)
		}

		if c._verbose {
			fmt.Fprintf(os.Stderr, "  [ws] sending frame: %d-%d bytes\n", i, end)
		}

		if _, err := w.Write(data[i:end]); err != nil {
			_ = w.Close()
			return err
		}
	}

	return w.Close()
}
