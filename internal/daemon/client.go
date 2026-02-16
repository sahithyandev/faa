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

// GetRoute retrieves the port for a specific host from the daemon
// Returns 0 if no route exists for the host
func (c *Client) GetRoute(host string) (int, error) {
	req, err := NewRequest(MessageTypeGetRoute, &GetRouteData{
		Host: host,
	})
	if err != nil {
		return 0, err
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return 0, err
	}

	if !resp.Ok {
		return 0, fmt.Errorf("get_route failed: %s", resp.Error)
	}

	var result map[string]int
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return 0, fmt.Errorf("failed to unmarshal route: %w", err)
	}

	return result["port"], nil
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
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
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

// Status retrieves the current daemon status including routes and processes
func (c *Client) Status() (*StatusResponseData, error) {
	req, err := NewRequest(MessageTypeStatus, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	if !resp.Ok {
		return nil, fmt.Errorf("status failed: %s", resp.Error)
	}

	var status StatusResponseData
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal status: %w", err)
	}

	return &status, nil
}

// Stop sends a stop request to the daemon
func (c *Client) Stop(clearRoutes bool) error {
	req, err := NewRequest(MessageTypeStop, &StopData{
		ClearRoutes: clearRoutes,
	})
	if err != nil {
		return err
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return err
	}

	if !resp.Ok {
		return fmt.Errorf("stop failed: %s", resp.Error)
	}

	return nil
}

// ListRoutes retrieves all routes from the daemon
func (c *Client) ListRoutes() ([]Route, error) {
	req, err := NewRequest(MessageTypeListRoutes, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, err
	}

	if !resp.Ok {
		return nil, fmt.Errorf("list_routes failed: %s", resp.Error)
	}

	var routes []Route
	if err := json.Unmarshal(resp.Data, &routes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal routes: %w", err)
	}

	return routes, nil
}
