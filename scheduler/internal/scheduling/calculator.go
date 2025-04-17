package scheduling

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/carbon-aware-kube/scheduler/internal/sharedtypes"
	"github.com/carbon-aware-kube/scheduler/internal/watttime"
	"github.com/carbon-aware-kube/scheduler/internal/zones"
)

func CalculateBestSchedule(
	ctx context.Context,
	wtClient watttime.WattTimeClientInterface,
	windows []sharedtypes.TimeRange,
	durationStr string,
	powerZones []zones.PowerZone,
	requestedZoneIdentifiers []string,
	numOptions int,
) (*sharedtypes.ScheduleResponse, error) {

	if len(powerZones) == 0 {
		log.Println("CalculateBestSchedule called with no power zones.")
		return nil, fmt.Errorf("internal error: no valid power zones provided for scheduling")
	}

	firstPowerZone := powerZones[0]
	signalType := "co2_moer"

	log.Printf("Requesting WattTime forecast for region: %s, signal: %s", firstPowerZone, signalType)

	forecast, err := wtClient.GetForecast(ctx, firstPowerZone, signalType)
	if err != nil {
		log.Printf("Error getting WattTime forecast: %v", err)
		return nil, fmt.Errorf("failed to retrieve carbon forecast data: %w", err)
	}

	if forecast != nil && len(forecast.Data) > 0 {
		log.Printf("Successfully received WattTime forecast. First data point: %+v", forecast.Data[0])

		// Parse duration once for use in filtering and later calculations
		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			return nil, fmt.Errorf("invalid duration format '%s': %w", durationStr, err)
		}

		// For each window, calculate the extended end time that includes the operation duration
		extendedWindows := make([]struct {
			start       time.Time
			end         time.Time
			extendedEnd time.Time
		}, len(windows))

		for i, win := range windows {
			extendedWindows[i] = struct {
				start       time.Time
				end         time.Time
				extendedEnd time.Time
			}{
				start:       win.Start,
				end:         win.End,
				extendedEnd: win.End.Add(duration),
			}
		}

		// Filter forecast data to include points within any window OR within duration after any window
		filtered := make([]watttime.ForecastDataPoint, 0, len(forecast.Data))
		for _, dp := range forecast.Data {
			for _, win := range extendedWindows {
				// Include data points that are:
				// 1. Within the original window (for task start times), OR
				// 2. Within the extended window (for calculating carbon intensity of tasks that start near window end)
				if !dp.PointTime.Before(win.start) && !dp.PointTime.After(win.extendedEnd) {
					filtered = append(filtered, dp)
					break
				}
			}
		}

		if len(filtered) == 0 {
			return nil, fmt.Errorf("no forecast data points found within the allowed time windows")
		}

		period := time.Duration(forecast.Meta.DataPointPeriodSeconds) * time.Second
		if period <= 0 {
			return nil, fmt.Errorf("invalid forecast data point period: %v seconds", forecast.Meta.DataPointPeriodSeconds)
		}
		pointsNeeded := int(duration / period)
		if duration%period != 0 {
			log.Printf("Warning: Requested duration (%v) is not an exact multiple of forecast period (%v). Results might be approximate.", duration, period)
		}
		if pointsNeeded < 1 {
			pointsNeeded = 1
		}

		type windowResult struct {
			avg   float64
			start int
		}
		var results []windowResult

		for i := 0; (i + pointsNeeded) <= len(filtered); i++ {
			windowStartTime := filtered[i].PointTime
			// We calculate the end time for logging/debugging, but it's no longer used for window validation
			// since we only care about the start time being within the allowed windows
			_ = windowStartTime.Add(duration) // Calculate but ignore to avoid unused variable warning
			lastPointIndex := i + pointsNeeded - 1
			lastPointTime := filtered[lastPointIndex].PointTime

			if lastPointTime.Sub(windowStartTime) != time.Duration(pointsNeeded-1)*period {
				log.Printf("Skipping window at index %d: Non-contiguous data points (gap detected). Start: %v, Expected End Point Time: %v, Actual End Point Time: %v", i, windowStartTime, windowStartTime.Add(time.Duration(pointsNeeded-1)*period), lastPointTime)
				continue
			}

			// Check if the start time is within any of the allowed windows
			// Note: We only check the start time, allowing the task to extend beyond the window's end
			valid := false
			for _, win := range windows {
				if !windowStartTime.Before(win.Start) && !windowStartTime.After(win.End) {
					valid = true
					break
				}
			}
			if !valid {
				continue
			}

			sum := 0.0
			for j := 0; j < pointsNeeded; j++ {
				sum += filtered[i+j].Value
			}
			avg := sum / float64(pointsNeeded)
			results = append(results, windowResult{avg: avg, start: i})
		}

		if len(results) == 0 {
			return nil, fmt.Errorf("no valid scheduling windows found for the requested duration within the allowed time ranges and forecast data")
		}

		sort.Slice(results, func(i, j int) bool {
			if results[i].avg != results[j].avg {
				return results[i].avg < results[j].avg
			}
			return filtered[results[i].start].PointTime.Before(filtered[results[j].start].PointTime)
		})

		firstZoneString := ""
		if len(requestedZoneIdentifiers) > 0 {
			firstZoneString = requestedZoneIdentifiers[0]
		}
		options := []sharedtypes.ScheduleOption{}
		for idx, res := range results {
			if idx >= numOptions {
				break
			}
			startTime := filtered[res.start].PointTime
			options = append(options, sharedtypes.ScheduleOption{
				Time:         startTime,
				Zone:         firstZoneString,
				CO2Intensity: res.avg,
			})
		}

		if len(options) == 0 {
			return nil, fmt.Errorf("internal error: failed to build schedule options after finding valid windows")
		}
		// Find the worst case (highest carbon intensity)
		worstResult := results[len(results)-1] // Since results are sorted by carbon intensity (ascending)
		worstCaseOption := sharedtypes.ScheduleOption{
			Time:         filtered[worstResult.start].PointTime,
			Zone:         firstZoneString,
			CO2Intensity: worstResult.avg,
		}

		// Calculate naive case (first valid time in the earliest window)
		naiveCaseIdx := -1
		for i, dp := range filtered {
			// Find the first data point that is within any window and has enough following points
			if (i + pointsNeeded) <= len(filtered) {
				for _, win := range windows {
					if !dp.PointTime.Before(win.Start) && !dp.PointTime.After(win.End) {
						// Check if we have contiguous points
						lastPointIndex := i + pointsNeeded - 1
						lastPointTime := filtered[lastPointIndex].PointTime
						if lastPointTime.Sub(dp.PointTime) == time.Duration(pointsNeeded-1)*period {
							naiveCaseIdx = i
							break
						}
					}
				}
				if naiveCaseIdx >= 0 {
					break
				}
			}
		}

		// If no naive case was found, use the first valid window
		if naiveCaseIdx < 0 && len(results) > 0 {
			log.Printf("Warning: Could not determine naive case, using first valid window")
			// Find the earliest valid window
			earliest := results[0]
			earliest.start = -1
			earliest.avg = 0
			for _, res := range results {
				if earliest.start < 0 || filtered[res.start].PointTime.Before(filtered[earliest.start].PointTime) {
					earliest = res
				}
			}
			naiveCaseIdx = earliest.start
		}

		// Calculate naive case carbon intensity
		var naiveCaseOption sharedtypes.ScheduleOption
		if naiveCaseIdx >= 0 {
			sum := 0.0
			for j := 0; j < pointsNeeded; j++ {
				sum += filtered[naiveCaseIdx+j].Value
			}
			naiveAvg := sum / float64(pointsNeeded)
			naiveCaseOption = sharedtypes.ScheduleOption{
				Time:         filtered[naiveCaseIdx].PointTime,
				Zone:         firstZoneString,
				CO2Intensity: naiveAvg,
			}
		} else {
			// This should not happen since we have results, but just in case
			naiveCaseOption = options[0]
			log.Printf("Warning: Could not calculate naive case, using ideal case instead")
		}

		// Calculate median case (middle carbon intensity)
		medianResultIdx := len(results) / 2
		medianResult := results[medianResultIdx]
		medianCaseOption := sharedtypes.ScheduleOption{
			Time:         filtered[medianResult.start].PointTime,
			Zone:         firstZoneString,
			CO2Intensity: medianResult.avg,
		}

		// Calculate carbon savings percentages
		ideal := options[0]
		var carbonSavings sharedtypes.CarbonSavings

		// vs Worst Case
		if worstCaseOption.CO2Intensity > 0 {
			carbonSavings.VsWorstCase = ((worstCaseOption.CO2Intensity - ideal.CO2Intensity) / worstCaseOption.CO2Intensity) * 100
		}

		// vs Naive Case
		if naiveCaseOption.CO2Intensity > 0 {
			carbonSavings.VsNaiveCase = ((naiveCaseOption.CO2Intensity - ideal.CO2Intensity) / naiveCaseOption.CO2Intensity) * 100
		}

		// vs Median Case
		if medianCaseOption.CO2Intensity > 0 {
			carbonSavings.VsMedianCase = ((medianCaseOption.CO2Intensity - ideal.CO2Intensity) / medianCaseOption.CO2Intensity) * 100
		}

		return &sharedtypes.ScheduleResponse{
			Ideal:         ideal,
			Options:       options,
			WorstCase:     worstCaseOption,
			NaiveCase:     naiveCaseOption,
			MedianCase:    medianCaseOption,
			CarbonSavings: carbonSavings,
		}, nil
	} else {
		log.Printf("WattTime forecast received but contains no data points for region %s", firstPowerZone)
		return nil, fmt.Errorf("forecast received but contains no data points for region %s", firstPowerZone)
	}
}
