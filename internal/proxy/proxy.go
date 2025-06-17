// Package proxy provides basic reverse proxy functionality with load
// balancing and fault tolerance.
//
// This package implements a high-performance HTTP reverse proxy that
// distributes incoming requests across multiple backend targets
// usign round-robin load balancing. It includes automatic retry logic
// to handle backend failures gracefull.
//
// Key features:
//   - Round-robin load balancing across multiple targets
//   - Automatic failover when backends are unavailable
//   - Request logging and error handling
//   - HTTP header forwarding for proper proxy behavior
//
// Example usage:
//
//	cfg := &config.Config{
//		Targets: []config.TargetConfig{
//			{URL: "http://backend1:3000", Enabled: true},
//			{URL: "http://backend2:3000", Enabled: true},
//		},
//	}
//	proxy, err := proxy.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//	http.Handle("/", proxy)
package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"

	"velocity/internal/config"
	"velocity/pkg/logger"
)

// Proxy handles reverse proxying to backend targets with load balancing
//
// The proxy mantains a list of backend target URLs and distributes requests
// among them using round-robin scheduling. It automatically retries failed
// requests on other available targets.
//
// Thread safety: All methods are safe for concurrent use by multiple goroutines
// The atomic counter ensures race-free round-robin distribution.
type Proxy struct {
	// targets contains parsed URLs of all enabled backend services
	targets []*url.URL

	// current is an atomic counter used for round-robin target selection
	current int64

	// stats tracks request statistics per target
	stats []TargetStats

	// logger for structured logging
	logger *logger.Logger
}

// TargetStats holds request statistics for a single target
type TargetStats struct {
	// Requests is the total number of requests sent to this target
	Requests int64

	// Successes is the number of successful requests
	Successes int64

	// Failures is the number of failed requests
	Failures int64
}

// New creates a new proxy instance configured with the given targets.
//
// This constructor:
//  1. Validates and parses all enabled target URLs
//  2. Filters out disabled targets
//  3. Returns an error if no enabled targets are found
//
// URL validation ensures that each target has a valid scheme (http/https)
// and can be parsed correctly. Invalid URLs cause initialization to fail
// rather than causing runtime errors.
//
// Parameters:
//
//	cfg: Configuration containing target definitions
//
// Returns:
//
//	*Proxy: Configured proxy instance ready for use
//	error: URL parsing error or no targets available
//
// Example:
//
//	proxy, err := New(cfg)
//	if err != nil {
//	    return fmt.Errorf("proxy setup failed: %w", err)
//	}
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

	stats := make([]TargetStats, len(targets))
	proxyLogger := logger.New(logger.LoggerConfig{
		Level: cfg.Logging.Level,
		Format: cfg.Logging.Format,
	})

	return &Proxy{
		targets: targets,
		stats:   stats,
		logger: proxyLogger,
	}, nil
}

// ServeHTTP implements http.Handler and proxies to targets using round-robin
// with retry
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if len(p.targets) == 0 {
		http.Error(w, "No targets available", http.StatusBadGateway)
		return
	}

	startIndex := atomic.AddInt64(&p.current, 1) - 1
	for attempt := 0; attempt < len(p.targets); attempt++ {
		targetIndex := (startIndex + int64(attempt)) % int64(len(p.targets))
		target := p.targets[targetIndex]

		p.logger.LogProxy(r.Method, r.URL.Path, target.Host, attempt+1, len(p.targets))

		if p.tryTarget(w, r, target, int(targetIndex), attempt == len(p.targets)-1) {
			return
		}
	}

	p.logger.LogAllTargetsFailed(r.Method, r.URL.Path)
}

// tryTarget attempts to proxy to a specific target, returns true if successful
func (p *Proxy) tryTarget(w http.ResponseWriter, r *http.Request,
	target *url.URL, targetIndex int, isLastAttempt bool) bool {
	atomic.AddInt64(&p.stats[targetIndex].Requests, 1)
	proxy := httputil.NewSingleHostReverseProxy(target)

	var failed bool
	proxy.ErrorHandler = func(ew http.ResponseWriter, er *http.Request,
		err error) {
		p.logger.LogProxyFailure(target.Host, err)
		failed = true

		atomic.AddInt64(&p.stats[targetIndex].Failures, 1)

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
		p.logger.LogProxySuccess(target.Host)
		atomic.AddInt64(&p.stats[targetIndex].Successes, 1)
	}

	return !failed
}

// GetStats returns current statistics for all targets
func (p *Proxy) GetStats() []TargetStats {
	stats := make([]TargetStats, len(p.stats))

	for i := range p.stats {
		stats[i] = TargetStats{
			Requests:  atomic.LoadInt64(&p.stats[i].Requests),
			Successes: atomic.LoadInt64(&p.stats[i].Successes),
			Failures:  atomic.LoadInt64(&p.stats[i].Failures),
		}
	}

	return stats
}
