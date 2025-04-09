package carbon_forecast

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	mockApiKey        = "test-api-key"
	mockStartTime     = time.Now().Add(time.Minute)
	mockPeriodSeconds = 300
)

func getWattimeClientMockForecast(horizonHours int, periodSeconds int) WattTimeForecast {
	var mockForecastData []WattTimeForecastData
	var numPeriods = int(((time.Duration(horizonHours) * time.Hour) / (time.Duration(periodSeconds) * time.Second)).Seconds())

	for i := 0; i < numPeriods; i++ {
		mockForecastData = append(mockForecastData, WattTimeForecastData{
			PointInTime:     mockStartTime.Add(time.Duration(i) * time.Duration(periodSeconds) * time.Second),
			CarbonIntensity: float64(i),
		})
	}

	mockForecast := WattTimeForecast{
		Data: mockForecastData,
		Metadata: WattTimeForecastMetadata{
			Region:        "CAISO_NORTH",
			PeriodSeconds: mockPeriodSeconds,
		},
	}

	return mockForecast
}

var _ = Describe("WattTimeClient", func() {
	Describe("GetCarbonForecast", func() {
		var (
			serverUrl    string
			server       *MockWattimeServer
			mockForecast WattTimeForecast
		)
		BeforeEach(func() {
			mockForecast = getWattimeClientMockForecast(4, mockPeriodSeconds)
			server = NewMockWattimeServer(mockApiKey, mockPeriodSeconds, &mockForecast)
			serverUrl = server.URL()
		})

		AfterEach(func() {
			server.Close()
		})

		It("should return a list of carbon intensity values", func() {
			client := NewWattTimeClient(mockApiKey, serverUrl)

			forecast, err := client.GetCarbonForecast(mockStartTime, mockStartTime.Add(time.Duration(mockPeriodSeconds)*4*time.Second))
			Expect(err).To(BeNil())
			Expect(len(forecast.Data)).To(Equal(len(mockForecast.Data)))
			for i := range forecast.Data {
				Expect(forecast.Data[i].PointInTime.Unix()).To(Equal(mockForecast.Data[i].PointInTime.Unix()))
				Expect(forecast.Data[i].CarbonIntensity).To(Equal(mockForecast.Data[i].CarbonIntensity))
			}
		})

		It("should return an error if the start time is before the current time", func() {
			client := NewWattTimeClient(mockApiKey)

			forecast, err := client.GetCarbonForecast(mockStartTime.Add(-1*time.Hour), mockStartTime)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("start time must be present or in the future"))
			Expect(forecast).To(BeNil())
		})

		It("should return an error if the end time is more than 72 hours from now", func() {
			client := NewWattTimeClient(mockApiKey)

			forecast, err := client.GetCarbonForecast(mockStartTime, mockStartTime.Add(time.Hour*73))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("end time must be <= 72 hours from now"))
			Expect(forecast).To(BeNil())
		})
	})
})
