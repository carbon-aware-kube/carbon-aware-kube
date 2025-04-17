package watttime

import (
	"context"
	"time"

	"github.com/carbon-aware-kube/scheduler/internal/zones"
)

// MockWattTimeClient is a mock implementation of WattTimeClientInterface for testing.
// It should be defined in a regular .go file to be accessible from other packages' tests.
type MockWattTimeClient struct {
	// ForecastResponse is the response to return from GetForecast.
	ForecastResponse *ForecastResponse
	// Error is the error to return from GetForecast.
	Error error
}

// GetForecast implements the WattTimeClientInterface for the mock.
func (m *MockWattTimeClient) GetForecast(ctx context.Context, region zones.PowerZone, signalType string) (*ForecastResponse, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	// Return a copy to prevent modification of the mock's state by the caller
	if m.ForecastResponse != nil {
		respCopy := *m.ForecastResponse // Shallow copy is okay here
		return &respCopy, nil
	}
	// Return a default empty response if neither ForecastResponse nor Error is set
	return &ForecastResponse{
		Meta: ForecastMeta{ // Correct struct name
			Region:                 string(region), // Use the string value directly
			SignalType:             signalType,
			DataPointPeriodSeconds: 300, // Default 5 min
			GeneratedAt:            time.Now().UTC(),
		},
		Data: []ForecastDataPoint{},
	}, nil
}
