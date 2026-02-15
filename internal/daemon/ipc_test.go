package daemon

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestEncodeDecodeRequest(t *testing.T) {
	tests := []struct {
		name    string
		msgType MessageType
		data    interface{}
	}{
		{
			name:    "ping request",
			msgType: MessageTypePing,
			data:    nil,
		},
		{
			name:    "upsert_route request",
			msgType: MessageTypeUpsertRoute,
			data: &UpsertRouteData{
				Host: "example.com",
				Port: 8080,
			},
		},
		{
			name:    "list_routes request",
			msgType: MessageTypeListRoutes,
			data:    nil,
		},
		{
			name:    "set_process request",
			msgType: MessageTypeSetProcess,
			data: &SetProcessData{
				ProjectRoot: "/path/to/project",
				PID:         12345,
				Host:        "localhost",
				Port:        3000,
				StartedAt:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			},
		},
		{
			name:    "clear_process request",
			msgType: MessageTypeClearProcess,
			data: &ClearProcessData{
				ProjectRoot: "/path/to/project",
			},
		},
		{
			name:    "status request",
			msgType: MessageTypeStatus,
			data:    nil,
		},
		{
			name:    "stop request without clearRoutes",
			msgType: MessageTypeStop,
			data: &StopData{
				ClearRoutes: false,
			},
		},
		{
			name:    "stop request with clearRoutes",
			msgType: MessageTypeStop,
			data: &StopData{
				ClearRoutes: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req, err := NewRequest(tt.msgType, tt.data)
			if err != nil {
				t.Fatalf("NewRequest() failed: %v", err)
			}

			// Encode request
			var buf bytes.Buffer
			if err := EncodeRequest(&buf, req); err != nil {
				t.Fatalf("EncodeRequest() failed: %v", err)
			}

			// Verify it contains a newline
			if !bytes.Contains(buf.Bytes(), []byte("\n")) {
				t.Error("Encoded request does not contain newline")
			}

			// Decode request
			reader := bufio.NewReader(&buf)
			decoded, err := DecodeRequest(reader)
			if err != nil {
				t.Fatalf("DecodeRequest() failed: %v", err)
			}

			// Verify type matches
			if decoded.Type != tt.msgType {
				t.Errorf("Type = %v, want %v", decoded.Type, tt.msgType)
			}

			// If there's data, verify it round-trips correctly
			if tt.data != nil {
				// Re-unmarshal the data field to compare
				var originalJSON, decodedJSON []byte
				var err error

				originalJSON, err = json.Marshal(tt.data)
				if err != nil {
					t.Fatalf("Failed to marshal original data: %v", err)
				}

				decodedJSON = decoded.Data

				// Compare JSON representations
				var original, decodedData interface{}
				if err := json.Unmarshal(originalJSON, &original); err != nil {
					t.Fatalf("Failed to unmarshal original JSON: %v", err)
				}
				if err := json.Unmarshal(decodedJSON, &decodedData); err != nil {
					t.Fatalf("Failed to unmarshal decoded JSON: %v", err)
				}

				if !reflect.DeepEqual(original, decodedData) {
					t.Errorf("Data mismatch:\nOriginal: %v\nDecoded:  %v", original, decodedData)
				}
			}
		})
	}
}

func TestEncodeDecodeResponse(t *testing.T) {
	tests := []struct {
		name     string
		response *Response
	}{
		{
			name: "success response without data",
			response: &Response{
				Ok: true,
			},
		},
		{
			name: "success response with data",
			response: &Response{
				Ok:   true,
				Data: json.RawMessage(`{"routes":[{"host":"example.com","port":8080}]}`),
			},
		},
		{
			name: "error response",
			response: &Response{
				Ok:    false,
				Error: "something went wrong",
			},
		},
		{
			name: "error response with details",
			response: &Response{
				Ok:    false,
				Error: "failed to parse configuration: invalid port number",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encode response
			var buf bytes.Buffer
			if err := EncodeResponse(&buf, tt.response); err != nil {
				t.Fatalf("EncodeResponse() failed: %v", err)
			}

			// Verify it contains a newline
			if !bytes.Contains(buf.Bytes(), []byte("\n")) {
				t.Error("Encoded response does not contain newline")
			}

			// Decode response
			reader := bufio.NewReader(&buf)
			decoded, err := DecodeResponse(reader)
			if err != nil {
				t.Fatalf("DecodeResponse() failed: %v", err)
			}

			// Verify fields match
			if decoded.Ok != tt.response.Ok {
				t.Errorf("Ok = %v, want %v", decoded.Ok, tt.response.Ok)
			}
			if decoded.Error != tt.response.Error {
				t.Errorf("Error = %q, want %q", decoded.Error, tt.response.Error)
			}
			if !bytes.Equal(decoded.Data, tt.response.Data) {
				t.Errorf("Data mismatch:\nGot:  %s\nWant: %s", decoded.Data, tt.response.Data)
			}
		})
	}
}

