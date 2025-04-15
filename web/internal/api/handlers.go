package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/carbon-aware-kube/web/internal/zones"
	"github.com/carbon-aware-kube/web/internal/watttime"
	"github.com/carbon-aware-kube/web/internal/scheduling"
	"github.com/carbon-aware-kube/web/internal/sharedtypes"
)

// Handler holds dependencies for API handlers
type Handler struct {
	wtClient   watttime.WattTimeClientInterface
	zoneLookup zones.ZoneLookup // Use the new interface
}

// NewHandler creates a new Handler with dependencies
func NewHandler(wtClient watttime.WattTimeClientInterface, zoneLookup zones.ZoneLookup) *Handler {
	return &Handler{
		wtClient:   wtClient,
		zoneLookup: zoneLookup,
	}
}

// AddHandlers registers API handlers with the given ServeMux, passing dependencies.
func AddHandlers(mux *http.ServeMux, wtClient watttime.WattTimeClientInterface, zoneLookup zones.ZoneLookup) {
	handler := NewHandler(wtClient, zoneLookup)
	mux.HandleFunc("/api/schedule", handler.ScheduleHandler)
}

// ScheduleHandler handles requests to /api/schedule
func (h *Handler) ScheduleHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req sharedtypes.ScheduleRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Prevent unexpected fields
	err := decoder.Decode(&req)
	if err != nil {
		log.Printf("Error decoding request body: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// --- Validation ---
	if len(req.Windows) == 0 {
		http.Error(w, "Missing required field 'windows'", http.StatusBadRequest)
		return
	}
	if req.Duration == "" {
		http.Error(w, "Missing required field 'duration'", http.StatusBadRequest)
		return
	}
	if len(req.Zones) == 0 {
		http.Error(w, "Missing or empty required field 'zones'", http.StatusBadRequest)
		return
	}

	// --- Validate Zones ---
	validZones := true
	powerZones := make([]zones.PowerZone, 0, len(req.Zones))
	for _, zoneStr := range req.Zones {
		powerZone, ok := h.zoneLookup.GetPowerZone(zoneStr)
		if !ok {
			log.Printf("Invalid zone identifier requested: %s", zoneStr)
			validZones = false
			break
		}
		powerZones = append(powerZones, powerZone)
	}
	if !validZones {
		http.Error(w, "Invalid zone identifier provided", http.StatusBadRequest)
		return
	}

	// Reject multi-zone requests (for now)
	if len(powerZones) > 1 {
		http.Error(w, "Multi-zone scheduling is not yet supported. Please specify only one zone.", http.StatusBadRequest)
		return
	}

	// --- Call Scheduling Logic ---
	// Make number of options configurable via API request (default to 3, allow 2-10)
	numOptions := 3
	if req.NumOptions != nil {
		if *req.NumOptions < 2 || *req.NumOptions > 10 {
			http.Error(w, "numOptions must be between 2 and 10 (inclusive)", http.StatusBadRequest)
			return
		}
		numOptions = *req.NumOptions
	}

	scheduleResponse, err := scheduling.CalculateBestSchedule(
		r.Context(),
		h.wtClient,
		req.Windows,
		req.Duration,
		powerZones,
		req.Zones,
		numOptions,
	)
	if err != nil {
		// CalculateBestSchedule already logged the internal error details
		http.Error(w, fmt.Sprintf("Failed to calculate schedule: %v", err), http.StatusInternalServerError)
		return
	}

	// --- Send Response ---
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(scheduleResponse) // Encode the response from calculator
	if err != nil {
		// Log the error on the server side if encoding fails
		log.Printf("Error encoding response: %v\n", err)
	}
}
