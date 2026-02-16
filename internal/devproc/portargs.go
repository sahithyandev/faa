package devproc

import "fmt"

// InjectPort takes a command with its arguments and a port number,
// and returns the modified arguments with "--port <port>" appended.
// It also returns an environment map with PORT set to the port value.
func InjectPort(command []string, port int) ([]string, map[string]string) {
	if command == nil || len(command) == 0 {
		return nil, nil
	}

	// Create new args slice with port flag appended
	newArgs := make([]string, len(command)+2)
	copy(newArgs, command)
	newArgs[len(command)] = "--port"
	newArgs[len(command)+1] = fmt.Sprintf("%d", port)

	// Create environment map with PORT
	env := map[string]string{
		"PORT": fmt.Sprintf("%d", port),
	}

	return newArgs, env
}
