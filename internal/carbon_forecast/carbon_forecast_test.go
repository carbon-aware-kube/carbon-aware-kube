package carbon_forecast

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("rollupForecast", func() {
	Context("when the forecast period is equal to the task duration", func() {
		var (
			forecastData = []ForecastDataPoint{
				{PointInTime: time.Now(), CarbonIntensity: 1},
				{PointInTime: time.Now().Add(time.Hour), CarbonIntensity: 2},
				{PointInTime: time.Now().Add(2 * time.Hour), CarbonIntensity: 3},
			}
		)
		It("should return the correct rollup forecast (equivalent to the original forecast)", func() {
			rollupForecast, err := rollupForecast(forecastData, time.Hour, time.Hour)
			Expect(err).To(BeNil())
			Expect(rollupForecast).To(Equal([]float64{1, 2, 3}))
		})
	})
	Context("when the forecast period is half the task duration", func() {
		var (
			fiveMinInSeconds = time.Minute.Seconds() * 5
			tenMinInSeconds  = 2 * fiveMinInSeconds
			forecastData     = []ForecastDataPoint{
				{PointInTime: time.Now(), CarbonIntensity: 1},
				{PointInTime: time.Now().Add(time.Duration(fiveMinInSeconds) * time.Second), CarbonIntensity: 2},
				{PointInTime: time.Now().Add(2 * time.Duration(fiveMinInSeconds) * time.Second), CarbonIntensity: 3},
				{PointInTime: time.Now().Add(3 * time.Duration(fiveMinInSeconds) * time.Second), CarbonIntensity: 4},
			}
		)
		It("should average the carbon intensity between adjacent forecast periods", func() {
			rollupForecast, err := rollupForecast(forecastData, time.Duration(fiveMinInSeconds)*time.Second, time.Duration(tenMinInSeconds)*time.Second)
			Expect(err).To(BeNil())
			Expect(rollupForecast).To(Equal([]float64{1.5, 2.5, 3.5}))
		})
	})
})
