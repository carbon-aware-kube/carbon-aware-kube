package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// ScheduleHandler handles requests to /api/schedule/
func ScheduleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScheduleRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// --- STUB IMPLEMENTATION ---
	// In a real implementation, you would:
	// 1. Validate the request (duration format, zones, window times)
	// 2. Fetch carbon intensity data for the specified zones and time window.
	// 3. Calculate the optimal time and zone based on the duration and intensity data.
	// 4. Populate the response with real data.

	// For now, return a hardcoded stub response.
	stubResponse := ScheduleResponse{
		Ideal: ScheduleOption{
			Time:         time.Now().UTC().Add(1 * time.Hour), // Example: 1 hour from now
			Zone:         "us-central1",
			CO2Intensity: 50.5,
		},
		Options: []ScheduleOption{
			{
				Time:         time.Now().UTC().Add(1 * time.Hour),
				Zone:         "us-central1",
				CO2Intensity: 50.5,
			},
			{
				Time:         time.Now().UTC().Add(2 * time.Hour), // Example: 2 hours from now
				Zone:         "europe-west1",
				CO2Intensity: 45.2,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(stubResponse)
	if err != nil {
		log.Printf("Error encoding response body: %v", err)
		// Note: Header might already be written, so we can't send http.Error
		// Log the error and potentially close the connection if necessary.
		return
	}
}
