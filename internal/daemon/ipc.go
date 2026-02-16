package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// MessageType represents the type of IPC message
type MessageType string

const (
	// Request message types
	MessageTypePing         MessageType = "ping"
	MessageTypeUpsertRoute  MessageType = "upsert_route"
	MessageTypeGetRoute     MessageType = "get_route"
	MessageTypeListRoutes   MessageType = "list_routes"
	MessageTypeSetProcess   MessageType = "set_process"
	MessageTypeGetProcess   MessageType = "get_process"
	MessageTypeClearProcess MessageType = "clear_process"
	MessageTypeStatus       MessageType = "status"
	MessageTypeStop         MessageType = "stop"
)

// Request represents an IPC request message
type Request struct {
	Type MessageType     `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// Response represents an IPC response message
type Response struct {
	Ok    bool            `json:"ok"`
	Error string          `json:"error,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// PingData is empty for ping requests
type PingData struct{}

// UpsertRouteData contains parameters for upserting a route
type UpsertRouteData struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// GetRouteData contains parameters for getting a route
type GetRouteData struct {
	Host string `json:"host"`
}

// ListRoutesData is empty for list routes requests
type ListRoutesData struct{}

// SetProcessData contains parameters for setting a process
type SetProcessData struct {
	ProjectRoot string    `json:"projectRoot"`
	PID         int       `json:"pid"`
	Host        string    `json:"host"`
	Port        int       `json:"port"`
	StartedAt   time.Time `json:"startedAt,omitempty"`
}

// GetProcessData contains parameters for getting a process
type GetProcessData struct {
	ProjectRoot string `json:"projectRoot"`
}

// ClearProcessData contains parameters for clearing a process
type ClearProcessData struct {
	ProjectRoot string `json:"projectRoot"`
}

// StatusData is empty for status requests
type StatusData struct{}

// StopData contains parameters for stop command
type StopData struct {
	ClearRoutes bool `json:"clearRoutes,omitempty"`
}

// StatusResponseData contains the current daemon status
type StatusResponseData struct {
	Routes    []Route    `json:"routes"`
	Processes []*Process `json:"processes"`
}

// EncodeRequest encodes a request to JSON line format
func EncodeRequest(w io.Writer, req *Request) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Write JSON followed by newline (JSON lines format)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// DecodeRequest decodes a request from JSON line format
func DecodeRequest(r *bufio.Reader) (*Request, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read request line: %w", err)
	}

	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	return &req, nil
}

// EncodeResponse encodes a response to JSON line format
func EncodeResponse(w io.Writer, resp *Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	// Write JSON followed by newline (JSON lines format)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

// DecodeResponse decodes a response from JSON line format
func DecodeResponse(r *bufio.Reader) (*Response, error) {
	line, err := r.ReadBytes('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read response line: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &resp, nil
}

// NewSuccessResponse creates a success response with optional data
func NewSuccessResponse(data interface{}) (*Response, error) {
	resp := &Response{
		Ok: true,
	}

	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal response data: %w", err)
		}
		resp.Data = jsonData
	}

	return resp, nil
}

// NewErrorResponse creates an error response with an error message
func NewErrorResponse(err error) *Response {
	return &Response{
		Ok:    false,
		Error: err.Error(),
	}
}

// NewRequest creates a new request with the given type and data
func NewRequest(msgType MessageType, data interface{}) (*Request, error) {
	req := &Request{
		Type: msgType,
	}

	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
		req.Data = jsonData
	}

	return req, nil
}
