# faa

faa is a CLI tool for streamlining local development workflows. It automatically manages ports, routes, and HTTPS certificates for multiple Node.js projects, making it easy to run and access your development servers.

## Status

Current Stage: Beta - Core functionality is stable and tested. The project is ready for early adopters.

Working Features:
- Daemon-based process management
- Automatic stable port allocation per project
- HTTPS reverse proxy with local CA (powered by Caddy)
- IPC-based communication between CLI and daemon
- Project detection via package.json
- Process lifecycle management

Known Limitations:
- `setup` command is not yet implemented
- Daemon auto-restart not implemented
- Configuration directory is hardcoded to `~/.config/faa`

## Key Features

- Stable Port Assignment: Each project gets a deterministic port based on its name hash, so ports stay consistent across restarts
- Automatic HTTPS: Embedded Caddy reverse proxy provides local HTTPS with automatic certificate management
- Daemon Architecture: Long-running background daemon manages routes and processes
- No Port Conflicts: Automatically avoids common development ports (3000, 8080, etc.)
- Process Management: Track and manage multiple running dev servers from a single daemon

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                  faa CLI Client                     │
│  (run, status, stop, routes commands)               │
└────────────────┬────────────────────────────────────┘
                 │ Unix Socket IPC
                 ▼
┌─────────────────────────────────────────────────────┐
│              faa Daemon Process                     │
│  • Registry (routes.json, processes.json)           │
│  • Process lifecycle management                     │
│  • Route management                                 │
└────────────────┬────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────┐
│          Embedded Caddy Proxy                       │
│  • HTTPS termination with local CA                  │
│  • Dynamic route updates                            │
│  • Host → localhost:port routing                    │
└─────────────────────────────────────────────────────┘
```

## Installation

### Using Go

```bash
go install github.com/sahithyandev/faa/cmd/faa@latest
```

### Building from Source

```bash
# Using just
just build
# Binary will be available at bin/faa

# Or using Go directly
go build -o bin/faa ./cmd/faa
```

## Usage

### Starting the Daemon

First, start the daemon in the background:

```bash
faa daemon &
# Or run in a separate terminal
faa daemon
```

### Running a Project

Navigate to your Node.js project directory and run:

```bash
faa run -- npm start
# or simply
faa npm start
```

faa will:
1. Find your project root (via package.json)
2. Assign a stable port
3. Configure HTTPS route (e.g., https://my-project-name.local)
4. Start your dev server with PORT injected
5. Display the HTTPS URL

### Checking Status

View all running projects and routes:

```bash
faa status
```

Output example:
```
Daemon Status: Running

Routes:
  my-project.local -> localhost:12345
  another-app.local -> localhost:23456

Running Processes:
  PID 12345: /home/user/projects/my-project (https://my-project.local, port 12345)
  PID 23456: /home/user/projects/another-app (https://another-app.local, port 23456)
```

### Viewing Routes

List all configured routes:

```bash
faa routes
```

### Stopping the Daemon

```bash
faa stop
# Or clear routes when stopping
faa stop --clear-routes
```

## Configuration

faa stores its configuration and state in `~/.config/faa/`:
- `routes.json` - Configured routes
- `processes.json` - Running process registry
- `daemon.sock` - Unix domain socket for IPC

Port allocation range: 10240-49151 (avoids common dev ports like 3000, 8080, 8000)

## Development

### Prerequisites

- Go 1.25 or later
- Unix-like OS (Linux, macOS)

### Building

```bash
go build -o bin/faa ./cmd/faa
```

Or using [just](https://github.com/casey/just):

```bash
just build
```

### Running Tests

```bash
go test -v ./...
# Or using just
just test
```

### Code Quality

```bash
# Format code
go fmt ./...

# Run static analysis
go vet ./...
```

### Project Structure

```
faa/
├── cmd/faa/           # CLI entry point
├── internal/
│   ├── daemon/        # Daemon process, IPC, registry
│   ├── devproc/       # Dev process lifecycle management
│   ├── lock/          # File-based locking
│   ├── port/          # Port allocation logic
│   ├── project/       # Project detection (package.json)
│   ├── proxy/         # Embedded Caddy proxy
│   └── setup/         # Setup utilities (stub)
└── README.md
```

## Troubleshooting

### Daemon won't start
- Check if another instance is running: `ps aux | grep "faa daemon"`
- Check daemon logs in `~/.config/faa/`
- Ensure port 2019 (Caddy admin) is available

### HTTPS certificate warnings
- faa uses Caddy's local CA for HTTPS certificates
- Trust the CA certificate: Check Caddy documentation for your OS
- Certificates are automatically managed in `~/.local/share/caddy/`

### Port already in use
- faa avoids common ports, but conflicts can still occur
- Check what's using the port: `lsof -i :PORT`
- Stop conflicting services or restart the daemon

### Project lock error
- Another faa instance is running for this project
- Check for stale `.faa.lock` file in project root
- Remove it if no other instance is actually running

## Contributing

Contributions are welcome! Please ensure:
- All tests pass (`go test ./...`)
- Code is formatted (`go fmt ./...`)
- No vet warnings (`go vet ./...`)

## License

MIT License - see [LICENSE.md](LICENSE.md) for details.

Copyright © 2026 Sahithyan K.