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

### Option 1: Download Pre-built Binary (Recommended)

Download the latest release for your platform from the [releases page](https://github.com/sahithyandev/faa/releases).

**Supported Platforms:**
- macOS: amd64 (Intel), arm64 (Apple Silicon)
- Linux: amd64, arm64

Extract the binary and add it to your PATH:

```bash
# Example for macOS arm64
curl -L https://github.com/sahithyandev/faa/releases/download/v0.1.0/faa_0.1.0_darwin_arm64.tar.gz | tar xz
sudo mv faa /usr/local/bin/
```

Verify the installation:

```bash
faa version
```

### Option 2: Install from GitHub Container Registry

You can also pull the binary from GHCR as a Docker image:

```bash
# Pull the latest version
docker pull ghcr.io/sahithyandev/faa:latest

# Or a specific version
docker pull ghcr.io/sahithyandev/faa:v0.1.0

# Extract the binary from the container
docker create --name faa-temp ghcr.io/sahithyandev/faa:latest
docker cp faa-temp:/usr/local/bin/faa ./faa
docker rm faa-temp
sudo mv faa /usr/local/bin/
```

### Option 3: Install from Source

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
- Dev server listening on wrong interface: ensure it binds to `0.0.0.0` (all interfaces) or `127.0.0.1`/`localhost` (loopback only), not a specific external IP

### .local domain not resolving

Error: `ping my-project.local` or `curl https://my-project.local` fails with "cannot resolve" or "unknown host".

Solution:

The `.local` domains used by faa are handled by Caddy's reverse proxy on port 443, not via DNS or mDNS. They work by:
1. Your browser connects to `https://my-project.local` (port 443)
2. Caddy intercepts the request and routes it to your dev server
3. The domain doesn't need to exist in DNS or `/etc/hosts`

Important notes:
- **Use in a browser**: Open `https://my-project.local` directly in your web browser
- **Don't use ping**: The `ping` command doesn't understand HTTPS URLs and won't work with .local domains
- **Use curl with -k**: If using curl for testing, use `curl -k https://my-project.local` (the `-k` flag accepts the self-signed certificate, or trust the CA certificate first)
- **Daemon must be running**: Run `faa status` to verify the daemon is active and routes are configured
- **CA must be trusted**: Run `faa setup` to install the CA certificate so browsers trust the HTTPS connection

### OCSP stapling warnings in logs

Warning: `WARN tls stapling OCSP {"identifiers": ["my-project.local"]}` appears in daemon logs.

This warning is harmless and can be safely ignored. It appears because:
- Caddy logs this message during certificate obtainment when checking for OCSP stapling options
- Internal CA certificates (used for `.local` domains) don't have OCSP responders
- OCSP stapling has been disabled in the configuration (`disable_ocsp_stapling: true`)
- The warning is logged before the actual OCSP stapling is attempted/skipped
- Your HTTPS connections work normally and no OCSP requests are actually made

No action needed - this is expected behavior for local development with internal CAs.

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

### Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/) for automatic changelog generation. Format your commit messages as:

- `feat: add new feature` - New features
- `fix: resolve bug` - Bug fixes
- `docs: update documentation` - Documentation changes
- `chore: update dependencies` - Maintenance tasks
- `test: add tests` - Test additions/updates
- `refactor: restructure code` - Code refactoring

Examples:
```
feat: add support for custom domain configuration
fix: resolve port allocation race condition
docs: update installation instructions for macOS
```

## Releasing

This project uses an automated release process with [GoReleaser](https://goreleaser.com/).

### Creating a Release

1. **Update the CHANGELOG.md**:
   - Move changes from `[Unreleased]` to a new version section
   - Set the release date
   - Update version links at the bottom

2. **Create and push a version tag**:
   ```bash
   # Create tag (using semantic versioning)
   git tag -a v0.2.0 -m "Release v0.2.0"
   
   # Push tag to trigger release workflow
   git push origin v0.2.0
   ```

3. **Automated Release Process**:
   - GitHub Actions workflow runs tests
   - GoReleaser builds binaries for:
     - macOS: amd64 (Intel), arm64 (Apple Silicon)
     - Linux: amd64, arm64
   - Creates GitHub Release with auto-generated notes
   - Publishes binaries as release assets
   - Publishes Docker images to GitHub Container Registry (GHCR)
   - Generates SHA256 checksums

4. **Verify the Release**:
   - Check the [releases page](https://github.com/sahithyandev/faa/releases)
   - Download and test binaries
   - Verify Docker images at [GHCR](https://github.com/sahithyandev/faa/pkgs/container/faa)

### Versioning

We follow [Semantic Versioning](https://semver.org/):

- **MAJOR** version: Incompatible API changes
- **MINOR** version: New functionality (backwards compatible)
- **PATCH** version: Bug fixes (backwards compatible)

Examples:
- `v0.1.0` → `v0.2.0`: New features added
- `v0.2.0` → `v0.2.1`: Bug fixes only
- `v0.2.1` → `v1.0.0`: First stable release or breaking changes

## License

MIT License - see [LICENSE.md](LICENSE.md)

Copyright © 2026 Sahithyan K.