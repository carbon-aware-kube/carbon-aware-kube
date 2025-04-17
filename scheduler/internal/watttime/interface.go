package watttime

import (
	"context"

	"github.com/carbon-aware-kube/scheduler/internal/zones"
)

// WattTimeClientInterface defines the methods required from a WattTime client.
// This allows for mocking in tests.
type WattTimeClientInterface interface {
	GetForecast(ctx context.Context, region zones.PowerZone, signalType string) (*ForecastResponse, error)
	// Add other WattTime methods here if used elsewhere
}

// Compile-time check to ensure *Client implements the interface
var _ WattTimeClientInterface = (*Client)(nil)
