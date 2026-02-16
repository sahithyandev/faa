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
- Linux setup command for privileged port binding and CA trust
- macOS setup command with LaunchDaemon and CA trust support

Known Limitations:
- Configuration directory is hardcoded to `~/.config/faa`. On macOS with LaunchDaemon, the socket is at `/var/run/faa` instead of the config directory.

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

### Initial Setup

Before using faa, run the setup command to configure your system:

```bash
faa setup
```

#### Linux Setup

On Linux, this command will:
1. Check if you can bind to privileged ports (80/443)
2. Optionally run `sudo setcap` to allow binding without root
3. Detect your Linux trust store (Debian/Ubuntu or RHEL/CentOS/Fedora)
4. Optionally install the Caddy root CA certificate to your system trust store

#### macOS Setup

On macOS, this command will:
1. Create a LaunchDaemon plist at `/Library/LaunchDaemons/dev.localhost-dev.plist`
2. Configure the daemon to run as root on system startup
3. Set up socket directory at `/var/run/faa` with user-accessible permissions
4. Load the LaunchDaemon with `launchctl`
5. Install the Caddy root CA certificate to the System keychain for HTTPS trust

After setup on macOS, the daemon will start automatically on boot. You can unload it with:
```bash
sudo launchctl unload /Library/LaunchDaemons/dev.localhost-dev.plist
```

Note: The CA certificate is created when you first start the daemon, so you may need to run `faa daemon` first, then run `faa setup` again to install the certificate.

### Starting the Daemon

#### Linux

After setup, start the daemon in the background:

```bash
faa daemon &
# Or run in a separate terminal
faa daemon
```

#### macOS

On macOS, the daemon is automatically started by the LaunchDaemon after running `faa setup`. No need to manually start it. If you need to manually control the daemon:

```bash
# Start the daemon
sudo launchctl load -w /Library/LaunchDaemons/dev.localhost-dev.plist

# Stop the daemon
sudo launchctl unload /Library/LaunchDaemons/dev.localhost-dev.plist

# Check daemon status
sudo launchctl list dev.localhost-dev
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

#### Linux

```bash
faa stop
# Or clear routes when stopping
faa stop --clear-routes
```

#### macOS

On macOS with LaunchDaemon, use launchctl to stop the daemon:

```bash
sudo launchctl unload /Library/LaunchDaemons/dev.localhost-dev.plist
```

Or use the `faa stop` command (but the daemon will restart automatically):

```bash
faa stop
```

## Configuration

### Linux

faa stores its configuration and state in `~/.config/faa/`:
- `routes.json` - Configured routes
- `processes.json` - Running process registry
- `ctl.sock` - Unix domain socket for IPC

### macOS (with LaunchDaemon)

When using LaunchDaemon setup on macOS:
- Configuration: `~/.config/faa/`
- Socket: `/var/run/faa/ctl.sock` (accessible to all users)
- Logs: `/var/log/faa-daemon.log` and `/var/log/faa-daemon-error.log`

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
│   └── setup/         # Setup utilities
└── README.md
```

## Troubleshooting

### Cannot bind to ports 80/443 (Linux)
- Run `faa setup` to configure privileged port binding
- The setup command will offer to run `sudo setcap cap_net_bind_service=+ep` on your binary
- Alternatively, run the daemon with `sudo` (not recommended)

### Cannot bind to ports 80/443 (macOS)
- Run `faa setup` to configure the LaunchDaemon which runs as root
- The daemon will have permission to bind to privileged ports
- No need for `sudo setcap` on macOS

### Daemon won't start

#### Linux
- Check if another instance is running: `ps aux | grep "faa daemon"`
- Check daemon logs in `~/.config/faa/`
- Ensure port 2019 (Caddy admin) is available

#### macOS
- Check LaunchDaemon status: `sudo launchctl list dev.localhost-dev`
- Check daemon logs: `/var/log/faa-daemon.log` and `/var/log/faa-daemon-error.log`
- Verify socket directory exists: `ls -la /var/run/faa/`
- Try reloading: `sudo launchctl unload /Library/LaunchDaemons/dev.localhost-dev.plist && sudo launchctl load -w /Library/LaunchDaemons/dev.localhost-dev.plist`

### HTTPS certificate warnings

#### Linux
- Run `faa setup` to install the Caddy root CA certificate
- The setup command will detect your Linux trust store and install the certificate
- For manual installation, see the instructions printed by `faa setup`
- Certificates are automatically managed in `~/.local/share/caddy/`

#### macOS
- Run `faa setup` to install the Caddy root CA to the System keychain
- The certificate will be trusted for all users
- For manual installation, use: `sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ~/.local/share/caddy/pki/authorities/local/root.crt`

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