package carbon_forecast

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strconv"
)

func NewMockWattimeServer(mockApiKey string, mockPeriodSeconds int, mockForecastBuilder func(horizonHours int, periodSeconds int) WattTimeForecast) *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != WattTimeApiPath {
				log.Printf("request path check failed: %s", r.URL.Path)
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || authHeader != "Bearer "+mockApiKey {
				log.Printf("auth header check failed: %s", authHeader)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			horizonHours := r.URL.Query().Get("horizon_hours")
			if horizonHours == "" {
				http.Error(w, "horizon_hours is required", http.StatusBadRequest)
				return
			}

			horizonHoursInt, _ := strconv.Atoi(horizonHours)
			mockForecast := mockForecastBuilder(horizonHoursInt, mockPeriodSeconds)
			mockForecastJson, _ := json.Marshal(mockForecast)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(mockForecastJson)
		}),
	)
}
