// Package proxy provides basic reverse proxy functionality
package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"

	"velocity/internal/config"
)

// Proxy handles reverse proxying to backend targets
type Proxy struct {
	targets []*url.URL
	current int64
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

// ServeHTTP implements http.Handler and proxies targets using round-robin
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(p.targets) == 0 {
		http.Error(w, "No targets available", http.StatusBadGateway)
		return
	}

	startIndex := atomic.AddInt64(&p.current, 1) - 1
	for attempt := 0; attempt < len(p.targets); attempt++ {
		targetIndex := (startIndex + int64(attempt)) % int64(len(p.targets))
		target := p.targets[targetIndex]

		fmt.Printf("🔄 Attempt %d: Proxying %s %s -> %s%s (target %d/%d)\n",
			attempt+1, r.Method, r.URL.Path, target.Host, r.URL.Path,
			targetIndex+1, len(p.targets))

		if p.tryTarget(w, r, target, attempt == len(p.targets)-1) {
			return
		}
	}

	fmt.Printf("❌ All targets failed for %s %s\n", r.Method, r.URL.Path)
}

// tryTarget attempts to proxy to a specific target, returns true if successful
func (p *Proxy) tryTarget(w http.ResponseWriter, r *http.Request,
	target *url.URL, isLastAttempt bool) bool {
	proxy := httputil.NewSingleHostReverseProxy(target)

	var failed bool
	proxy.ErrorHandler = func(ew http.ResponseWriter, er *http.Request,
		err error) {
		fmt.Printf("Target %s failed: %v\n", target.Host, err)
		failed = true

		if isLastAttempt {
			ew.Header().Set("Content-Type", "application/json")
			ew.WriteHeader(http.StatusBadGateway)

			fmt.Fprintf(ew, `{"error":"All targets unavailable","last_target":"%s","message":"%s"}`, target.Host, err.Error())
		}
	}

	r.Header.Set("X-Forwarded-Host", r.Host)
	r.Header.Set("X-Forwarded-For", r.RemoteAddr)

	proxy.ServeHTTP(w, r)

	if !failed {
		fmt.Printf("Target %s succeeded\n", target.Host)
	}

	return !failed
}
