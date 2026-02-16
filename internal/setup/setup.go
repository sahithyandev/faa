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
)

// Run executes the setup process for the current platform
func Run() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("setup command is currently only supported on Linux")
	}

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

	// Get Caddy CA certificate path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	caCertPath := filepath.Join(homeDir, ".local", "share", "caddy", "pki", "authorities", "local", "root.crt")

	// Check if the certificate exists
	if _, err := os.Stat(caCertPath); os.IsNotExist(err) {
		fmt.Println("⚠ Caddy root CA certificate not found")
		fmt.Printf("  Expected location: %s\n", caCertPath)
		fmt.Println()
		fmt.Println("The certificate will be created automatically when you start the daemon.")
		fmt.Println("After starting the daemon, run 'faa setup' again to install the certificate.")
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
