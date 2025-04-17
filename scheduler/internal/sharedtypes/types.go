package sharedtypes

import "time"

// TimeRange represents a time window with a start and end time.
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ScheduleRequest represents the input for the /api/schedule endpoint.
type ScheduleRequest struct {
	Windows    []TimeRange `json:"windows"`
	Duration   string      `json:"duration"`
	Zones      []string    `json:"zones"`
	NumOptions *int        `json:"numOptions,omitempty"` // Optional, defaults to 3
}

// ScheduleOption represents a potential scheduling option.
type ScheduleOption struct {
	Time         time.Time `json:"time"`
	Zone         string    `json:"zone"` // Power zone identifier (e.g. "CAISO_NORTH") or original cloud region string for stub
	CO2Intensity float64   `json:"co2Intensity"`
}

// CarbonSavings represents the carbon savings compared to different scenarios.
type CarbonSavings struct {
	VsWorstCase  float64 `json:"vsWorstCase"`  // Percentage of carbon saved compared to worst case
	VsNaiveCase  float64 `json:"vsNaiveCase"`  // Percentage of carbon saved compared to naive case
	VsMedianCase float64 `json:"vsMedianCase"` // Percentage of carbon saved compared to median case
}

// ScheduleResponse represents the output for the /api/schedule endpoint.
type ScheduleResponse struct {
	Ideal       ScheduleOption   `json:"ideal"`
	Options     []ScheduleOption `json:"options"`
	WorstCase   ScheduleOption   `json:"worstCase"`   // Time with highest carbon intensity
	NaiveCase   ScheduleOption   `json:"naiveCase"`   // Carbon intensity if scheduled at start of window
	MedianCase  ScheduleOption   `json:"medianCase"`  // Time with median carbon intensity
	CarbonSavings CarbonSavings  `json:"carbonSavings"` // Carbon savings compared to different scenarios
}
