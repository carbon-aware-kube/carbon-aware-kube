package scheduling

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/carbon-aware-kube/web/internal/sharedtypes"
	"github.com/carbon-aware-kube/web/internal/watttime"
	"github.com/carbon-aware-kube/web/internal/zones"
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
			start time.Time
			end   time.Time
			extendedEnd time.Time
		}, len(windows))

		for i, win := range windows {
			extendedWindows[i] = struct {
				start time.Time
				end   time.Time
				extendedEnd time.Time
			}{
				start: win.Start,
				end:   win.End,
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
				if (!dp.PointTime.Before(win.start) && !dp.PointTime.After(win.extendedEnd)) {
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
		ideal := options[0]
		return &sharedtypes.ScheduleResponse{
			Ideal:   ideal,
			Options: options,
		}, nil
	} else {
		log.Printf("WattTime forecast received but contains no data points for region %s", firstPowerZone)
		return nil, fmt.Errorf("forecast received but contains no data points for region %s", firstPowerZone)
	}
}
