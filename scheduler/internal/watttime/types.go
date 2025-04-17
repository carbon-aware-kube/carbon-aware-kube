package watttime

import "time"

// ForecastDataPoint represents a single data point in the forecast.
type ForecastDataPoint struct {
	PointTime time.Time `json:"point_time"` // Use time.Time, Go's json decoder handles RFC3339Z format
	Value     float64   `json:"value"`      // Assuming value can be float for flexibility
}

// ForecastMeta holds metadata about the forecast response.
type ForecastMeta struct {
	DataPointPeriodSeconds  int               `json:"data_point_period_seconds"`
	GeneratedAt             time.Time         `json:"generated_at"`
	GeneratedAtPeriodSeconds int               `json:"generated_at_period_seconds"`
	// Model field is complex/variable, using interface{} for now or omit if not needed
	// Model                   map[string]interface{} `json:"model"`
	Region                  string            `json:"region"` // Corresponds to zones.PowerZone
	SignalType              string            `json:"signal_type"`
	Units                   string            `json:"units"`
	Warnings                []string          `json:"warnings"`
}

// ForecastResponse is the top-level structure for the /v3/forecast response.
type ForecastResponse struct {
	Data []ForecastDataPoint `json:"data"`
	Meta ForecastMeta        `json:"meta"`
}

// LoginResponse represents the structure for the /login response.
type LoginResponse struct {
	Token string `json:"token"`
}
