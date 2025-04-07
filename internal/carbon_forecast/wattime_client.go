package carbon_forecast

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	WattTimeApiDomain = "https://api.watttime.org"
	WattTimeApiPath   = "/v3/forecast"
	// TODO: Make this configurable
	WattTimeRegion     = "CAISO_NORTH"
	WattTimeSignalType = "carbon-intensity"
)

type WattTimeForecast struct {
	Data     []WattTimeForecastData   `json:"data"`
	Metadata WattTimeForecastMetadata `json:"meta"`
}

type WattTimeForecastData struct {
	PointInTime     time.Time `json:"point_time"`
	CarbonIntensity float64   `json:"value"`
}

type WattTimeForecastMetadata struct {
	Region        string `json:"region"`
	PeriodSeconds int    `json:"data_point_period_seconds"`
}

type WattTimeClient struct {
	apiKey     string
	apiBaseUrl string
}

func NewWattTimeClient(apiKey string, apiBaseUrl ...string) *WattTimeClient {
	if len(apiBaseUrl) == 0 {
		apiBaseUrl = []string{WattTimeApiDomain}
	}

	return &WattTimeClient{
		apiKey:     apiKey,
		apiBaseUrl: apiBaseUrl[0],
	}
}

// GetCarbonIntensity returns the carbon intensity forecast for a given time range
// The forecast is a list of carbon intensity values for each 5 minute interval in the time range
func (c *WattTimeClient) GetCarbonForecast(start time.Time, end time.Time) (*WattTimeForecast, error) {
	// Start time must be present or in the future, end time must be <= 72 hours from now
	now := time.Now()
	if start.Before(now) {
		return nil, fmt.Errorf("start time must be present or in the future")
	}

	horizonHours := int(end.Sub(now).Hours())
	if horizonHours > 72 {
		return nil, fmt.Errorf("end time must be <= 72 hours from now")
	}

	// Build the request
	url := fmt.Sprintf("%s%s?region=%s&signal_type=%s&horizon_hours=%d", c.apiBaseUrl, WattTimeApiPath, WattTimeRegion, WattTimeSignalType, horizonHours)
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Make the request to the WattTime API
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to WattTime API: %w", err)
	}

	// Parse the response
	var forecast WattTimeForecast
	err = json.NewDecoder(resp.Body).Decode(&forecast)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response from WattTime API: %w", err)
	}

	return &forecast, nil
}
