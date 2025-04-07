package carbon_forecast

import (
	"fmt"
	"log"
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

	// Rollup the forecast to find the carbon intensity for the task duration, at every possible start point
	rollupForecast, err := rollupForecast(&forecast.Data, forecastPeriod, taskDuration)
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

func rollupForecast(forecastData *[]WattTimeForecastData, forecastPeriod time.Duration, taskDuration time.Duration) ([]float64, error) {
	if forecastPeriod > taskDuration {
		return nil, fmt.Errorf("forecast period is longer than the task duration")
	}

	windowSize := float64(taskDuration) / float64(forecastPeriod)
	windowSizeInt := int(math.Ceil(windowSize))
	// TODO: do I want to normalize this into a set time period -- so instead of being based on the forecastPeriod, normalize to a set time period (e.g. 1 hour)
	normalizationFactor := 1 / windowSize
	log.Printf("windowSize: %f, windowSizeInt: %d, normalizationFactor: %f", windowSize, windowSizeInt, normalizationFactor)

	rollupForecast := make([]float64, len(*forecastData)-windowSizeInt+1)
	for i := range len(rollupForecast) {
		sum := 0.0
		for j := range windowSizeInt {
			sum += (*forecastData)[i+j].CarbonIntensity
		}
		log.Printf("sum for i: %d, sum: %f, end: %d", i, sum, i+windowSizeInt)
		rollupForecast[i] = sum * normalizationFactor
	}

	return rollupForecast, nil
}
