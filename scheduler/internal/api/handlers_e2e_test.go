package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/carbon-aware-kube/scheduler/internal/api"
	"github.com/carbon-aware-kube/scheduler/internal/sharedtypes"
	"github.com/carbon-aware-kube/scheduler/internal/watttime"
	"github.com/carbon-aware-kube/scheduler/internal/zones"
	"github.com/go-chi/chi/v5"
)

var _ = Describe("Schedule API Endpoint E2E", func() {
	var (
		mockWtClient     *watttime.MockWattTimeClient
		mockZoneLookup   zones.ZoneLookup
		handler          *api.Handler
		router           *chi.Mux
		rr               *httptest.ResponseRecorder
		testRegionAbbrev string
		startTime        time.Time
	)

	BeforeEach(func() {
		// 1. Setup Mocks
		mockWtClient = &watttime.MockWattTimeClient{}
		testRegionAbbrev = "TEST_REGION"
		startTime = time.Now().UTC().Truncate(5 * time.Minute)
		mockWtClient.ForecastResponse = &watttime.ForecastResponse{
			Meta: watttime.ForecastMeta{
				Region:                 testRegionAbbrev,
				SignalType:             "co2_moer",
				DataPointPeriodSeconds: 300,                       // 5 minutes
				GeneratedAt:            startTime.Add(-time.Hour), // Arbitrary past time
			},
			Data: []watttime.ForecastDataPoint{
				{PointTime: startTime, Value: 100},
				{PointTime: startTime.Add(5 * time.Minute), Value: 50}, // Best slot
				{PointTime: startTime.Add(10 * time.Minute), Value: 75},
				{PointTime: startTime.Add(15 * time.Minute), Value: 120},
			},
		}

		// Mock ZoneLookup
		mockZoneLookup = zones.NewMockStaticZoneLookup(map[string]zones.PowerZone{
			"TestZone": zones.PowerZone(testRegionAbbrev),
		})

		// 2. Setup Handler & Router
		handler = api.NewHandler(mockWtClient, mockZoneLookup)
		router = chi.NewRouter()
		router.Post("/api/schedule", handler.ScheduleHandler) // Use exported handler name

		// Recorder for response
		rr = httptest.NewRecorder()
	})

	Context("when receiving a valid schedule request", func() {
		It("should return the best schedule based on mock forecast data", func() {
			// 3. Prepare Request
			numOptions := 3
			reqBody := sharedtypes.ScheduleRequest{
				Windows: []sharedtypes.TimeRange{
					{Start: startTime.Add(-time.Hour), End: startTime.Add(time.Hour)}, // Wide window encompassing data
				},
				Duration:   "5m",
				Zones:      []string{"TestZone"},
				NumOptions: &numOptions,
			}
			bodyBytes, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred()) // Gomega assertion

			req, err := http.NewRequestWithContext(context.Background(), "POST", "/api/schedule", bytes.NewReader(bodyBytes))
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")

			// 4. Execute Request
			router.ServeHTTP(rr, req)

			// 5. Assert Response
			Expect(rr.Code).To(Equal(http.StatusOK)) // Gomega assertion

			var respBody sharedtypes.ScheduleResponse
			err = json.Unmarshal(rr.Body.Bytes(), &respBody)
			Expect(err).NotTo(HaveOccurred())

			// Expect the best time slot (lowest value) to be returned
			Expect(respBody.Ideal.Time).To(Equal(startTime.Add(5 * time.Minute))) // Gomega assertion
			Expect(respBody.Ideal.CO2Intensity).To(Equal(50.0))                   // Gomega assertion
			Expect(respBody.Ideal.Zone).To(Equal("TestZone"))                     // Gomega assertion

			Expect(len(respBody.Options)).To(Equal(3)) // Gomega assertion
			// Check first option (should be the ideal one)
			Expect(respBody.Options[0].Time).To(Equal(startTime.Add(5 * time.Minute))) // Gomega assertion
			Expect(respBody.Options[0].CO2Intensity).To(Equal(50.0))                   // Gomega assertion
			// Check second option
			Expect(respBody.Options[1].Time).To(Equal(startTime.Add(10 * time.Minute))) // Gomega assertion
			Expect(respBody.Options[1].CO2Intensity).To(Equal(75.0))                    // Gomega assertion
			// Check third option
			Expect(respBody.Options[2].Time).To(Equal(startTime))     // Gomega assertion
			Expect(respBody.Options[2].CO2Intensity).To(Equal(100.0)) // Gomega assertion
		})

		It("should include carbon savings data in the response", func() {
			// Setup a forecast with a clear pattern for testing carbon savings
			startTime := time.Now().UTC().Truncate(5 * time.Minute)
			mockWtClient.ForecastResponse = &watttime.ForecastResponse{
				Meta: watttime.ForecastMeta{
					Region:                 testRegionAbbrev,
					SignalType:             "co2_moer",
					DataPointPeriodSeconds: 300, // 5 minutes
					GeneratedAt:            startTime.Add(-time.Hour),
				},
				Data: []watttime.ForecastDataPoint{
					{PointTime: startTime, Value: 500},                      // Naive case (start of window)
					{PointTime: startTime.Add(5 * time.Minute), Value: 400}, // Ideal case (lowest)
					{PointTime: startTime.Add(10 * time.Minute), Value: 600},
					{PointTime: startTime.Add(15 * time.Minute), Value: 650},
					{PointTime: startTime.Add(20 * time.Minute), Value: 700}, // Median case
					{PointTime: startTime.Add(25 * time.Minute), Value: 750},
					{PointTime: startTime.Add(30 * time.Minute), Value: 800}, // Worst case
				},
			}

			// Prepare request
			numOptions := 3
			reqBody := sharedtypes.ScheduleRequest{
				Windows: []sharedtypes.TimeRange{
					{Start: startTime, End: startTime.Add(35 * time.Minute)},
				},
				Duration:   "5m",
				Zones:      []string{"TestZone"},
				NumOptions: &numOptions,
			}
			bodyBytes, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequestWithContext(context.Background(), "POST", "/api/schedule", bytes.NewReader(bodyBytes))
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")

			// Execute request
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			// Assert response
			Expect(rr.Code).To(Equal(http.StatusOK))

			var respBody sharedtypes.ScheduleResponse
			err = json.Unmarshal(rr.Body.Bytes(), &respBody)
			Expect(err).NotTo(HaveOccurred())

			// Verify all the new fields are present
			Expect(respBody.Ideal).NotTo(BeNil())
			Expect(respBody.WorstCase).NotTo(BeNil())
			Expect(respBody.NaiveCase).NotTo(BeNil())
			Expect(respBody.MedianCase).NotTo(BeNil())

			// Verify the ideal case is the lowest carbon intensity
			Expect(respBody.Ideal.Time).To(Equal(startTime.Add(5 * time.Minute)))
			Expect(respBody.Ideal.CO2Intensity).To(Equal(400.0))

			// Verify worst case has highest carbon intensity
			Expect(respBody.WorstCase.CO2Intensity).To(BeNumerically(">", respBody.Ideal.CO2Intensity))

			// Verify naive case is at the start of the window
			Expect(respBody.NaiveCase.Time).To(Equal(startTime))

			// Verify carbon savings are calculated correctly
			Expect(respBody.CarbonSavings.VsWorstCase).To(BeNumerically(">", 0))
			Expect(respBody.CarbonSavings.VsMedianCase).To(BeNumerically(">", 0))

			// If naive case is not the same as ideal, savings should be positive
			if !respBody.NaiveCase.Time.Equal(respBody.Ideal.Time) {
				Expect(respBody.CarbonSavings.VsNaiveCase).To(BeNumerically(">", 0))
			}

			// Log the actual values for debugging
			GinkgoWriter.Printf("Ideal: %v, CO2: %.2f\n", respBody.Ideal.Time, respBody.Ideal.CO2Intensity)
			GinkgoWriter.Printf("Worst: %v, CO2: %.2f\n", respBody.WorstCase.Time, respBody.WorstCase.CO2Intensity)
			GinkgoWriter.Printf("Naive: %v, CO2: %.2f\n", respBody.NaiveCase.Time, respBody.NaiveCase.CO2Intensity)
			GinkgoWriter.Printf("Median: %v, CO2: %.2f\n", respBody.MedianCase.Time, respBody.MedianCase.CO2Intensity)
			GinkgoWriter.Printf("Carbon Savings - vs Worst: %.2f%%, vs Naive: %.2f%%, vs Median: %.2f%%\n",
				respBody.CarbonSavings.VsWorstCase,
				respBody.CarbonSavings.VsNaiveCase,
				respBody.CarbonSavings.VsMedianCase)
		})
	})

	Context("when receiving an invalid request body", func() {
		It("should return Bad Request for malformed JSON", func() {
			malformedJSON := []byte(`{ "location": "TestZone", "duration": "PT5M" `) // Missing closing brace

			req, err := http.NewRequestWithContext(context.Background(), "POST", "/api/schedule", bytes.NewReader(malformedJSON))
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
		})

		It("should return Bad Request for missing required fields (e.g., duration)", func() {
			reqBody := map[string]interface{}{
				"zones": []string{"TestZone"},
				// duration is missing
				"windows": []map[string]time.Time{{"start": startTime, "end": startTime.Add(time.Hour)}},
			}
			bodyBytes, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequestWithContext(context.Background(), "POST", "/api/schedule", bytes.NewReader(bodyBytes))
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")

			router.ServeHTTP(rr, req)

			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			// Optionally check error message in body
			Expect(rr.Body.String()).To(ContainSubstring("Missing required field 'duration'")) // Match actual error
		})
	})

	Context("when WattTime client returns an error", func() {
		BeforeEach(func() {
			// Configure the mock to return an error
			mockWtClient.ForecastResponse = nil // Ensure no valid response is set
			mockWtClient.Error = fmt.Errorf("WattTime API unavailable")
		})

		It("should return Internal Server Error", func() {
			// Prepare a valid request
			numOptions := 2 // Use a valid number of options (2-10)
			reqBody := sharedtypes.ScheduleRequest{
				Windows: []sharedtypes.TimeRange{
					{Start: startTime.Add(-time.Hour), End: startTime.Add(time.Hour)},
				},
				Duration:   "5m",
				Zones:      []string{"TestZone"},
				NumOptions: &numOptions,
			}
			bodyBytes, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequestWithContext(context.Background(), "POST", "/api/schedule", bytes.NewReader(bodyBytes))
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")

			// Execute Request
			router.ServeHTTP(rr, req)

			// Assert Response
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(rr.Body.String()).To(ContainSubstring("Failed to calculate schedule")) // Match actual error prefix
		})
	})

	Context("when the zone identifier is invalid", func() {
		It("should return Bad Request", func() {
			// Prepare request with an unknown zone
			numOptions := 1
			reqBody := sharedtypes.ScheduleRequest{
				Windows: []sharedtypes.TimeRange{
					{Start: startTime.Add(-time.Hour), End: startTime.Add(time.Hour)},
				},
				Duration:   "5m",
				Zones:      []string{"UnknownZone"}, // This zone is not in mockZoneLookup
				NumOptions: &numOptions,
			}
			bodyBytes, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req, err := http.NewRequestWithContext(context.Background(), "POST", "/api/schedule", bytes.NewReader(bodyBytes))
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")

			// Execute Request
			router.ServeHTTP(rr, req)

			// Assert Response
			Expect(rr.Code).To(Equal(http.StatusBadRequest))
			Expect(rr.Body.String()).To(ContainSubstring("Invalid zone identifier provided")) // Match actual error
		})
	})
})
