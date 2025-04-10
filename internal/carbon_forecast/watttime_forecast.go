package carbon_forecast

import (
	"math"
	"time"
)

type WattimeForecastProvider struct {
	client *WattTimeClient
}

func NewWattimeForecastProvider(client *WattTimeClient) *WattimeForecastProvider {
	return &WattimeForecastProvider{
		client: client,
	}
}

func (fp *WattimeForecastProvider) Evaluate(windowStart time.Time, windowEnd time.Time, taskDuration time.Duration) (time.Time, error) {
	// Get the forecast for the window
	forecast, err := fp.client.GetCarbonForecast(windowStart, windowEnd)
	if err != nil {
		return time.Time{}, err
	}

	// Fetch the forecast period
	forecastPeriod := time.Duration(forecast.Metadata.PeriodSeconds) * time.Second

	// Convert WattTimeForecastData to ForecastDataPoint
	forecastData := ToForecastDataPoints(forecast.Data)

	// Rollup the forecast to find the carbon intensity for the task duration, at every possible start point
	rollupForecast, err := rollupForecast(forecastData, forecastPeriod, taskDuration)
	if err != nil {
		return time.Time{}, err
	}

	// Find the index of the minimum carbon intensity
	minCarbonIntensity := math.Inf(1)
	minIndex := 0
	for i, intensity := range rollupForecast {
		if intensity < minCarbonIntensity {
			minCarbonIntensity = intensity
			minIndex = i
		}
	}

	// Return the time of the minimum carbon intensity
	return windowStart.Add(time.Duration(minIndex) * forecastPeriod), nil
}

// Sanitizes the forecast by rounding the time to the nearest second
func SanitizeForecast(forecast *WattTimeForecast) *WattTimeForecast {
	for i := range forecast.Data {
		forecast.Data[i].PointInTime = forecast.Data[i].PointInTime.Round(0)
	}
	return forecast
}


