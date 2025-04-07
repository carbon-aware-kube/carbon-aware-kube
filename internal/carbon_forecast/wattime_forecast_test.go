package carbon_forecast

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func buildMockWattTimeForecast(
	output []WattTimeForecastData,
	periodSeconds int,
) WattTimeForecast {
	return WattTimeForecast{
		Data:     output,
		Metadata: WattTimeForecastMetadata{Region: "CAISO_NORTH", PeriodSeconds: periodSeconds},
	}
}

var _ = Describe("WattimeForecastProvider", func() {
	Describe("rollupForecast", func() {
		Context("when the forecast period is equal to the task duration", func() {
			var (
				mockWattimeForecast = buildMockWattTimeForecast(
					[]WattTimeForecastData{
						{PointInTime: time.Now(), CarbonIntensity: 1},
						{PointInTime: time.Now().Add(time.Hour), CarbonIntensity: 2},
						{PointInTime: time.Now().Add(2 * time.Hour), CarbonIntensity: 3},
					},
					int(time.Hour.Seconds()),
				)
			)
			It("should return the correct rollup forecast (equivalent to the original forecast)", func() {
				rollupForecast, err := rollupForecast(&mockWattimeForecast.Data, time.Hour, time.Hour)
				Expect(err).To(BeNil())
				Expect(rollupForecast).To(Equal([]float64{1, 2, 3}))
			})
		})
		Context("when the forecast period is half the task duration", func() {
			var (
				fiveMinInSeconds    = time.Minute.Seconds() * 5
				tenMinInSeconds     = 2 * fiveMinInSeconds
				mockWattimeForecast = buildMockWattTimeForecast(
					[]WattTimeForecastData{
						{PointInTime: time.Now(), CarbonIntensity: 1},
						{PointInTime: time.Now().Add(time.Duration(fiveMinInSeconds) * time.Second), CarbonIntensity: 2},
						{PointInTime: time.Now().Add(2 * time.Duration(fiveMinInSeconds) * time.Second), CarbonIntensity: 3},
						{PointInTime: time.Now().Add(3 * time.Duration(fiveMinInSeconds) * time.Second), CarbonIntensity: 4},
					},
					int(fiveMinInSeconds),
				)
			)
			It("should average the carbon intensity between adjacent forecast periods", func() {
				rollupForecast, err := rollupForecast(&mockWattimeForecast.Data, time.Duration(fiveMinInSeconds)*time.Second, time.Duration(tenMinInSeconds)*time.Second)
				Expect(err).To(BeNil())
				Expect(rollupForecast).To(Equal([]float64{1.5, 2.5, 3.5}))
			})
		})
		Context("when the forecast period is longer than the task duration", func() {
			var (
				fiveMinInSeconds    = time.Minute.Seconds() * 5
				tenMinInSeconds     = 2 * fiveMinInSeconds
				mockWattimeForecast = buildMockWattTimeForecast(
					[]WattTimeForecastData{
						{PointInTime: time.Now(), CarbonIntensity: 1},
						{PointInTime: time.Now().Add(time.Duration(tenMinInSeconds) * time.Second), CarbonIntensity: 2},
						{PointInTime: time.Now().Add(2 * time.Duration(tenMinInSeconds) * time.Second), CarbonIntensity: 3},
						{PointInTime: time.Now().Add(3 * time.Duration(tenMinInSeconds) * time.Second), CarbonIntensity: 4},
					},
					int(fiveMinInSeconds),
				)
			)
			It("should error", func() {
				rollupForecast, err := rollupForecast(&mockWattimeForecast.Data, time.Duration(tenMinInSeconds)*time.Second, time.Duration(fiveMinInSeconds)*time.Second)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError("forecast period is longer than the task duration"))
				Expect(rollupForecast).To(BeNil())
			})
		})
	})
})
