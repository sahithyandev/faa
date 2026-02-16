# faa

A development server manager that provides stable HTTPS URLs for local Node.js projects.

## What faa does

faa runs a background daemon that:
- Assigns each project a stable port based on its name (consistent across restarts)
- Provides HTTPS access via .local domains (e.g., https://my-app.local)
- Manages an embedded Caddy reverse proxy with automatic certificate generation
- Tracks running development servers across multiple projects

When you run `faa npm start` in a project directory, faa:
1. Detects your project from package.json
2. Assigns a stable port (e.g., 12345)
3. Starts your dev server with PORT=12345
4. Configures HTTPS routing: https://my-app.local -> localhost:12345
5. Displays the HTTPS URL you can access in your browser

## Why .local domains

Using .local domains (e.g., my-app.local instead of localhost:3000) provides:
- Consistent URLs that don't change when port numbers change
- Proper HTTPS with trusted certificates for testing secure features
- Cleaner URLs that are easier to share and remember
- Ability to run multiple projects simultaneously without port conflicts

## Requirements

- Go 1.25 or later (for building from source)
- Linux or macOS
- Node.js projects with package.json

## Installation

Install from source using Go:

```bash
go install github.com/sahithyandev/faa/cmd/faa@latest
```

Or build locally:

```bash
git clone https://github.com/sahithyandev/faa.git
cd faa
go build -o bin/faa ./cmd/faa
# Optionally add bin/faa to your PATH
```

## Setup

Run the setup command after installation. This configures your system to allow faa to bind to ports 80 and 443 (required for HTTPS) and installs the CA certificate.

```bash
faa setup
```

### Linux Setup

The setup command will:

1. Check if you can bind to privileged ports 80 and 443
2. Prompt to run `sudo setcap cap_net_bind_service=+ep` on the faa binary (allows binding without sudo)
3. Detect your Linux distribution's trust store
4. Prompt to install the Caddy CA certificate to your system trust store

After setup, start the daemon manually:

```bash
# Start daemon in background
faa daemon &

# Or run in a separate terminal window
faa daemon
```

To stop the daemon:

```bash
faa stop
```

### macOS Setup

The setup command will:

1. Create a LaunchDaemon configuration file
2. Install the daemon to run automatically at startup as root (required for port 443 binding)
3. Configure the socket at `/var/run/faa/ctl.sock` (accessible to all users)
4. Start the daemon automatically
5. Prompt to install the Caddy CA certificate to the System keychain

After setup, the daemon runs automatically. No need to start it manually.

To manually control the daemon:

```bash
# Start daemon
sudo launchctl load -w /Library/LaunchDaemons/dev.localhost-dev.plist

# Stop daemon
sudo launchctl unload /Library/LaunchDaemons/dev.localhost-dev.plist

# Check daemon status
sudo launchctl list dev.localhost-dev

# View daemon logs
tail -f /var/log/faa-daemon.log
tail -f /var/log/faa-daemon-error.log
```

## Usage

### Running Your Dev Server

Navigate to your Node.js project directory (must contain package.json) and run:

```bash
faa run -- npm start
```

This command:
- Detects your project name from package.json
- Assigns a stable port (e.g., 12345)
- Starts your dev server with PORT environment variable set
- Configures HTTPS routing
- Displays the URL: https://my-project.local

The PORT environment variable is automatically injected, so your dev server should use it:

```javascript
// Express example
const port = process.env.PORT || 3000;
app.listen(port, () => console.log(`Server on port ${port}`));
```

### Using with npm Scripts

You can use faa with any npm script. Common examples:

```bash
# Run npm start
faa run -- npm start

# Run npm run dev
faa run -- npm run dev

# Run yarn dev
faa run -- yarn dev

# Run pnpm dev
faa run -- pnpm dev

# Shorthand: omit "run --" for simple commands
faa npm start
faa npm run dev
```

The `--` separator is optional when the command doesn't conflict with faa's own commands.

### Available Commands

Check running projects:

```bash
faa status
```

Output shows daemon status, configured routes, and running processes:

```
Daemon Status: Running

Routes:
  my-project.local -> localhost:12345
  another-app.local -> localhost:23456

Running Processes:
  PID 12345: /home/user/projects/my-project (https://my-project.local, port 12345)
  PID 23456: /home/user/projects/another-app (https://another-app.local, port 23456)
```

List all routes:

```bash
faa routes
```

Stop the daemon:

```bash
# Stop daemon (keeps routes)
faa stop

# Stop daemon and clear all routes
faa stop --clear-routes
```

Note: On macOS with LaunchDaemon, the daemon will restart automatically after `faa stop`. Use `launchctl unload` to permanently stop it.

Get CA certificate path:

```bash
faa ca-path
```

This shows where the CA certificate is stored, useful for manual trust configuration.

## Troubleshooting

### Cannot bind to port 443 (Linux)

Error: "bind: permission denied" when starting daemon.

Solution:

```bash
# Run setup to configure port binding
faa setup

# Or manually grant capability
sudo setcap cap_net_bind_service=+ep $(which faa)
```

If this doesn't work:
- Check if another service is using port 443: `sudo lsof -i :443`
- Stop conflicting services: `sudo systemctl stop apache2` or `sudo systemctl stop nginx`
- As a last resort, run daemon with sudo: `sudo faa daemon` (not recommended)

### Cannot bind to port 443 (macOS)

Error: "bind: permission denied" when starting daemon.

Solution:

```bash
# Run setup to install LaunchDaemon (runs as root)
faa setup
```

The LaunchDaemon automatically has permission to bind to privileged ports.

If the daemon won't start:
- Check if another service is using port 443: `sudo lsof -i :443`
- Unload and reload the daemon:
  ```bash
  sudo launchctl unload /Library/LaunchDaemons/dev.localhost-dev.plist
  sudo launchctl load -w /Library/LaunchDaemons/dev.localhost-dev.plist
  ```

### Certificate trust warnings in browser

Error: "Your connection is not private" or "NET::ERR_CERT_AUTHORITY_INVALID" when accessing https://my-project.local.

Solution:

```bash
# Run setup to install CA certificate
faa setup
```

For Linux, manually install the certificate:

```bash
# Get certificate path
faa ca-path

# Debian/Ubuntu
sudo cp ~/.config/faa/root.crt /usr/local/share/ca-certificates/caddy-local-ca.crt
sudo update-ca-certificates

# RHEL/CentOS/Fedora
sudo cp ~/.config/faa/root.crt /etc/pki/ca-trust/source/anchors/caddy-local-ca.crt
sudo update-ca-trust

# Arch Linux
sudo cp ~/.config/faa/root.crt /etc/ca-certificates/trust-source/anchors/caddy-local-ca.crt
sudo trust extract-compat
```

For macOS, manually install to System keychain:

```bash
# Get certificate path
faa ca-path

# Install to System keychain
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain ~/.config/faa/root.crt
```

After installation, restart your browser to apply the changes.

### Daemon not running

Error: "Daemon is not running" when running faa commands.

Solution:

Linux:
```bash
# Start daemon
faa daemon &

# Check if daemon is running
faa status

# Check for errors in terminal output where daemon was started
```

macOS:
```bash
# Check daemon status
sudo launchctl list dev.localhost-dev

# If not loaded, load it
sudo launchctl load -w /Library/LaunchDaemons/dev.localhost-dev.plist

# Check logs for errors
tail -f /var/log/faa-daemon-error.log
```

Common causes:
- Port 443 already in use (see "Cannot bind to port 443" section)
- Port 2019 (Caddy admin) already in use: `lsof -i :2019`
- Socket file permission issues: check `~/.config/faa/ctl.sock` (Linux) or `/var/run/faa/ctl.sock` (macOS)

### Port conflicts

Error: Port already in use or projects get unexpected ports.

Solution:

```bash
# Check what's using a port
lsof -i :12345

# Stop conflicting service or kill process
kill <PID>

# Restart daemon to reassign ports
faa stop
faa daemon &  # Linux
# On macOS, daemon restarts automatically
```

faa assigns ports in the range 10240-49151, avoiding common dev ports (3000, 8080, etc.). If you experience conflicts:
- Stop other development servers running outside faa
- Clear routes and restart daemon: `faa stop --clear-routes`

### Project lock error

Error: "Failed to acquire project lock" when running faa.

This means another faa instance is already running for this project.

Solution:

```bash
# Check for running processes
faa status

# If process is shown, it's already running
# If process is not shown but error persists, stale lock file exists
ls .faa.lock

# Remove stale lock file (only if no faa process is actually running)
rm .faa.lock

# Try running again
faa npm start
```

### Connection refused when accessing URL

You can access the daemon and routes are configured, but https://my-project.local shows "connection refused".

Solution:

Check if your dev server is actually running:

```bash
# Check faa status
faa status

# Check if process is alive
ps aux | grep <PID>

# Test direct connection
curl http://localhost:<port>
```

Common causes:
- Dev server crashed: check terminal output where you ran `faa run`
- Dev server not listening on PORT: make sure your server uses `process.env.PORT`
- Dev server listening on wrong interface: ensure it binds to `0.0.0.0` or `localhost`, not `127.0.0.1`

## Safety and Security Notes

- The faa daemon requires permission to bind to ports 80 and 443 (privileged ports on Unix systems)
- On Linux, this is granted via `setcap` capability, which is safer than running as root
- On macOS, the LaunchDaemon runs as root but only binds to network ports; it does not execute your dev server code as root
- Your development servers run with your user permissions, not as root
- The Caddy CA certificate is stored locally and only trusted on your machine
- Do not share your CA certificate private key with others
- Only use faa for local development; it is not designed for production use
- faa creates a .faa.lock file in your project directory to prevent multiple instances; this file is automatically cleaned up
- Configuration and state files are stored in ~/.config/faa/ (Linux) or /var/run/faa/ (macOS socket only)

## Configuration

Linux:
- Configuration directory: ~/.config/faa/
- Socket: ~/.config/faa/ctl.sock
- Routes: ~/.config/faa/routes.json
- Processes: ~/.config/faa/processes.json
- CA certificate: ~/.config/faa/root.crt

macOS with LaunchDaemon:
- Configuration directory: ~/.config/faa/
- Socket: /var/run/faa/ctl.sock (accessible to all users)
- Logs: /var/log/faa-daemon.log and /var/log/faa-daemon-error.log
- LaunchDaemon plist: /Library/LaunchDaemons/dev.localhost-dev.plist

Port allocation: 10240-49151 (avoids common development ports like 3000, 8080, 8000)

## Contributing

Contributions are welcome. Before submitting:

```bash
# Run tests
go test -v ./...

# Format code
go fmt ./...

# Check for issues
go vet ./...
```

## License

MIT License - see [LICENSE.md](LICENSE.md)

Copyright © 2026 Sahithyan K.