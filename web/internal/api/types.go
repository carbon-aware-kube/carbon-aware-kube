package api

import "time"

// TimeRange represents a time window with a start and end time.
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ScheduleRequest represents the input for the /api/schedule endpoint.
type ScheduleRequest struct {
	Window   []TimeRange `json:"window"`
	Duration string      `json:"duration"` // Using string for duration initially for flexibility (e.g., "2h", "30m")
	Zones    []string    `json:"zones"`
}

// ScheduleOption represents a potential scheduling option.
type ScheduleOption struct {
	Time         time.Time `json:"time"`
	Zone         string    `json:"zone"`
	CO2Intensity float64   `json:"co2Intensity"`
}

// ScheduleResponse represents the output for the /api/schedule endpoint.
type ScheduleResponse struct {
	Ideal   ScheduleOption   `json:"ideal"`
	Options []ScheduleOption `json:"options"`
}
