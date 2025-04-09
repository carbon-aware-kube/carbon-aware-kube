package carbon_forecast

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
)

type MockWattimeServer struct {
	server            *httptest.Server
	mockApiKey        string
	mockPeriodSeconds int
	mockForecast      *WattTimeForecast
}

func NewMockWattimeServer(mockApiKey string, mockPeriodSeconds int, mockForecast *WattTimeForecast) *MockWattimeServer {
	s := &MockWattimeServer{
		mockApiKey:        mockApiKey,
		mockPeriodSeconds: mockPeriodSeconds,
		mockForecast:      mockForecast,
	}

	s.server = httptest.NewServer(
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

			mockForecastJson, _ := json.Marshal(s.mockForecast)

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(mockForecastJson)
		}),
	)

	return s
}

func (s *MockWattimeServer) Close() {
	s.server.Close()
}

func (s *MockWattimeServer) URL() string {
	return s.server.URL
}

func (s *MockWattimeServer) SetForecast(mockForecast *WattTimeForecast) {
	s.mockForecast = mockForecast
}
