package client

import (
	"context"
	"strings"
	"time"
)

// MockSchedulingClient is a mock implementation of the scheduling client for testing
type MockSchedulingClient struct {
	// MockGetOptimalSchedule is a function that will be called by GetOptimalSchedule
	MockGetOptimalSchedule func(ctx context.Context, startTime time.Time, maxDelay time.Duration, jobDuration time.Duration, location string) (*ScheduleResponse, error)
}

// Ensure MockSchedulingClient implements SchedulingClientInterface
var _ SchedulingClientInterface = (*MockSchedulingClient)(nil)

// GetOptimalSchedule calls the mock function
func (m *MockSchedulingClient) GetOptimalSchedule(ctx context.Context, startTime time.Time, maxDelay time.Duration, jobDuration time.Duration, location string) (*ScheduleResponse, error) {
	if m.MockGetOptimalSchedule != nil {
		return m.MockGetOptimalSchedule(ctx, startTime, maxDelay, jobDuration, location)
	}
	
	// Default implementation if no mock function is provided
	now := time.Now()
	
	// Parse the location string (format: "provider:region")
	var zone CloudZone
	parts := strings.Split(location, ":")
	if len(parts) == 2 {
		zone = CloudZone{
			Provider: parts[0],
			Region:   parts[1],
		}
	} else {
		// Default to aws:us-east-1 if invalid format
		zone = CloudZone{
			Provider: "aws",
			Region:   "us-east-1",
		}
	}
	
	return &ScheduleResponse{
		Ideal: ScheduleOption{
			Time:         now.Add(1 * time.Hour),
			Zone:         zone,
			CO2Intensity: 400.0,
		},
		WorstCase: ScheduleOption{
			Time:         now.Add(2 * time.Hour),
			Zone:         zone,
			CO2Intensity: 700.0,
		},
		NaiveCase: ScheduleOption{
			Time:         now,
			Zone:         zone,
			CO2Intensity: 600.0,
		},
		MedianCase: ScheduleOption{
			Time:         now.Add(30 * time.Minute),
			Zone:         zone,
			CO2Intensity: 550.0,
		},
		CarbonSavings: CarbonSavings{
			VsWorstCase:  42.85, // (700-400)/700 * 100
			VsNaiveCase:  33.33, // (600-400)/600 * 100
			VsMedianCase: 27.27, // (550-400)/550 * 100
		},
	}, nil
}
