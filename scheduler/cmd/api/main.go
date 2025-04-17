package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/carbon-aware-kube/scheduler/internal/api"
	"github.com/carbon-aware-kube/scheduler/internal/watttime"
	"github.com/carbon-aware-kube/scheduler/internal/zones"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// --- Initialize Dependencies ---

	// WattTime Client Setup (reads credentials from ENV)
	wtClient, err := watttime.NewClient()
	if err != nil {
		log.Fatalf("Failed to create WattTime client: %v", err)
	}

	// Zone Lookup Setup
	zoneLookup := zones.NewSimpleZoneLookup()

	// --- Setup HTTP Server ---
	mux := http.NewServeMux()

	// Default route for unmatched paths
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	})

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Add API handlers
	api.AddHandlers(mux, wtClient, zoneLookup) // Pass WattTime client and zone lookup

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
