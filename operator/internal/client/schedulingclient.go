package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SchedulingClientInterface defines the interface for the carbon-aware scheduling client
type SchedulingClientInterface interface {
	GetOptimalSchedule(ctx context.Context, startTime time.Time, maxDelay time.Duration, jobDuration time.Duration, location string) (*ScheduleResponse, error)
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

// ScheduleRequest represents the input for the /api/schedule endpoint
type ScheduleRequest struct {
	Windows    []TimeRange `json:"windows"`
	Duration   string      `json:"duration"`
	Zones      []string    `json:"zones"`
	NumOptions *int        `json:"numOptions,omitempty"`
}

// ScheduleOption represents a potential scheduling option
type ScheduleOption struct {
	Time         time.Time `json:"time"`
	Zone         string    `json:"zone"`
	CO2Intensity float64   `json:"co2Intensity"`
}

// CarbonSavings represents the carbon savings compared to different scenarios
type CarbonSavings struct {
	VsWorstCase  float64 `json:"vsWorstCase"`
	VsNaiveCase  float64 `json:"vsNaiveCase"`
	VsMedianCase float64 `json:"vsMedianCase"`
}

// ScheduleResponse represents the output from the /api/schedule endpoint
type ScheduleResponse struct {
	Ideal         ScheduleOption   `json:"ideal"`
	Options       []ScheduleOption `json:"options"`
	WorstCase     ScheduleOption   `json:"worstCase"`
	NaiveCase     ScheduleOption   `json:"naiveCase"`
	MedianCase    ScheduleOption   `json:"medianCase"`
	CarbonSavings CarbonSavings    `json:"carbonSavings"`
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
func (c *SchedulingClient) GetOptimalSchedule(ctx context.Context, startTime time.Time, maxDelay time.Duration, jobDuration time.Duration, location string) (*ScheduleResponse, error) {
	// Create the scheduling window from start time to start time + max delay
	window := TimeRange{
		Start: startTime,
		End:   startTime.Add(maxDelay),
	}

	// Format the duration as a string (e.g., "1h30m")
	durationStr := jobDuration.String()

	// Create the request payload
	req := ScheduleRequest{
		Windows:  []TimeRange{window},
		Duration: durationStr,
		Zones:    []string{location},
	}

	// Convert the request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP request
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/api/schedule", c.BaseURL),
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
