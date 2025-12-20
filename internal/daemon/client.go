package daemon

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
)

// DefaultSocketPath returns the standard daemon socket location.
func DefaultSocketPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/openkanban/daemon.sock"
	}
	return filepath.Join(home, ".openkanban", "daemon.sock")
}

// Client connects to the daemon server over Unix socket.
type Client struct {
	socketPath string
	conn       net.Conn
	mu         sync.Mutex
	connected  bool
}

// NewClient creates a new daemon client.
func NewClient(socketPath string) *Client {
	if socketPath == "" {
		socketPath = DefaultSocketPath()
	}
	return &Client{
		socketPath: socketPath,
	}
}

// Connect establishes connection to the daemon.
func (c *Client) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	conn, err := net.Dial("unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("connect to daemon: %w", err)
	}

	c.conn = conn
	c.connected = true
	return nil
}

// Close disconnects from the daemon.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	c.connected = false
	return err
}

// IsConnected returns true if client is connected to daemon.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}

// Write sends data to the daemon PTY.
func (c *Client) Write(data []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return 0, fmt.Errorf("not connected")
	}

	err := WriteMessage(c.conn, Message{
		Type: MsgData,
		Data: data,
	})
	if err != nil {
		c.handleDisconnect()
		return 0, err
	}

	return len(data), nil
}

// ReadMessage reads a single message from the daemon.
// Returns the message type and data.
func (c *Client) ReadMessage() (byte, []byte, error) {
	c.mu.Lock()
	conn := c.conn
	connected := c.connected
	c.mu.Unlock()

	if !connected || conn == nil {
		return 0, nil, fmt.Errorf("not connected")
	}

	msg, err := ReadMessage(conn)
	if err != nil {
		c.mu.Lock()
		c.handleDisconnect()
		c.mu.Unlock()
		return 0, nil, err
	}

	return msg.Type, msg.Data, nil
}

// Resize sends terminal dimensions to the daemon.
func (c *Client) Resize(rows, cols uint16) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return fmt.Errorf("not connected")
	}

	err := WriteMessage(c.conn, Message{
		Type: MsgResize,
		Data: EncodeResize(rows, cols),
	})
	if err != nil {
		c.handleDisconnect()
		return err
	}

	return nil
}

func (c *Client) SocketPath() string {
	return c.socketPath
}

func (c *Client) CreateSession(sessionID, workdir, command string, args []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return fmt.Errorf("not connected")
	}

	err := WriteMessage(c.conn, Message{
		Type: MsgCreate,
		Data: EncodeCreate(sessionID, workdir, command, args),
	})
	if err != nil {
		c.handleDisconnect()
		return err
	}

	return c.waitSessionResponse()
}

func (c *Client) AttachSession(sessionID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return fmt.Errorf("not connected")
	}

	err := WriteMessage(c.conn, Message{
		Type: MsgAttach,
		Data: EncodeAttach(sessionID),
	})
	if err != nil {
		c.handleDisconnect()
		return err
	}

	return c.waitSessionResponse()
}

func (c *Client) waitSessionResponse() error {
	msg, err := ReadMessage(c.conn)
	if err != nil {
		c.handleDisconnect()
		return err
	}

	switch msg.Type {
	case MsgSessionOK:
		return nil
	case MsgSessionError:
		return fmt.Errorf("%s", string(msg.Data))
	default:
		return fmt.Errorf("unexpected response: %d", msg.Type)
	}
}

func (c *Client) handleDisconnect() {
	c.connected = false
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
}

func (c *Client) ListSessions() ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	err := WriteMessage(c.conn, Message{Type: MsgList})
	if err != nil {
		c.handleDisconnect()
		return nil, err
	}

	msg, err := ReadMessage(c.conn)
	if err != nil {
		c.handleDisconnect()
		return nil, err
	}

	if msg.Type != MsgListResponse {
		return nil, fmt.Errorf("unexpected response: %d", msg.Type)
	}

	if len(msg.Data) == 0 {
		return nil, nil
	}

	var ids []string
	start := 0
	for i, b := range msg.Data {
		if b == 0 {
			ids = append(ids, string(msg.Data[start:i]))
			start = i + 1
		}
	}
	if start < len(msg.Data) {
		ids = append(ids, string(msg.Data[start:]))
	}
	return ids, nil
}

// DaemonAvailable checks if the daemon socket exists and is connectable.
func DaemonAvailable(socketPath string) bool {
	if socketPath == "" {
		socketPath = DefaultSocketPath()
	}

	info, err := os.Stat(socketPath)
	if err != nil {
		return false
	}

	if info.Mode()&os.ModeSocket == 0 {
		return false
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	conn.Close()

	return true
}
