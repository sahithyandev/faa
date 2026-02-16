package setup

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sahithyandev/faa/internal/proxy"
)

// Run executes the setup process for the current platform
func Run() error {
	switch runtime.GOOS {
	case "linux":
		return runLinuxSetup()
	case "darwin":
		return runDarwinSetup()
	default:
		return fmt.Errorf("setup command is not supported on %s", runtime.GOOS)
	}
}

// runLinuxSetup executes the setup process for Linux
func runLinuxSetup() error {
	fmt.Println("faa setup - Linux Development Environment")
	fmt.Println()

	// Check privileged port binding
	if err := checkPrivilegedPorts(); err != nil {
		return fmt.Errorf("privileged port check failed: %w", err)
	}

	// Check and install CA trust
	if err := checkCATrust(); err != nil {
		return fmt.Errorf("CA trust setup failed: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Setup complete!")
	return nil
}

// runDarwinSetup executes the setup process for macOS
func runDarwinSetup() error {
	fmt.Println("faa setup - macOS Development Environment")
	fmt.Println()

	// Setup LaunchDaemon
	if err := setupLaunchDaemon(); err != nil {
		return fmt.Errorf("LaunchDaemon setup failed: %w", err)
	}

	// Check and install CA trust
	if err := checkCATrustDarwin(); err != nil {
		return fmt.Errorf("CA trust setup failed: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ Setup complete!")
	return nil
}

// checkPrivilegedPorts checks if the current binary can bind to ports 80 and 443
func checkPrivilegedPorts() error {
	fmt.Println("Checking privileged port binding (80/443)...")

	// Try to bind to port 80
	canBind80 := canBindPort(80)
	canBind443 := canBindPort(443)

	if canBind80 && canBind443 {
		fmt.Println("✓ Can bind to ports 80 and 443")
		return nil
	}

	fmt.Println("✗ Cannot bind to privileged ports")
	fmt.Println()

	// Get current binary path
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get binary path: %w", err)
	}

	// Resolve symlinks
	binaryPath, err = filepath.EvalSymlinks(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to resolve binary path: %w", err)
	}

	fmt.Println("To allow binding to privileged ports without root, run:")
	setcapCmd := fmt.Sprintf("sudo setcap cap_net_bind_service=+ep %s", binaryPath)
	fmt.Printf("  %s\n", setcapCmd)
	fmt.Println()

	// Ask user if they want to run the command
	fmt.Print("Run this command now? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		// If we can't read (e.g., EOF), default to "no"
		fmt.Println()
		fmt.Println("Skipped. You can run the command manually later.")
		return nil
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "y" || response == "yes" {
		fmt.Println("Running setcap command...")
		cmd := exec.Command("sudo", "setcap", "cap_net_bind_service=+ep", binaryPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run setcap: %w", err)
		}

		// Verify the capability was set
		if canBindPort(80) && canBindPort(443) {
			fmt.Println("✓ Capability set successfully")
		} else {
			fmt.Println("⚠ Warning: Capability was set but ports still cannot be bound")
			fmt.Println("  This may be due to ports already in use by other services")
		}
	} else {
		fmt.Println("Skipped. You can run the command manually later.")
	}

	return nil
}

// canBindPort checks if the current process can bind to the specified port
func canBindPort(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// checkCATrust checks and installs the Caddy root CA certificate
func checkCATrust() error {
	fmt.Println()
	fmt.Println("Checking CA certificate trust...")

	// Get CA certificate path from proxy package
	caCertPath, err := proxy.GetCAPath()
	if err != nil {
		return fmt.Errorf("failed to get CA certificate path: %w", err)
	}

	// Check if the certificate exists
	if _, err := os.Stat(caCertPath); os.IsNotExist(err) {
		fmt.Println("⚠ CA certificate not found")
		fmt.Printf("  Expected location: %s\n", caCertPath)
		fmt.Println()
		fmt.Println("The certificate will be created automatically when you start the daemon.")
		fmt.Println("After starting the daemon, run 'faa setup' again to install the certificate.")
		fmt.Println()
		fmt.Println("You can check the certificate path with: faa ca-path")
		return nil
	}

	fmt.Printf("✓ Found Caddy root CA: %s\n", caCertPath)

	// Detect trust store location
	trustStores := []struct {
		path        string
		description string
		installFunc func(string) error
	}{
		{
			path:        "/usr/local/share/ca-certificates",
			description: "Debian/Ubuntu",
			installFunc: installDebianCA,
		},
		{
			path:        "/etc/pki/ca-trust/source/anchors",
			description: "RHEL/CentOS/Fedora",
			installFunc: installRHELCA,
		},
	}

	var selectedStore *struct {
		path        string
		description string
		installFunc func(string) error
	}

	for i := range trustStores {
		if _, err := os.Stat(trustStores[i].path); err == nil {
			selectedStore = &trustStores[i]
			break
		}
	}

	if selectedStore == nil {
		fmt.Println()
		fmt.Println("✗ Could not detect system trust store")
		fmt.Println()
		printManualCAInstructions(caCertPath)
		return nil
	}

	fmt.Printf("✓ Detected trust store: %s (%s)\n", selectedStore.path, selectedStore.description)

	// Check if already installed
	destPath := filepath.Join(selectedStore.path, "caddy-local-ca.crt")
	if _, err := os.Stat(destPath); err == nil {
		// Check if it's the same certificate
		if filesAreEqual(caCertPath, destPath) {
			fmt.Println("✓ CA certificate is already installed and up to date")
			return nil
		}
		fmt.Println("⚠ CA certificate exists but differs from current Caddy CA")
	}

	// Ask user if they want to install
	fmt.Println()
	fmt.Print("Install Caddy root CA to system trust store? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		// If we can't read (e.g., EOF), default to "no"
		fmt.Println()
		fmt.Println("Skipped. To install manually:")
		printManualCAInstructions(caCertPath)
		return nil
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "y" || response == "yes" {
		if err := selectedStore.installFunc(caCertPath); err != nil {
			fmt.Printf("✗ Failed to install certificate: %v\n", err)
			fmt.Println()
			printManualCAInstructions(caCertPath)
			return nil
		}
		fmt.Println("✓ CA certificate installed successfully")
	} else {
		fmt.Println("Skipped. To install manually:")
		printManualCAInstructions(caCertPath)
	}

	return nil
}

// installDebianCA installs the CA certificate on Debian/Ubuntu systems
func installDebianCA(caCertPath string) error {
	destPath := "/usr/local/share/ca-certificates/caddy-local-ca.crt"

	// Copy certificate
	fmt.Printf("Installing certificate to %s...\n", destPath)
	cmd := exec.Command("sudo", "cp", caCertPath, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy certificate: %w", err)
	}

	// Update CA certificates
	fmt.Println("Updating CA certificates...")
	cmd = exec.Command("sudo", "update-ca-certificates")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update CA certificates: %w", err)
	}

	return nil
}

// installRHELCA installs the CA certificate on RHEL/CentOS/Fedora systems
func installRHELCA(caCertPath string) error {
	destPath := "/etc/pki/ca-trust/source/anchors/caddy-local-ca.crt"

	// Copy certificate
	fmt.Printf("Installing certificate to %s...\n", destPath)
	cmd := exec.Command("sudo", "cp", caCertPath, destPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy certificate: %w", err)
	}

	// Update CA trust
	fmt.Println("Updating CA trust...")
	cmd = exec.Command("sudo", "update-ca-trust")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update CA trust: %w", err)
	}

	return nil
}

// printManualCAInstructions prints manual instructions for installing the CA certificate
func printManualCAInstructions(caCertPath string) {
	fmt.Println()
	fmt.Println("Manual CA Certificate Installation:")
	fmt.Println()
	fmt.Println("For Debian/Ubuntu:")
	fmt.Printf("  sudo cp %s /usr/local/share/ca-certificates/caddy-local-ca.crt\n", caCertPath)
	fmt.Println("  sudo update-ca-certificates")
	fmt.Println()
	fmt.Println("For RHEL/CentOS/Fedora:")
	fmt.Printf("  sudo cp %s /etc/pki/ca-trust/source/anchors/caddy-local-ca.crt\n", caCertPath)
	fmt.Println("  sudo update-ca-trust")
	fmt.Println()
	fmt.Println("For Arch Linux:")
	fmt.Printf("  sudo cp %s /etc/ca-certificates/trust-source/anchors/caddy-local-ca.crt\n", caCertPath)
	fmt.Println("  sudo trust extract-compat")
	fmt.Println()
	fmt.Println("After installation, verify with:")
	fmt.Println("  curl -v https://<your-project>.local")
}

// filesAreEqual checks if two files have the same content
func filesAreEqual(path1, path2 string) bool {
	content1, err := os.ReadFile(path1)
	if err != nil {
		return false
	}

	content2, err := os.ReadFile(path2)
	if err != nil {
		return false
	}

	return bytes.Equal(content1, content2)
}

// setupLaunchDaemon sets up the macOS LaunchDaemon for faa
func setupLaunchDaemon() error {
	fmt.Println("Setting up LaunchDaemon for faa daemon...")

	// Get the current binary path
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get binary path: %w", err)
	}

	// Resolve symlinks
	binaryPath, err = filepath.EvalSymlinks(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to resolve binary path: %w", err)
	}

	// Get the socket path for daemon communication
	// On macOS, we'll use /var/run/faa instead of user's home directory
	// to ensure the daemon running as root can create the socket and
	// regular users can still connect to it
	socketDir := "/var/run/faa"
	socketPath := filepath.Join(socketDir, "ctl.sock")

	// Generate the LaunchDaemon plist
	plistContent := generateLaunchDaemonPlist(binaryPath, socketDir)
	plistPath := "/Library/LaunchDaemons/dev.localhost-dev.plist"

	// Check if LaunchDaemon is already installed
	if _, err := os.Stat(plistPath); err == nil {
		fmt.Printf("✓ LaunchDaemon plist already exists at %s\n", plistPath)
		
		// Check if it needs updating
		currentContent, err := os.ReadFile(plistPath)
		if err == nil && string(currentContent) == plistContent {
			fmt.Println("✓ LaunchDaemon configuration is up to date")
			
			// Check if daemon is loaded
			if isLaunchDaemonLoaded() {
				fmt.Println("✓ LaunchDaemon is loaded and running")
				return nil
			}
		} else {
			fmt.Println("⚠ LaunchDaemon configuration differs from expected")
		}
	}

	fmt.Println()
	fmt.Println("This will:")
	fmt.Printf("  1. Create LaunchDaemon plist at %s\n", plistPath)
	fmt.Printf("  2. Configure daemon to run as root with socket at %s\n", socketPath)
	fmt.Println("  3. Load the LaunchDaemon with launchctl")
	fmt.Println()
	fmt.Print("Proceed with LaunchDaemon setup? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println()
		fmt.Println("Skipped. You can set up the LaunchDaemon manually later.")
		printManualLaunchDaemonInstructions(binaryPath, plistPath, plistContent)
		return nil
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Skipped. You can set up the LaunchDaemon manually later.")
		printManualLaunchDaemonInstructions(binaryPath, plistPath, plistContent)
		return nil
	}

	// Write the plist file
	fmt.Printf("Creating LaunchDaemon plist at %s...\n", plistPath)
	tmpPlistPath := filepath.Join(os.TempDir(), "dev.localhost-dev.plist")
	if err := os.WriteFile(tmpPlistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write temporary plist: %w", err)
	}
	defer os.Remove(tmpPlistPath)

	// Copy to system location with sudo
	cmd := exec.Command("sudo", "cp", tmpPlistPath, plistPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to copy plist to system location: %w", err)
	}

	// Set proper ownership and permissions
	cmd = exec.Command("sudo", "chown", "root:wheel", plistPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set plist ownership: %w", err)
	}

	cmd = exec.Command("sudo", "chmod", "644", plistPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set plist permissions: %w", err)
	}

	// Create socket directory with proper permissions
	fmt.Printf("Creating socket directory at %s...\n", socketDir)
	cmd = exec.Command("sudo", "mkdir", "-p", socketDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Set socket directory permissions to allow user access (0755)
	cmd = exec.Command("sudo", "chmod", "755", socketDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set socket directory permissions: %w", err)
	}

	// Load the LaunchDaemon
	fmt.Println("Loading LaunchDaemon with launchctl...")
	cmd = exec.Command("sudo", "launchctl", "load", "-w", plistPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to load LaunchDaemon: %w", err)
	}

	fmt.Println("✓ LaunchDaemon setup complete")
	fmt.Println()
	fmt.Println("The faa daemon will now start automatically on boot.")
	fmt.Printf("Socket location: %s\n", socketPath)
	fmt.Println()
	fmt.Println("To unload the daemon:")
	fmt.Printf("  sudo launchctl unload %s\n", plistPath)

	return nil
}

// generateLaunchDaemonPlist generates the LaunchDaemon plist content
func generateLaunchDaemonPlist(binaryPath, socketDir string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>dev.localhost-dev</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>daemon</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>/var/log/faa-daemon.log</string>
	<key>StandardErrorPath</key>
	<string>/var/log/faa-daemon-error.log</string>
	<key>EnvironmentVariables</key>
	<dict>
		<key>FAA_SOCKET_DIR</key>
		<string>%s</string>
	</dict>
</dict>
</plist>
`, binaryPath, socketDir)
}

// isLaunchDaemonLoaded checks if the LaunchDaemon is currently loaded
func isLaunchDaemonLoaded() bool {
	cmd := exec.Command("sudo", "launchctl", "list", "dev.localhost-dev")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	// If the service is loaded, the output will contain PID or status info
	return len(output) > 0
}

// printManualLaunchDaemonInstructions prints manual setup instructions
func printManualLaunchDaemonInstructions(binaryPath, plistPath, plistContent string) {
	fmt.Println()
	fmt.Println("Manual LaunchDaemon Setup:")
	fmt.Println()
	fmt.Println("1. Create the plist file:")
	fmt.Printf("   Save the following content to %s\n", plistPath)
	fmt.Println()
	fmt.Println(plistContent)
	fmt.Println()
	fmt.Println("2. Set proper permissions:")
	fmt.Printf("   sudo chown root:wheel %s\n", plistPath)
	fmt.Printf("   sudo chmod 644 %s\n", plistPath)
	fmt.Println()
	fmt.Println("3. Create socket directory:")
	fmt.Println("   sudo mkdir -p /var/run/faa")
	fmt.Println("   sudo chmod 755 /var/run/faa")
	fmt.Println()
	fmt.Println("4. Load the LaunchDaemon:")
	fmt.Printf("   sudo launchctl load -w %s\n", plistPath)
}

// checkCATrustDarwin checks and installs the Caddy root CA certificate on macOS
func checkCATrustDarwin() error {
	fmt.Println()
	fmt.Println("Checking CA certificate trust...")

	// Get CA certificate path from proxy package
	caCertPath, err := proxy.GetCAPath()
	if err != nil {
		return fmt.Errorf("failed to get CA certificate path: %w", err)
	}

	// Check if the certificate exists
	if _, err := os.Stat(caCertPath); os.IsNotExist(err) {
		fmt.Println("⚠ CA certificate not found")
		fmt.Printf("  Expected location: %s\n", caCertPath)
		fmt.Println()
		fmt.Println("The certificate will be created automatically when the daemon starts.")
		fmt.Println("After starting the daemon, run 'faa setup' again to install the certificate.")
		fmt.Println()
		fmt.Println("You can check the certificate path with: faa ca-path")
		return nil
	}

	fmt.Printf("✓ Found CA certificate: %s\n", caCertPath)

	// Check if certificate is already trusted
	if isCertificateTrustedDarwin(caCertPath) {
		fmt.Println("✓ CA certificate is already trusted in System keychain")
		return nil
	}

	fmt.Println()
	fmt.Println("This will install the Caddy root CA to the System keychain.")
	fmt.Println("This allows your browser to trust HTTPS certificates for *.local domains.")
	fmt.Println()
	fmt.Print("Install CA certificate to System keychain? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println()
		fmt.Println("Skipped. To install manually:")
		printManualCATrustInstructionsDarwin(caCertPath)
		return nil
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Skipped. To install manually:")
		printManualCATrustInstructionsDarwin(caCertPath)
		return nil
	}

	// Install certificate using security command
	fmt.Println("Installing CA certificate to System keychain...")
	cmd := exec.Command("sudo", "security", "add-trusted-cert",
		"-d", // Add to admin cert store
		"-r", "trustRoot", // Set trust policy to root
		"-k", "/Library/Keychains/System.keychain", // System keychain
		caCertPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Printf("✗ Failed to install certificate: %v\n", err)
		fmt.Println()
		printManualCATrustInstructionsDarwin(caCertPath)
		return nil
	}

	fmt.Println("✓ CA certificate installed successfully")
	return nil
}

// isCertificateTrustedDarwin checks if a certificate is already in the System keychain
func isCertificateTrustedDarwin(certPath string) bool {
	// Read certificate to get subject/issuer info
	cmd := exec.Command("openssl", "x509", "-in", certPath, "-noout", "-subject")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	subject := strings.TrimSpace(string(output))

	// Search for certificate in System keychain
	cmd = exec.Command("security", "find-certificate", "-a", "-c", "Caddy Local Authority",
		"/Library/Keychains/System.keychain")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// If we find the certificate, it's trusted
	return len(output) > 0 && strings.Contains(subject, "Caddy Local Authority")
}

// printManualCATrustInstructionsDarwin prints manual CA trust instructions for macOS
func printManualCATrustInstructionsDarwin(caCertPath string) {
	fmt.Println()
	fmt.Println("Manual CA Certificate Installation for macOS:")
	fmt.Println()
	fmt.Println("Option 1 - Using security command:")
	fmt.Printf("  sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %s\n", caCertPath)
	fmt.Println()
	fmt.Println("Option 2 - Using Keychain Access app:")
	fmt.Println("  1. Open Keychain Access.app")
	fmt.Printf("  2. Drag and drop %s into the System keychain\n", caCertPath)
	fmt.Println("  3. Double-click the certificate")
	fmt.Println("  4. Expand the 'Trust' section")
	fmt.Println("  5. Set 'When using this certificate' to 'Always Trust'")
	fmt.Println()
	fmt.Println("After installation, verify with:")
	fmt.Println("  curl -v https://<your-project>.local")
}
