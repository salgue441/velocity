// Velocity Gateway - High-Performance API Gateway
// Author: Carlos Salguero
// Version: 0.1.0

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"velocity/internal/config"
)

func main() {
	var cfg *config.Config

	if _, err := os.Stat("config.yaml"); err == nil {
		var loadErr error

		cfg, loadErr = config.LoadFromFile("config.yaml")
		if loadErr != nil {
			log.Printf("Failed to load config file: %v, using defaults", loadErr)
			cfg = config.DefaultConfig()
		} else {
			log.Printf("Loaded configuration from config.yaml")
		}
	} else {
		cfg = config.DefaultConfig()
		log.Println("Using default configuration")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"velocity-gateway"}`)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message":"Velocity Gateway - Coming Soon"}`)
	})

	addr := ":8080"
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
