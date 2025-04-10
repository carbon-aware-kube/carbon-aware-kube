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
	Describe("Evaluate", func() {
		var (
			startTime               time.Time
			mockWattimeServer       *MockWattimeServer
			mockWattimeClient       *WattTimeClient
			wattimeForecastProvider *WattimeForecastProvider
		)

		BeforeEach(func() {
			startTime = time.Now().Add(time.Minute)
			mockWattimeServer = NewMockWattimeServer(mockApiKey, mockPeriodSeconds, nil)
			mockWattimeClient = NewWattTimeClient(mockApiKey, mockWattimeServer.URL())
			wattimeForecastProvider = NewWattimeForecastProvider(mockWattimeClient)
		})

		AfterEach(func() {
			mockWattimeServer.Close()
		})

		Context("with a flat forecast", func() {
			Context("with a forecast period of 1 hour, horizon of 4 hours", func() {
				BeforeEach(func() {
					mockWattimeServer.SetForecast(&WattTimeForecast{
						Data: []WattTimeForecastData{
							{PointInTime: startTime, CarbonIntensity: 1},
							{PointInTime: startTime.Add(time.Hour), CarbonIntensity: 1},
							{PointInTime: startTime.Add(2 * time.Hour), CarbonIntensity: 1},
							{PointInTime: startTime.Add(3 * time.Hour), CarbonIntensity: 1},
						},
						Metadata: WattTimeForecastMetadata{
							Region:        "CAISO_NORTH",
							PeriodSeconds: 3600,
						},
					})
				})
				Context("with a task duration of 1 hour", func() {
					It("should return now as the start time", func() {
						evaluatedStartTime, err := wattimeForecastProvider.Evaluate(startTime, startTime.Add(4*time.Hour), time.Hour)
						Expect(err).To(BeNil())
						Expect(evaluatedStartTime).To(Equal(startTime))
					})
				})
				Context("with a task duration of 2 hours", func() {
					It("should return now as the start time", func() {
						evaluatedStartTime, err := wattimeForecastProvider.Evaluate(startTime, startTime.Add(4*time.Hour), 2*time.Hour)
						Expect(err).To(BeNil())
						Expect(evaluatedStartTime).To(Equal(startTime))
					})
				})
			})
			Context("with a forecast period of 5 minutes, horizon of 30 minutes", func() {
				BeforeEach(func() {
					fiveMin := time.Minute * 5
					mockWattimeServer.SetForecast(&WattTimeForecast{
						Data: []WattTimeForecastData{
							{PointInTime: startTime, CarbonIntensity: 1},
							{PointInTime: startTime.Add(fiveMin), CarbonIntensity: 1},
							{PointInTime: startTime.Add(2 * fiveMin), CarbonIntensity: 1},
							{PointInTime: startTime.Add(3 * fiveMin), CarbonIntensity: 1},
							{PointInTime: startTime.Add(4 * fiveMin), CarbonIntensity: 1},
							{PointInTime: startTime.Add(5 * fiveMin), CarbonIntensity: 1},
						},
						Metadata: WattTimeForecastMetadata{
							Region:        "CAISO_NORTH",
							PeriodSeconds: int(fiveMin.Seconds()),
						},
					})
				})
				Context("with a task duration of 10 minutes", func() {
					tenMin := time.Minute * 10
					It("should return now as the start time", func() {
						evaluatedStartTime, err := wattimeForecastProvider.Evaluate(startTime, startTime.Add(30*time.Minute), tenMin)
						Expect(err).To(BeNil())
						Expect(evaluatedStartTime).To(Equal(startTime))
					})
				})
				Context("with a task duration of 10 minutes", func() {
					tenMin := time.Minute * 10
					It("should return now as the start time", func() {
						evaluatedStartTime, err := wattimeForecastProvider.Evaluate(startTime, startTime.Add(30*time.Minute), tenMin)
						Expect(err).To(BeNil())
						Expect(evaluatedStartTime).To(Equal(startTime))
					})
				})
			})
		})
		Context("with a forecast that starts high (100) then goes lower (50) then goes higher (100)", func() {
			BeforeEach(func() {
				mockWattimeServer.SetForecast(&WattTimeForecast{
					Data: []WattTimeForecastData{
						{PointInTime: startTime, CarbonIntensity: 100},
						{PointInTime: startTime.Add(time.Hour), CarbonIntensity: 50},
						{PointInTime: startTime.Add(2 * time.Hour), CarbonIntensity: 100},
						{PointInTime: startTime.Add(3 * time.Hour), CarbonIntensity: 100},
					},
					Metadata: WattTimeForecastMetadata{
						Region:        "CAISO_NORTH",
						PeriodSeconds: 3600,
					},
				})
			})
			Context("with a task duration of 1 hour (same as forecast period)", func() {
				It("should return the point in time that the carbon intensity is lowest", func() {
					evaluatedStartTime, err := wattimeForecastProvider.Evaluate(startTime, startTime.Add(4*time.Hour), time.Hour)
					Expect(err).To(BeNil())
					Expect(evaluatedStartTime).To(Equal(startTime.Add(time.Hour)))
				})
			})
			Context("with a task duration of 2 hours (twice the forecast period)", func() {
				It("should return now as the start time (earliest possible start time with the lowest carbon intensity)", func() {
					evaluatedStartTime, err := wattimeForecastProvider.Evaluate(startTime, startTime.Add(4*time.Hour), 2*time.Hour)
					Expect(err).To(BeNil())
					Expect(evaluatedStartTime).To(Equal(startTime))
				})
			})
		})
		Context("with a forecast that goes [100, 50, 10, 0]", func() {
			BeforeEach(func() {
				mockWattimeServer.SetForecast(&WattTimeForecast{
					Data: []WattTimeForecastData{
						{PointInTime: startTime, CarbonIntensity: 100},
						{PointInTime: startTime.Add(time.Hour), CarbonIntensity: 50},
						{PointInTime: startTime.Add(2 * time.Hour), CarbonIntensity: 10},
						{PointInTime: startTime.Add(3 * time.Hour), CarbonIntensity: 0},
					},
					Metadata: WattTimeForecastMetadata{
						Region:        "CAISO_NORTH",
						PeriodSeconds: 3600,
					},
				})
			})
			Context("with a task duration of 3 hours", func() {
				It("should return the second point in time (inclusive of the section with the lowest intensity)", func() {
					evaluatedStartTime, err := wattimeForecastProvider.Evaluate(startTime, startTime.Add(4*time.Hour), 3*time.Hour)
					Expect(err).To(BeNil())
					Expect(evaluatedStartTime).To(Equal(startTime.Add(time.Hour)))
				})
			})
			Context("with a task duration of 2 hours", func() {
				It("should return the third point in time (inclusive of the section with the lowest intensity)", func() {
					evaluatedStartTime, err := wattimeForecastProvider.Evaluate(startTime, startTime.Add(4*time.Hour), 2*time.Hour)
					Expect(err).To(BeNil())
					Expect(evaluatedStartTime).To(Equal(startTime.Add(2 * time.Hour)))
				})
			})
			Context("with a task duration of 1 hour", func() {
				It("should return the third point in time (inclusive of the section with the lowest intensity)", func() {
					evaluatedStartTime, err := wattimeForecastProvider.Evaluate(startTime, startTime.Add(4*time.Hour), time.Hour)
					Expect(err).To(BeNil())
					Expect(evaluatedStartTime).To(Equal(startTime.Add(3 * time.Hour)))
				})
			})
			Context("with a task duration of 30 minutes", func() {
				It("should return the third point in time (inclusive of the section with the lowest intensity)", func() {
					evaluatedStartTime, err := wattimeForecastProvider.Evaluate(startTime, startTime.Add(4*time.Hour), 30*time.Minute)
					Expect(err).To(BeNil())
					Expect(evaluatedStartTime).To(Equal(startTime.Add(3 * time.Hour)))
				})
			})
		})
		Context("with a forecast that goes [100, 50, 10, 0, 20]", func() {
			BeforeEach(func() {
				mockWattimeServer.SetForecast(&WattTimeForecast{
					Data: []WattTimeForecastData{
						{PointInTime: startTime, CarbonIntensity: 100},
						{PointInTime: startTime.Add(time.Hour), CarbonIntensity: 50},
						{PointInTime: startTime.Add(2 * time.Hour), CarbonIntensity: 10},
						{PointInTime: startTime.Add(3 * time.Hour), CarbonIntensity: 0},
						{PointInTime: startTime.Add(4 * time.Hour), CarbonIntensity: 20},
					},
					Metadata: WattTimeForecastMetadata{
						Region:        "CAISO_NORTH",
						PeriodSeconds: 3600,
					},
				})
			})
			Context("with a task duration of 3 hours", func() {
				It("should return the third point in time (inclusive of the section with the lowest intensity)", func() {
					evaluatedStartTime, err := wattimeForecastProvider.Evaluate(startTime, startTime.Add(4*time.Hour), 3*time.Hour)
					Expect(err).To(BeNil())
					Expect(evaluatedStartTime).To(Equal(startTime.Add(2 * time.Hour)))
				})
			})
			Context("with a task duration of 2 hours", func() {
				It("should return the third point in time (inclusive of the section with the lowest intensity)", func() {
					evaluatedStartTime, err := wattimeForecastProvider.Evaluate(startTime, startTime.Add(4*time.Hour), 2*time.Hour)
					Expect(err).To(BeNil())
					Expect(evaluatedStartTime).To(Equal(startTime.Add(2 * time.Hour)))
				})
			})
			Context("with a task duration of 1 hour", func() {
				It("should return the third point in time (inclusive of the section with the lowest intensity)", func() {
					evaluatedStartTime, err := wattimeForecastProvider.Evaluate(startTime, startTime.Add(4*time.Hour), time.Hour)
					Expect(err).To(BeNil())
					Expect(evaluatedStartTime).To(Equal(startTime.Add(3 * time.Hour)))
				})
			})
			Context("with a task duration of 30 minutes", func() {
				It("should return the third point in time (inclusive of the section with the lowest intensity)", func() {
					evaluatedStartTime, err := wattimeForecastProvider.Evaluate(startTime, startTime.Add(4*time.Hour), 30*time.Minute)
					Expect(err).To(BeNil())
					Expect(evaluatedStartTime).To(Equal(startTime.Add(3 * time.Hour)))
				})
			})
		})
	})
})
