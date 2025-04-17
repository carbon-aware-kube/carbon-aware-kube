package watttime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync" // For handling token refresh safely
	"time"

	"github.com/carbon-aware-kube/scheduler/internal/zones" // To use PowerZone type
)

const (
	baseURL      = "https://api.watttime.org"
	loginPath    = "/login"
	forecastPath = "/v3/forecast"
)

// Client handles communication with the WattTime API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	username   string
	password   string
	token      string
	tokenMu    sync.RWMutex // Protects token access
}

// NewClient creates a new WattTime API client.
// Reads WATTIME_USERNAME and WATTIME_PASSWORD from environment variables.
func NewClient() (*Client, error) {
	username := os.Getenv("WATTIME_USERNAME")
	password := os.Getenv("WATTIME_PASSWORD")

	if username == "" || password == "" {
		return nil, fmt.Errorf("WATTIME_USERNAME and WATTIME_PASSWORD environment variables must be set")
	}

	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    baseURL,
		username:   username,
		password:   password,
	}, nil
}

// login fetches an API token using basic authentication.
func (c *Client) login(ctx context.Context) error {
	c.tokenMu.Lock() // Get exclusive lock for potential token update
	defer c.tokenMu.Unlock()

	// Check if token might have been refreshed by another goroutine while waiting for lock
	if c.token != "" {
		return nil // Assume token is still valid, GetForecast can retry login if needed
	}

	loginURL := c.baseURL + loginPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, loginURL, nil)
	if err != nil {
		return fmt.Errorf("error creating login request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error performing login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Consider reading body for more error details if API provides them
		bodyStr := readBody(resp) // Use helper to attempt reading body
		return fmt.Errorf("login failed with status code: %d, body: %s", resp.StatusCode, bodyStr)
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("error decoding login response: %w", err)
	}

	if loginResp.Token == "" {
		return fmt.Errorf("login successful but token is empty")
	}

	c.token = loginResp.Token
	fmt.Println("Successfully obtained WattTime token.") // Log success (optional)
	return nil
}

// GetForecast fetches the carbon intensity forecast for a specific region.
// It handles login automatically if no token is present.
func (c *Client) GetForecast(ctx context.Context, region zones.PowerZone, signalType string) (*ForecastResponse, error) {
	// Ensure token exists, login if necessary
	c.tokenMu.RLock() // Read lock to check token
	token := c.token
	c.tokenMu.RUnlock()

	if token == "" {
		if err := c.login(ctx); err != nil {
			return nil, fmt.Errorf("automatic login failed: %w", err)
		}
		c.tokenMu.RLock() // Re-acquire read lock after login
		token = c.token
		c.tokenMu.RUnlock()
	}

	// Build URL with parameters
	forecastURL, err := url.Parse(c.baseURL + forecastPath)
	if err != nil {
		return nil, fmt.Errorf("error parsing forecast base URL: %w", err) // Should not happen
	}
	params := url.Values{}
	params.Add("region", string(region))
	if signalType != "" {
		params.Add("signal_type", signalType)
	} // Add other params like start/end time if needed
	forecastURL.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, forecastURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating forecast request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error performing forecast request: %w", err)
	}
	defer resp.Body.Close()

	// Handle potential token expiry (WattTime might return 401 or 403)
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		fmt.Println("WattTime token potentially expired or invalid, attempting re-login...")
		// Invalidate current token and retry login + forecast ONCE
		c.tokenMu.Lock()
		c.token = ""
		c.tokenMu.Unlock()
		// Re-login
		if err := c.login(ctx); err != nil {
			return nil, fmt.Errorf("re-login failed after auth error: %w", err)
		}
		// Get new token
		c.tokenMu.RLock()
		newToken := c.token
		c.tokenMu.RUnlock()
		// Retry the request with the new token
		req.Header.Set("Authorization", "Bearer "+newToken)
		// Need to recreate the request body reader if it was consumed, but GET has no body
		// Ensure we close the previous response body before making a new request
		resp.Body.Close()                                 // Close the old response body explicitly
		resp, err = c.httpClient.Do(req.WithContext(ctx)) // Re-execute the request using the same context
		if err != nil {
			return nil, fmt.Errorf("error performing forecast request after re-login: %w", err)
		}
		defer resp.Body.Close() // Ensure body is closed on the retry path too
	}

	if resp.StatusCode != http.StatusOK {
		// Use helper to attempt reading body for better error reporting
		bodyStr := readBody(resp)
		return nil, fmt.Errorf("forecast request failed with status code: %d, body: %s", resp.StatusCode, bodyStr)
	}

	var forecastResp ForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&forecastResp); err != nil {
		return nil, fmt.Errorf("error decoding forecast response: %w", err)
	}

	// Add basic validation on response
	if forecastResp.Meta.Region != string(region) {
		fmt.Printf("Warning: WattTime response region '%s' does not match requested region '%s'\n", forecastResp.Meta.Region, region)
		// Decide if this should be an error or just a warning
	}

	return &forecastResp, nil
}

// Helper function to read response body safely, returning the original body reader.
func readBody(resp *http.Response) string {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		// Restore original closer even on error
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		return fmt.Sprintf("[error reading body: %v]", err)
	}
	// Restore original closer so JSON decoder can read it
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return string(bodyBytes)
}
