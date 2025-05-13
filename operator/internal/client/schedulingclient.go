package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// toISO8601Duration converts a time.Duration to an ISO 8601 duration string (e.g., "PT1H30M")
func toISO8601Duration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	seconds := int64(d.Seconds())
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	result := "PT"
	if hours > 0 {
		result += fmt.Sprintf("%dH", hours)
	}
	if minutes > 0 {
		result += fmt.Sprintf("%dM", minutes)
	}
	if secs > 0 || (hours == 0 && minutes == 0) {
		result += fmt.Sprintf("%dS", secs)
	}
	return result
}

// SchedulingClientInterface defines the interface for the carbon-aware scheduling client
type SchedulingClientInterface interface {
	GetOptimalSchedule(ctx context.Context, startTime time.Time, maxDelay time.Duration, jobDuration time.Duration, location CloudZone) (*ScheduleResponse, error)
}

// SchedulingClient is a client for the carbon-aware scheduling API
type SchedulingClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// TimeRange represents a time window with a start and end time
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// CloudZone represents a cloud provider and region
type CloudZone struct {
	Provider string `json:"provider"`
	Region   string `json:"region"`
}

// ScheduleRequest represents the input for the /v0/schedule endpoint
type ScheduleRequest struct {
	Windows    []TimeRange `json:"windows"`
	Duration   string      `json:"duration"`
	Zones      []CloudZone  `json:"zones"`
	NumOptions *int        `json:"num_options,omitempty"`
}

// ScheduleOption represents a potential scheduling option
type ScheduleOption struct {
	Time         time.Time `json:"time"`
	Zone         CloudZone `json:"zone"`
	CO2Intensity float64   `json:"co2_intensity"`
}

// CarbonSavings represents the carbon savings compared to different scenarios
type CarbonSavings struct {
	VsWorstCase  float64 `json:"vs_worst_case"`
	VsNaiveCase  float64 `json:"vs_naive_case"`
	VsMedianCase float64 `json:"vs_median_case"`
}

// ScheduleResponse represents the output from the /v0/schedule endpoint
type ScheduleResponse struct {
	Ideal         ScheduleOption   `json:"ideal"`
	Options       []ScheduleOption `json:"options"`
	WorstCase     ScheduleOption   `json:"worst_case"`
	NaiveCase     ScheduleOption   `json:"naive_case"`
	MedianCase    ScheduleOption   `json:"median_case"`
	CarbonSavings CarbonSavings    `json:"carbon_savings"`
}

// NewSchedulingClient creates a new client for the carbon-aware scheduling API
func NewSchedulingClient(baseURL string) SchedulingClientInterface {
	return &SchedulingClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetOptimalSchedule calculates the optimal schedule for a job based on carbon intensity forecasts
func (c *SchedulingClient) GetOptimalSchedule(ctx context.Context, startTime time.Time, maxDelay time.Duration, jobDuration time.Duration, location CloudZone) (*ScheduleResponse, error) {
	// Create the scheduling window from start time to start time + max delay
	window := TimeRange{
		Start: startTime,
		End:   startTime.Add(maxDelay),
	}

	// Format the duration as an ISO 8601 string (e.g., "PT1H30M")
	durationStr := toISO8601Duration(jobDuration)

	// Create the request payload
	req := ScheduleRequest{
		Windows:  []TimeRange{window},
		Duration: durationStr,
		Zones:    []CloudZone{{Provider: location.Provider, Region: location.Region}},
	}

	// Convert the request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	fmt.Printf("[DEBUG] ScheduleRequest payload: %s\n", string(reqBody))

	// Create the HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/v0/schedule/", c.BaseURL),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse the response
	var scheduleResp ScheduleResponse
	if err := json.NewDecoder(resp.Body).Decode(&scheduleResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &scheduleResp, nil
}