func TestNewSuccessResponse(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		wantErr bool
	}{
		{
			name:    "nil data",
			data:    nil,
			wantErr: false,
		},
		{
			name: "simple data",
			data: map[string]string{
				"message": "pong",
			},
			wantErr: false,
		},
		{
			name: "complex data",
			data: &StatusResponseData{
				Routes: []Route{
					{Host: "example.com", Port: 8080},
					{Host: "test.com", Port: 3000},
				},
				Processes: []*Process{
					{
						ProjectRoot: "/path/to/project",
						PID:         12345,
						Host:        "localhost",
						Port:        3000,
						StartedAt:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := NewSuccessResponse(tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewSuccessResponse() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {
				if !resp.Ok {
					t.Error("Response Ok = false, want true")
				}
				if resp.Error != "" {
					t.Errorf("Response Error = %q, want empty", resp.Error)
				}

				if tt.data != nil {
					if resp.Data == nil {
						t.Error("Response Data is nil, but data was provided")
					}

					// Verify we can unmarshal it back
					var unmarshaled interface{}
					if err := json.Unmarshal(resp.Data, &unmarshaled); err != nil {
						t.Errorf("Failed to unmarshal response data: %v", err)
					}
				} else {
					if resp.Data != nil {
						t.Errorf("Response Data = %v, want nil", resp.Data)
					}
				}
			}
		})
	}
}

func TestNewErrorResponse(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantError string
	}{
		{
			name:      "simple error",
			err:       ErrNotFound,
			wantError: "not found",
		},
		{
			name:      "formatted error",
			err:       fmt.Errorf("failed to connect: %w", ErrNotFound),
			wantError: "failed to connect: not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewErrorResponse(tt.err)

			if resp.Ok {
				t.Error("Response Ok = true, want false")
			}
			if resp.Error != tt.wantError {
				t.Errorf("Response Error = %q, want %q", resp.Error, tt.wantError)
			}
			if resp.Data != nil {
				t.Errorf("Response Data = %v, want nil", resp.Data)
			}
		})
	}
}

func TestRoundTripCompleteFlow(t *testing.T) {
	// Test a complete request-response cycle for different operations
	testCases := []struct {
		name         string
		request      *Request
		responseData interface{}
	}{
		{
			name: "ping request-response",
			request: &Request{
				Type: MessageTypePing,
			},
			responseData: map[string]string{"message": "pong"},
		},
		{
			name: "list_routes request-response",
			request: &Request{
				Type: MessageTypeListRoutes,
			},
			responseData: []Route{
				{Host: "example.com", Port: 8080},
				{Host: "test.com", Port: 3000},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encode request
			var reqBuf bytes.Buffer
			if err := EncodeRequest(&reqBuf, tc.request); err != nil {
				t.Fatalf("EncodeRequest() failed: %v", err)
			}

			// Decode request
			reqReader := bufio.NewReader(&reqBuf)
			decodedReq, err := DecodeRequest(reqReader)
			if err != nil {
				t.Fatalf("DecodeRequest() failed: %v", err)
			}

			if decodedReq.Type != tc.request.Type {
				t.Errorf("Request type = %v, want %v", decodedReq.Type, tc.request.Type)
			}

			// Create and encode response
			resp, err := NewSuccessResponse(tc.responseData)
			if err != nil {
				t.Fatalf("NewSuccessResponse() failed: %v", err)
			}

			var respBuf bytes.Buffer
			if err := EncodeResponse(&respBuf, resp); err != nil {
				t.Fatalf("EncodeResponse() failed: %v", err)
			}

			// Decode response
			respReader := bufio.NewReader(&respBuf)
			decodedResp, err := DecodeResponse(respReader)
			if err != nil {
				t.Fatalf("DecodeResponse() failed: %v", err)
			}

			if !decodedResp.Ok {
				t.Errorf("Response Ok = false, want true")
			}
			if decodedResp.Error != "" {
				t.Errorf("Response Error = %q, want empty", decodedResp.Error)
			}
		})
	}
}

// ErrNotFound is a sample error for testing
var ErrNotFound = errors.New("not found")
