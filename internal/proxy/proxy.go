// Package proxy provides an embedded Caddy v2 proxy server with dynamic route configuration.
//
// The proxy package enables running a fully-featured reverse proxy with:
//   - Automatic HTTP to HTTPS redirection
//   - Internal CA for TLS certificate generation
//   - Dynamic route management without restarts
//   - Thread-safe operations
//
// Example usage:
//
//	p := proxy.New()
//	ctx := context.Background()
//
//	// Set up routes
//	routes := map[string]int{
//	    "app.local": 3000,
//	    "api.local": 3001,
//	}
//	p.ApplyRoutes(routes)
//
//	// Start the proxy
//	if err := p.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer p.Stop()
//
// For testing with unprivileged ports, use NewWithPorts():
//
//	p := proxy.NewWithPorts(8080, 8443)
package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/caddyserver/caddy/v2"

	// Import Caddy modules to register them
	_ "github.com/caddyserver/caddy/v2/modules/caddyhttp"
	_ "github.com/caddyserver/caddy/v2/modules/caddyhttp/reverseproxy"
	_ "github.com/caddyserver/caddy/v2/modules/caddypki"
	_ "github.com/caddyserver/caddy/v2/modules/caddytls"
)

// Proxy manages an embedded Caddy server with dynamic route configuration
type Proxy struct {
	mu        sync.RWMutex
	routes    map[string]int // host -> port mapping
	running   bool
	httpPort  int // HTTP port (default 80)
	httpsPort int // HTTPS port (default 443)
}

// New creates a new Proxy instance with default ports 80 and 443
func New() *Proxy {
	return &Proxy{
		routes:    make(map[string]int),
		running:   false,
		httpPort:  80,
		httpsPort: 443,
	}
}

// NewWithPorts creates a new Proxy instance with custom ports
func NewWithPorts(httpPort, httpsPort int) *Proxy {
	return &Proxy{
		routes:    make(map[string]int),
		running:   false,
		httpPort:  httpPort,
		httpsPort: httpsPort,
	}
}

// Start starts the embedded Caddy server with the given context
func (p *Proxy) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("proxy is already running")
	}

	// Build Caddy configuration as JSON
	configJSON, err := p.buildConfigJSON()
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	// Run Caddy with the configuration
	err = caddy.Run(&caddy.Config{
		Admin: &caddy.AdminConfig{
			Listen: "localhost:2019",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to start Caddy: %w", err)
	}

	// Load the configuration
	err = caddy.Load(configJSON, true)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	p.running = true
	return nil
}

// Stop stops the Caddy server gracefully
func (p *Proxy) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	err := caddy.Stop()
	p.running = false

	if err != nil {
		return fmt.Errorf("failed to stop Caddy: %w", err)
	}

	return nil
}

// ApplyRoutes rebuilds the Caddy configuration with the provided routes
func (p *Proxy) ApplyRoutes(routes map[string]int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		// If not running, just update routes for next start
		p.routes = make(map[string]int)
		for host, port := range routes {
			p.routes[host] = port
		}
		return nil
	}

	// Update routes
	p.routes = make(map[string]int)
	for host, port := range routes {
		p.routes[host] = port
	}

	// Rebuild configuration
	configJSON, err := p.buildConfigJSON()
	if err != nil {
		return fmt.Errorf("failed to build config: %w", err)
	}

	// Reload Caddy with new configuration
	err = caddy.Load(configJSON, true)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	return nil
}

// buildConfigJSON constructs the Caddy configuration JSON from current routes
func (p *Proxy) buildConfigJSON() ([]byte, error) {
	// Build reverse proxy handlers for each route
	httpsRoutes := make([]map[string]interface{}, 0, len(p.routes))
	for host, port := range p.routes {
		route := map[string]interface{}{
			"match": []map[string]interface{}{
				{
					"host": []string{host},
				},
			},
			"handle": []map[string]interface{}{
				{
					"handler": "reverse_proxy",
					"upstreams": []map[string]interface{}{
						{
							"dial": fmt.Sprintf("127.0.0.1:%d", port),
						},
					},
				},
			},
		}
		httpsRoutes = append(httpsRoutes, route)
	}

	// Build the configuration
	config := map[string]interface{}{
		"admin": map[string]interface{}{
			"listen": "localhost:2019",
		},
		"apps": map[string]interface{}{
			"http": map[string]interface{}{
				"http_port":  p.httpPort,
				"https_port": p.httpsPort,
				"servers": map[string]interface{}{
					"http_redirector": map[string]interface{}{
						"listen": []string{fmt.Sprintf(":%d", p.httpPort)},
						"routes": []map[string]interface{}{
							{
								"handle": []map[string]interface{}{
									{
										"handler":     "static_response",
										"status_code": 301,
										"headers": map[string]interface{}{
											"Location": []string{"https://{http.request.host}{http.request.uri}"},
										},
									},
								},
							},
						},
					},
					"https_server": map[string]interface{}{
						"listen": []string{fmt.Sprintf(":%d", p.httpsPort)},
						"routes": httpsRoutes,
						"tls_connection_policies": []map[string]interface{}{
							{},
						},
					},
				},
			},
			"tls": map[string]interface{}{
				"automation": map[string]interface{}{
					"policies": []map[string]interface{}{
						{
							"issuers": []map[string]interface{}{
								{
									"module": "internal",
									"ca":     "local",
								},
							},
						},
					},
				},
			},
		},
	}

	return json.Marshal(config)
}
