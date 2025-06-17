// Package proxy provides basic reverse proxy functionality
package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"velocity/internal/config"
)

// Proxy handles reverse proxying to backend targets
type Proxy struct {
	targets []*url.URL
}

// New creates a new proxy with the given targets
func New(cfg *config.Config) (*Proxy, error) {
	var targets []*url.URL

	for _, target := range cfg.Targets {
		if !target.Enabled {
			continue
		}

		u, err := url.Parse(target.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid target URL %s: %w", target.URL, err)
		}

		targets = append(targets, u)
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no enabled targets configured")
	}

	return &Proxy{targets: targets}, nil
}

// ServeHTTP implements http.Handler and proxies to the first target (for now)
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(p.targets) == 0 {
		http.Error(w, "No targets available", http.StatusBadGateway)
		return
	}

	// Use first target for now (simple round-robin later)
	target := p.targets[0]
	proxy := httputil.NewSingleHostReverseProxy(target)

	r.Header.Set("X-Forwarded-Host", r.Host)
	r.Header.Set("X-Forwarded-For", r.RemoteAddr)

	proxy.ServeHTTP(w, r)
}
