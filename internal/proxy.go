// Package internal provides the core reverse proxy implementation for the
// Velocity Gateway MVP.
//
// This package contains the fundamental HTTP proxying logic that forms the
// foundation of the Velocity API Gateway. The implementation focuses on
// performance, reliability, and simplicity while providing a solid base for
// future feature expansion.
//
// Core Components:
//   - Proxy: Main reverse proxy structure managing target configuration
//   - HTTP transport optimization for connection pooling and timeouts
//   - Request/response modification hooks for future middleware integration
//   - Comprehensive error handling with structured error responses
//
// Performance Optimizations:
//   - Connection reuse through HTTP keep-alive
//   - Configurable timeout and connection limits
//   - Efficient request forwarding with minimal allocation overhead
//   - Transport-level optimizations for high-throughput scenarios
//
// Thread Safety:
// All exported functions and types in this package are safe for concurrent use
// by multiple goroutines. The underlying http.Transport handles connection
// pooling and thread safety automatically.
//
// Example Usage:
//
//	proxy, err := internal.NewProxy("http://backend.example.com:8080")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	http.HandleFunc("/", proxy.ServeHTTP)
//	http.ListenAndServe(":8080", nil)
package internal

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"
)

// Proxy represents a high-performance HTTP reverse proxy that forwards requests
// to a single backend target. This structure encapsulates the target configuration
// and the underlying httputil.ReverseProxy instance with optimized transport settings.
//
// The Proxy implements intelligent request forwarding with the following features:
//   - Automatic header modification for proxy identification
//   - Connection pooling for optimal resource utilization
//   - Configurable timeouts for reliability
//   - Structured error handling for upstream failures
//
// Fields:
//
//	target: Parsed URL of the backend service
//	proxy: Configured httputil.ReverseProxy instance with custom behavior
type Proxy struct {
	// target holds the parsed backend URL for the proxy destination
	target *url.URL

	// proxy is the configured httputil.ReverseProxy with custom transport
	// and response modification behavior
	proxy *httputil.ReverseProxy
}

// NewProxy creates and initializes a new Proxy instance configured to forward
// requests to the specified target URL.
//
// This constructor performs comprehensive validation of the target URL and
// configures the underlying reverse proxy with production-ready settings
// including connection pooling, timeouts, and custom response handling.
//
// Transport Configuration:
//   - ResponseHeaderTimeout: 30 seconds (prevents hanging on slow responses)
//   - IdleConnTimeout: 90 seconds (connection reuse optimization)
//   - MaxIdleConns: 100 (global connection pool limit)
//   - MaxIdleConnsPerHost: 10 (per-host connection limit)
//
// Parameters:
//
//	targetURL: The backend service URL (must be valid HTTP/HTTPS URL)
//
// Returns:
//
//	*Proxy: Configured proxy instance ready for request handling
//	error: Configuration error if target URL is invalid
//
// Example:
//
//	proxy, err := NewProxy("https://api.backend.com:8080")
//	if err != nil {
//	    return fmt.Errorf("proxy setup failed: %w", err)
//	}
func NewProxy(targetURL string) (*Proxy, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid target URL '%s': %w", targetURL, err)
	}

	if target.Scheme != "http" && target.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme '%s': must be http or https", target.Scheme)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	proxy.ModifyResponse = modifyResponse
	proxy.ErrorHandler = errorHandler
	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: 30 * time.Second, // Prevent hanging on slow responses
		IdleConnTimeout:       90 * time.Second, // Connection reuse window
		MaxIdleConns:          100,              // Global connection pool size
		MaxIdleConnsPerHost:   10,               // Per-host connection limit
		MaxConnsPerHost:       50,               // Maximum concurrent connections per host
		DisableCompression:    false,            // Enable gzip compression
		ForceAttemptHTTP2:     true,             // Prefer HTTP/2 when available
	}

	return &Proxy{
		target: target,
		proxy:  proxy,
	}, nil
}

