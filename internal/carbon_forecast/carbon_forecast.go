package carbon_forecast

import (
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
