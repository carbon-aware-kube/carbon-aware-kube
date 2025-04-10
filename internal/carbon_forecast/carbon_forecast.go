package carbon_forecast

import (
	"math"
	"time"
)

type ForecastProvider interface {
	Evaluate(windowStart time.Time, windowEnd time.Time, taskDuration time.Duration) (time.Time, error)
}

type ForecastData struct {
	Data []ForecastDataPoint
}

type ForecastDataPoint struct {
	PointInTime     time.Time
	CarbonIntensity float64
}

// rollupForecast calculates the average carbon intensity for a task of specified duration
// at each possible start time within the forecast data.
// forecastData: The forecast data points
// forecastPeriod: The time period between each forecast data point
// taskDuration: The duration of the task to be scheduled
// Returns a slice of carbon intensity values for each possible start time
func rollupForecast(forecastData []ForecastDataPoint, forecastPeriod time.Duration, taskDuration time.Duration) ([]float64, error) {
	windowSize := float64(taskDuration) / float64(forecastPeriod)
	windowSizeInt := int(math.Ceil(windowSize))
	// TODO: do I want to normalize this into a set time period -- so instead of being based on the forecastPeriod, normalize to a set time period (e.g. 1 hour)
	normalizationFactor := 1 / windowSize

	rollupForecast := make([]float64, len(forecastData)-windowSizeInt+1)
	for i := 0; i < len(rollupForecast); i++ {
		sum := 0.0
		for j := 0; j < windowSizeInt; j++ {
			sum += forecastData[i+j].CarbonIntensity
		}
		rollupForecast[i] = sum * normalizationFactor
	}

	return rollupForecast, nil
}