// ServeHTTP implements the http.Handler interface to process incoming HTTP requests
// and forward them to the configured backend target.
//
// This method performs the following operations:
//  1. Logs the incoming request for debugging and monitoring
//  2. Adds proxy-specific headers for backend identification
//  3. Forwards the request using the configured reverse proxy
//  4. Handles any upstream errors with structured error responses
//
// Request Headers Added:
//   - X-Forwarded-Host: Original request host for backend identification
//   - X-Forwarded-Proto: Protocol scheme (http/https) for secure handling
//
// Parameters:
//
//	w: HTTP response writer for sending responses to the client
//	r: HTTP request to be proxied to the backend service
//
// The method is thread-safe and can handle multiple concurrent requests.
// Request logging includes method, path, and target information for debugging.
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("🔄 %s %s -> %s%s\n", r.Method, r.URL.Path, p.target.Host, r.URL.Path)

	r.Header.Set("X-Forwarded-Host", r.Host)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	r.Header.Set("X-Forwarded-Proto", scheme)

	if clientIP := r.Header.Get("X-Real-IP"); clientIP == "" {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded == "" {
			r.Header.Set("X-Forwarded-For", r.RemoteAddr)
		}
	}

	p.proxy.ServeHTTP(w, r)
}

// modifyResponse is a response modification hook that allows customization
// of responses returned from the backend service before sending them to clients.
//
// Current modifications:
//   - Adds X-Proxied-By header to identify the gateway
//   - Preserves all original response headers and body content
//
// This function provides an extension point for future features such as:
//   - Response header manipulation
//   - Content transformation
//   - Response caching directives
//   - Security header injection
//
// Parameters:
//
//	r: HTTP response from the backend service
//
// Returns:
//
//	error: Always nil in current implementation; reserved for future use
func modifyResponse(r *http.Response) error {
	r.Header.Set("X-Proxied-By", "Velocity-Gateway/0.1.0")
	r.Header.Set("X-Gateway-Time", time.Now().UTC().Format(time.RFC3339))

	return nil
}

// errorHandler processes errors that occur during request proxying and generates
// appropriate HTTP error responses for clients.
//
// This function handles various types of upstream failures including:
//   - Connection timeouts and network errors
//   - DNS resolution failures
//   - Backend service unavailability
//   - HTTP protocol errors
//
// Error Response Format:
//
//	Content-Type: application/json
//	Status Code: 502 Bad Gateway
//	Body: Structured JSON with error details
//
// Response Schema:
//
//	{
//	  "error": "Bad Gateway",
//	  "message": "Human-readable error description",
//	  "code": "UPSTREAM_UNAVAILABLE",
//	  "timestamp": "2024-01-01T00:00:00Z"
//	}
//
// Parameters:
//
//	w: HTTP response writer for sending error responses
//	r: Original HTTP request that failed
//	err: Error that occurred during proxying
//
// The function logs all errors for debugging and monitoring while providing
// clean, structured error responses to clients without exposing internal details.
func errorHandler(w http.ResponseWriter, r *http.Request, err error) {
	fmt.Printf("❌ Proxy error for %s %s: %v\n", r.Method, r.URL.Path, err)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Gateway-Error", "true")

	w.WriteHeader(http.StatusBadGateway)
	errorResponse := fmt.Sprintf(`{
		"error": "Bad Gateway",
		"message": "The upstream server is not available",
		"code": "UPSTREAM_UNAVAILABLE",
		"timestamp": "%s",
		"request_id": "%s"
	}`, time.Now().UTC().Format(time.RFC3339), generateRequestID(r))

	w.Write([]byte(errorResponse))
}

// generateRequestID creates a unique identifier for request tracking and
// debugging. This is a simple implementation for the MVP; future versions will
// use more sophisticated correlation ID generation.
//
// Parameters:
//
//	r: HTTP request for context
//
// Returns:
//
//	string: Unique request identifier
func generateRequestID(r *http.Request) string {
	return fmt.Sprintf("%d-%s", time.Now().UnixNano(), r.Method)
}
