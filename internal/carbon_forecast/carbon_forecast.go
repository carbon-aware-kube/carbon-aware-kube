package carbon_forecast

import (
	"time"
)

func EvaluateCarbonForecast(windowStart time.Time, windowEnd time.Time, windowDuration time.Duration) (time.Time, error) {
	// Find the midpoint of the start and end times
	midpointDelta := windowEnd.Sub(windowStart) + (windowDuration / 2)

	// Return the midpoint
	return windowStart.Add(midpointDelta), nil
}
