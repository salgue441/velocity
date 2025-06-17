// Velocity Gateway - High-Performance API Gateway
// Author: Carlos Salguero
// Version: 0.1.0

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"velocity/internal/config"
	"velocity/internal/proxy"
)

func main() {
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	var cfg *config.Config
	if _, err := os.Stat(*configFile); err == nil {
		var loadErr error

		cfg, loadErr = config.LoadFromFile(*configFile)
		if loadErr != nil {
			log.Printf("Failed to load config file: %v, using defaults", loadErr)
			cfg = config.DefaultConfig()
		} else {
			log.Printf("Loaded configuration from %s", *configFile)
		}
	} else {
		cfg = config.DefaultConfig()
		log.Printf("Config file %s not found, using default configuration", *configFile)
	}

	// Create proxy
	var proxyHandler *proxy.Proxy
	var proxyErr error

	proxyHandler, proxyErr = proxy.New(cfg)
	if proxyErr != nil {
		log.Printf("Failed to create proxy: %v", proxyErr)
		log.Fatal("Cannot start gateway without proxy functionality")
	}

	// Basic HTTP server to start with
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"velocity-gateway"}`)
	})

	mux.HandleFunc("/targets", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"targets":[`)

		for i, target := range cfg.Targets {
			if i > 0 {
				fmt.Fprintf(w, `,`)
			}

			fmt.Fprintf(w, `{"url":"%s","enabled":%t}`, target.URL, target.Enabled)
		}

		fmt.Fprintf(w, `]}`)
	})

	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if proxyHandler != nil {
			stats := proxyHandler.GetStats()
			fmt.Fprintf(w, `{"stats":[`)

			for i, stat := range stats {
				if i > 0 {
					fmt.Fprintf(w, `,`)
				}

				fmt.Fprintf(w, `{"target":"%s","requests":%d,"successes":%d,"failures":%d}`,
					cfg.Targets[i].URL, stat.Requests, stat.Successes, stat.Failures)
			}

			fmt.Fprintf(w, `]}`)
		} else {
			fmt.Fprintf(w, `{"error":"Proxy not configured"}`)
		}
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if proxyHandler != nil {
			proxyHandler.ServeHTTP(w, r)
		} else {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"message":"Velocity Gateway - Coming Soon"}`)
		}
	})

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Starting Velocity Gateway on %s", addr)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Server failed to start: ", err)
	}
}
