package devproc

import (
	"fmt"
	"reflect"
	"testing"
)

func TestInjectPort_Basic(t *testing.T) {
	tests := []struct {
		name        string
		command     []string
		port        int
		wantArgs    []string
		wantEnvPort string
	}{
		{
			name:        "simple command with port",
			command:     []string{"npm", "start"},
			port:        3000,
			wantArgs:    []string{"npm", "start", "--port", "3000"},
			wantEnvPort: "3000",
		},
		{
			name:        "command with existing args",
			command:     []string{"node", "server.js", "--verbose"},
			port:        8080,
			wantArgs:    []string{"node", "server.js", "--verbose", "--port", "8080"},
			wantEnvPort: "8080",
		},
		{
			name:        "single command",
			command:     []string{"python"},
			port:        5000,
			wantArgs:    []string{"python", "--port", "5000"},
			wantEnvPort: "5000",
		},
		{
			name:        "high port number",
			command:     []string{"go", "run", "main.go"},
			port:        45678,
			wantArgs:    []string{"go", "run", "main.go", "--port", "45678"},
			wantEnvPort: "45678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArgs, gotEnv := InjectPort(tt.command, tt.port)

			// Check args
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("InjectPort() args = %v, want %v", gotArgs, tt.wantArgs)
			}

			// Check environment
			if gotEnv == nil {
				t.Fatal("InjectPort() env is nil, want map with PORT")
			}
			if gotEnv["PORT"] != tt.wantEnvPort {
				t.Errorf("InjectPort() env[PORT] = %v, want %v", gotEnv["PORT"], tt.wantEnvPort)
			}
		})
	}
}

func TestInjectPort_EmptyCommand(t *testing.T) {
	gotArgs, gotEnv := InjectPort([]string{}, 3000)

	// Should return nil for both args and env for empty command
	if gotArgs != nil {
		t.Errorf("InjectPort([]string{}, 3000) args = %v, want nil", gotArgs)
	}
	if gotEnv != nil {
		t.Errorf("InjectPort([]string{}, 3000) env = %v, want nil", gotEnv)
	}
}

func TestInjectPort_NilCommand(t *testing.T) {
	gotArgs, gotEnv := InjectPort(nil, 3000)

	// Should handle nil command gracefully
	if gotArgs != nil {
		t.Errorf("InjectPort(nil, 3000) args = %v, want nil", gotArgs)
	}
	if gotEnv != nil {
		t.Errorf("InjectPort(nil, 3000) env = %v, want nil", gotEnv)
	}
}

func TestInjectPort_PreservesOriginal(t *testing.T) {
	// Test that original command slice is not modified
	original := []string{"node", "app.js"}
	originalCopy := make([]string, len(original))
	copy(originalCopy, original)

	InjectPort(original, 8080)

	if !reflect.DeepEqual(original, originalCopy) {
		t.Errorf("InjectPort() modified original slice: got %v, want %v", original, originalCopy)
	}
}

func TestInjectPort_PortValues(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{name: "port 80", port: 80},
		{name: "port 443", port: 443},
		{name: "port 1024", port: 1024},
		{name: "port 3000", port: 3000},
		{name: "port 8080", port: 8080},
		{name: "port 49151", port: 49151},
		{name: "port 65535", port: 65535},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := []string{"server"}
			gotArgs, gotEnv := InjectPort(command, tt.port)

			// Verify port flag is correct
			if len(gotArgs) < 2 {
				t.Fatalf("InjectPort() returned too few args: %v", gotArgs)
			}
			gotPortFlag := gotArgs[len(gotArgs)-2]
			gotPortArg := gotArgs[len(gotArgs)-1]

			if gotPortFlag != "--port" {
				t.Errorf("InjectPort() port flag = %v, want --port", gotPortFlag)
			}

			expectedPortStr := fmt.Sprintf("%d", tt.port)
			if gotPortArg != expectedPortStr {
				t.Errorf("InjectPort() port value = %v, want %v", gotPortArg, expectedPortStr)
			}

			// Verify environment
			if gotEnv["PORT"] != expectedPortStr {
				t.Errorf("InjectPort() env[PORT] = %v, want %v", gotEnv["PORT"], expectedPortStr)
			}
		})
	}
}

func TestInjectPort_EnvironmentMapStructure(t *testing.T) {
	command := []string{"npm", "start"}
	port := 3000

	_, gotEnv := InjectPort(command, port)

	// Verify environment map has exactly one entry
	if len(gotEnv) != 1 {
		t.Errorf("InjectPort() env has %d entries, want 1", len(gotEnv))
	}

	// Verify PORT key exists
	if _, ok := gotEnv["PORT"]; !ok {
		t.Error("InjectPort() env missing PORT key")
	}
}

func TestInjectPort_ArgsOrderPreserved(t *testing.T) {
	// Verify that original args order is preserved
	command := []string{"cmd", "arg1", "arg2", "arg3"}
	port := 5000

	gotArgs, _ := InjectPort(command, port)

	// Check that original args are in the same order
	for i := 0; i < len(command); i++ {
		if gotArgs[i] != command[i] {
			t.Errorf("InjectPort() arg[%d] = %v, want %v", i, gotArgs[i], command[i])
		}
	}

	// Check that port flag is appended at the end
	if gotArgs[len(gotArgs)-2] != "--port" {
		t.Errorf("InjectPort() second-to-last arg = %v, want --port", gotArgs[len(gotArgs)-2])
	}
	if gotArgs[len(gotArgs)-1] != "5000" {
		t.Errorf("InjectPort() last arg = %v, want 5000", gotArgs[len(gotArgs)-1])
	}
}
