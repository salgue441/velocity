// Package main implements the Velocity API Gateway MVP.
// A high-performance HTTP reverse proxy that serves as the
// foundation for a full-featured API gateway.
//
// The Velocity Gateway MVP provides basic HTTP proxying capabilities with a
// focus on simplicity and performance.
//
// Usage:
//   velocity -port=8080 -target=http://backend.example.com:3000
//
// The server exposes two main endpoints:
//   - /health: Returns service health status in JSON format
//   - /*: Proxies all other requests to the configured backend target
//
// Command Line Arguments:
//   -port: HTTP server port (default: 8080)
//   -target: Backend target URL (default: http://localhost:3000)
//
// Author: Carlos Salguero
// Version: 0.1.0-MVP
// License: MIT

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"velocity/internal"
)

// main is the entry point for the Velocity API Gateway MVP Server.
// It initializes the command-line arguments, creates the proxy instance,
// configures HTTP handlers, and starts the server.
//
// The function performs the following operations:
//  1. Parse command-line flags for port and target configuration.
//  2. Initialize the reverse proxy with the specified target.
//  3. Register HTTP handlers for health checks and proxying requests.
//  4. Start the HTTP server with grafecul logging and error handling.
//
// Exit Codes:
//
//	0: Normal termination
//	1: Configuration error or server startup failure
func main() {
	port := flag.String("port", "8080", "HTTP Server Port Number")
	target := flag.String("target", "http://localhost:3000", "Backend Target URL")
	flag.Parse()

	proxy, err := internal.NewProxy(*target)
	if err != nil {
		log.Fatalf("Failed to create proxy: %v", err)
	}

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", proxy.ServeHTTP)

	// Start the HTTP server
	addr := ":" + *port
	fmt.Printf("Velocity Gateway starting on port %s\n", *port)
	fmt.Printf("Proxying to: %s\n", *target)
	fmt.Printf("Health check: http://localhost%s/health\n", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// healthHandler implements the health check endpoint for service
// monitoring. This endpoint provides a simple way to verify that the
// gateway is running and responding to requests. e
//
// Response Format:
//
//	Content-Type: application/json
//	Status Code: 200 OK
//	Body: {"status": "healthy", "service": "velocity-gateway"}
//
// Parameters:
//
//	w: HTTP response writer for sending the health status
//	r: HTTP request (method and path are ignored)
//
// The health check always returns a successful status in the MVP version.
// Future iterations will include upstream health validation and more
// comprehensive service status reporting.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte(`{"status":"healthy","service":"velocity-gateway","version":"0.1.0-mvp"}`)); err != nil {
		log.Printf("Failed to write health check response: %v", err)
	}
}
