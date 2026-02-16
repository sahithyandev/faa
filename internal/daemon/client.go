package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

// Client represents a client connection to the daemon
type Client struct {
	conn net.Conn
}

// Connect connects to the daemon via Unix socket
func Connect() (*Client, error) {
	sockPath, err := SocketPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get socket path: %w", err)
	}

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to daemon: %w", err)
	}

	return &Client{conn: conn}, nil
}

// Close closes the connection to the daemon
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// sendRequest sends a request and waits for a response
func (c *Client) sendRequest(req *Request) (*Response, error) {
	// Encode and send request
	if err := EncodeRequest(c.conn, req); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	reader := bufio.NewReader(c.conn)
	resp, err := DecodeResponse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return resp, nil
}

// Ping sends a ping request to check if daemon is alive
func (c *Client) Ping() error {
	req, err := NewRequest(MessageTypePing, nil)
	if err != nil {
		return err
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return err
	}

	if !resp.Ok {
		return fmt.Errorf("ping failed: %s", resp.Error)
	}

	return nil
}

// UpsertRoute adds or updates a route in the daemon
func (c *Client) UpsertRoute(host string, port int) error {
	req, err := NewRequest(MessageTypeUpsertRoute, &UpsertRouteData{
		Host: host,
		Port: port,
	})
	if err != nil {
		return err
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return err
	}

	if !resp.Ok {
		return fmt.Errorf("upsert_route failed: %s", resp.Error)
	}

	return nil
}

// SetProcess registers a process in the daemon
func (c *Client) SetProcess(data *SetProcessData) error {
	req, err := NewRequest(MessageTypeSetProcess, data)
	if err != nil {
		return err
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return err
	}

	if !resp.Ok {
		return fmt.Errorf("set_process failed: %s", resp.Error)
	}

	return nil
}

// GetProcess retrieves a process from the daemon registry
func (c *Client) GetProcess(projectRoot string) (*Process, error) {
	req, err := NewRequest(MessageTypeGetProcess, &GetProcessData{
		ProjectRoot: projectRoot,
	})
	if err != nil {
		return nil, err
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	if !resp.Ok {
		return nil, fmt.Errorf("get_process failed: %s", resp.Error)
	}

	// If data is null or empty, no process found
	if resp.Data == nil || len(resp.Data) == 0 || string(resp.Data) == "null" {
		return nil, nil
	}

	var proc Process
	if err := json.Unmarshal(resp.Data, &proc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal process: %w", err)
	}

	return &proc, nil
}

// ClearProcess removes a process from the daemon registry
func (c *Client) ClearProcess(projectRoot string) error {
	req, err := NewRequest(MessageTypeClearProcess, &ClearProcessData{
		ProjectRoot: projectRoot,
	})
	if err != nil {
		return err
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return err
	}

	if !resp.Ok {
		return fmt.Errorf("clear_process failed: %s", resp.Error)
	}

	return nil
}
