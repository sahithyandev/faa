package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	labHostsStartMarker = "# faa lab hosts start"
	labHostsEndMarker   = "# faa lab hosts end"
	labHostsPathEnv     = "FAA_HOSTS_PATH"
)

func hostsFilePath() string {
	if override := os.Getenv(labHostsPathEnv); override != "" {
		return override
	}
	return "/etc/hosts"
}

func (d *Daemon) syncLabHosts(routes map[string]int) {
	hosts := collectLabHosts(routes)
	if err := updateLabHostsFile(hostsFilePath(), hosts); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to update /etc/hosts for .lab domains: %v\n", err)
	}
}

func collectLabHosts(routes map[string]int) []string {
	hosts := make([]string, 0, len(routes))
	for host := range routes {
		if !strings.HasSuffix(host, ".lab") {
			continue
		}
		if !isSafeHostForHostsFile(host) {
			continue
		}
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	return hosts
}

func isSafeHostForHostsFile(host string) bool {
	if host == "" {
		return false
	}
	for _, r := range host {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '.':
		default:
			return false
		}
	}
	return true
}

func updateLabHostsFile(hostsPath string, hosts []string) error {
	original, perm, err := readHostsFile(hostsPath)
	if err != nil {
		return err
	}

	updated := replaceLabHostsBlock(original, renderLabHostsBlock(hosts))
	return writeHostsFileAtomic(hostsPath, updated, perm)
}

func readHostsFile(path string) (string, os.FileMode, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", 0644, nil
		}
		return "", 0, fmt.Errorf("failed to stat hosts file: %w", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read hosts file: %w", err)
	}

	return string(content), info.Mode().Perm(), nil
}

func renderLabHostsBlock(hosts []string) string {
	if len(hosts) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(labHostsStartMarker)
	builder.WriteString("\n")
	for _, host := range hosts {
		builder.WriteString("127.0.0.1 ")
		builder.WriteString(host)
		builder.WriteString("\n")
		builder.WriteString("::1 ")
		builder.WriteString(host)
		builder.WriteString("\n")
	}
	builder.WriteString(labHostsEndMarker)
	builder.WriteString("\n")
	return builder.String()
}

func replaceLabHostsBlock(content, block string) string {
	start := strings.Index(content, labHostsStartMarker)
	end := strings.Index(content, labHostsEndMarker)
	if start != -1 && end != -1 && end > start {
		end += len(labHostsEndMarker)
		suffix := content[end:]
		if strings.HasPrefix(suffix, "\n") {
			suffix = strings.TrimPrefix(suffix, "\n")
		}
		content = content[:start] + suffix
	}

	if block == "" {
		return strings.TrimRight(content, "\n") + "\n"
	}

	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + block
}

func writeHostsFileAtomic(path, content string, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, "faa-hosts-")
	if err != nil {
		return fmt.Errorf("failed to create temp hosts file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if err := tmpFile.Chmod(perm); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to set temp hosts permissions: %w", err)
	}

	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp hosts file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp hosts file: %w", err)
	}

	if err := os.Rename(tmpFile.Name(), path); err != nil {
		return fmt.Errorf("failed to replace hosts file: %w", err)
	}

	return nil
}
