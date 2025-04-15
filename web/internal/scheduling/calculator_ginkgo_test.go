package scheduling_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/carbon-aware-kube/web/internal/scheduling"
	"github.com/carbon-aware-kube/web/internal/sharedtypes"
	"github.com/carbon-aware-kube/web/internal/watttime"
	"github.com/carbon-aware-kube/web/internal/zones"
)

var _ = Describe("CalculateBestSchedule", func() {
	var (
		ctx         context.Context
		mockClient  *watttime.MockWattTimeClient
		zone        zones.PowerZone
		zoneList    []zones.PowerZone
		zoneStrList []string
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockClient = &watttime.MockWattTimeClient{}
		zone = zones.PowerZone("TestZone")
		zoneList = []zones.PowerZone{zone}
		zoneStrList = []string{"TestZone"}
	})

	// 1. Basic Functionality
	Describe("Basic scheduling behavior", func() {
		It("selects the lowest carbon slot in a single window", func() {
			start := time.Now().UTC().Truncate(time.Hour)
			mockClient.ForecastResponse = &watttime.ForecastResponse{
				Meta: watttime.ForecastMeta{Region: string(zone), SignalType: "co2_moer", DataPointPeriodSeconds: 300, GeneratedAt: start},
				Data: []watttime.ForecastDataPoint{
					{PointTime: start, Value: 100},
					{PointTime: start.Add(5 * time.Minute), Value: 50},
					{PointTime: start.Add(10 * time.Minute), Value: 150},
				},
			}
			windows := []sharedtypes.TimeRange{{Start: start, End: start.Add(20 * time.Minute)}}
			res, err := scheduling.CalculateBestSchedule(ctx, mockClient, windows, "5m", zoneList, zoneStrList, 3)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Ideal.Time).To(Equal(start.Add(5 * time.Minute)))
			Expect(res.Ideal.CO2Intensity).To(Equal(50.0))
		})

		It("selects the best slot in the second window if it is better", func() {
			start := time.Now().UTC().Truncate(time.Hour)
			mockClient.ForecastResponse = &watttime.ForecastResponse{
				Meta: watttime.ForecastMeta{Region: string(zone), SignalType: "co2_moer", DataPointPeriodSeconds: 300, GeneratedAt: start},
				Data: []watttime.ForecastDataPoint{
					{PointTime: start, Value: 90},
					{PointTime: start.Add(5 * time.Minute), Value: 80},
					{PointTime: start.Add(10 * time.Minute), Value: 10}, // Only in second window
				},
			}
			windows := []sharedtypes.TimeRange{
				{Start: start, End: start.Add(9 * time.Minute)},
				{Start: start.Add(10 * time.Minute), End: start.Add(20 * time.Minute)},
			}
			res, err := scheduling.CalculateBestSchedule(ctx, mockClient, windows, "5m", zoneList, zoneStrList, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Ideal.Time).To(Equal(start.Add(10 * time.Minute)))
			Expect(res.Ideal.CO2Intensity).To(Equal(10.0))
		})

		It("selects the only valid slot when operation length matches exactly one interval", func() {
			start := time.Now().UTC().Truncate(time.Hour)
			mockClient.ForecastResponse = &watttime.ForecastResponse{
				Meta: watttime.ForecastMeta{Region: string(zone), SignalType: "co2_moer", DataPointPeriodSeconds: 600, GeneratedAt: start},
				Data: []watttime.ForecastDataPoint{
					{PointTime: start, Value: 123},
				},
			}
			windows := []sharedtypes.TimeRange{{Start: start, End: start.Add(10 * time.Minute)}}
			res, err := scheduling.CalculateBestSchedule(ctx, mockClient, windows, "10m", zoneList, zoneStrList, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Ideal.Time).To(Equal(start))
			Expect(res.Ideal.CO2Intensity).To(Equal(123.0))
		})
	})

	// 2. Window and Duration Handling
	Describe("Window and duration handling", func() {
		It("allows operations that start within a window but extend beyond it", func() {
			start := time.Now().UTC().Truncate(time.Hour)
			mockClient.ForecastResponse = &watttime.ForecastResponse{
				Meta: watttime.ForecastMeta{Region: string(zone), SignalType: "co2_moer", DataPointPeriodSeconds: 300, GeneratedAt: start},
				Data: []watttime.ForecastDataPoint{
					{PointTime: start, Value: 100},
					{PointTime: start.Add(5 * time.Minute), Value: 200},
				},
			}
			windows := []sharedtypes.TimeRange{{Start: start, End: start.Add(5 * time.Minute)}}
			// With the updated behavior, operations can start within a window and extend beyond it
			// So this should now succeed even though the operation (10m) is longer than the window (5m)
			res, err := scheduling.CalculateBestSchedule(ctx, mockClient, windows, "10m", zoneList, zoneStrList, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Ideal.Time).To(Equal(start))
			Expect(res.Ideal.CO2Intensity).To(Equal(150.0))
		})

		It("selects the interval with the lowest average intensity if operation length is shorter than the forecast interval", func() {
			start := time.Now().UTC().Truncate(time.Hour)
			mockClient.ForecastResponse = &watttime.ForecastResponse{
				Meta: watttime.ForecastMeta{Region: string(zone), SignalType: "co2_moer", DataPointPeriodSeconds: 600, GeneratedAt: start},
				Data: []watttime.ForecastDataPoint{
					{PointTime: start, Value: 10},
					{PointTime: start.Add(10 * time.Minute), Value: 20},
				},
			}
			windows := []sharedtypes.TimeRange{{Start: start, End: start.Add(20 * time.Minute)}}
			res, err := scheduling.CalculateBestSchedule(ctx, mockClient, windows, "5m", zoneList, zoneStrList, 1)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Ideal.Time).To(Equal(start))
			Expect(res.Ideal.CO2Intensity).To(Equal(10.0))
		})

		It("averages intensity over the operation period spanning multiple data points", func() {
			start := time.Now().UTC().Truncate(time.Hour)
			mockClient.ForecastResponse = &watttime.ForecastResponse{
				Meta: watttime.ForecastMeta{Region: string(zone), SignalType: "co2_moer", DataPointPeriodSeconds: 300, GeneratedAt: start},
				Data: []watttime.ForecastDataPoint{
					{PointTime: start, Value: 10},
					{PointTime: start.Add(5 * time.Minute), Value: 30},
					{PointTime: start.Add(10 * time.Minute), Value: 50},
				},
			}
			windows := []sharedtypes.TimeRange{{Start: start, End: start.Add(15 * time.Minute)}}
			res, err := scheduling.CalculateBestSchedule(ctx, mockClient, windows, "10m", zoneList, zoneStrList, 2)
			Expect(err).NotTo(HaveOccurred())
			// The best slot is start (average of 10 and 30 = 20), next is start+5m (average 30 and 50 = 40)
			Expect(res.Ideal.Time).To(Equal(start))
			Expect(res.Ideal.CO2Intensity).To(Equal(20.0))
			Expect(res.Options[1].Time).To(Equal(start.Add(5 * time.Minute)))
			Expect(res.Options[1].CO2Intensity).To(Equal(40.0))
		})

		It("finds the best slot at the overlap of multiple windows", func() {
			start := time.Now().UTC().Truncate(time.Hour)
			mockClient.ForecastResponse = &watttime.ForecastResponse{
				Meta: watttime.ForecastMeta{Region: string(zone), SignalType: "co2_moer", DataPointPeriodSeconds: 300, GeneratedAt: start},
				Data: []watttime.ForecastDataPoint{
					{PointTime: start, Value: 100},
					{PointTime: start.Add(5 * time.Minute), Value: 50},
					{PointTime: start.Add(10 * time.Minute), Value: 200},
				},
			}
			windows := []sharedtypes.TimeRange{
				{Start: start, End: start.Add(10 * time.Minute)},
				{Start: start.Add(5 * time.Minute), End: start.Add(15 * time.Minute)},
			}
			res, err := scheduling.CalculateBestSchedule(ctx, mockClient, windows, "5m", zoneList, zoneStrList, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Ideal.Time).To(Equal(start.Add(5 * time.Minute)))
			Expect(res.Ideal.CO2Intensity).To(Equal(50.0))
		})
	})

	// 7. Tiebreakers and Sorting
	Describe("Tiebreakers and sorting", func() {
		It("picks the earliest slot when multiple have the same lowest intensity", func() {
			start := time.Now().UTC().Truncate(time.Hour)
			mockClient.ForecastResponse = &watttime.ForecastResponse{
				Meta: watttime.ForecastMeta{Region: string(zone), SignalType: "co2_moer", DataPointPeriodSeconds: 300, GeneratedAt: start},
				Data: []watttime.ForecastDataPoint{
					{PointTime: start, Value: 50},
					{PointTime: start.Add(5 * time.Minute), Value: 50},
					{PointTime: start.Add(10 * time.Minute), Value: 100},
				},
			}
			windows := []sharedtypes.TimeRange{{Start: start, End: start.Add(15 * time.Minute)}}
			res, err := scheduling.CalculateBestSchedule(ctx, mockClient, windows, "5m", zoneList, zoneStrList, 2)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Ideal.Time).To(Equal(start))
			Expect(res.Ideal.CO2Intensity).To(Equal(50.0))
		})

		It("sorts options by intensity then by time", func() {
			start := time.Now().UTC().Truncate(time.Hour)
			mockClient.ForecastResponse = &watttime.ForecastResponse{
				Meta: watttime.ForecastMeta{Region: string(zone), SignalType: "co2_moer", DataPointPeriodSeconds: 300, GeneratedAt: start},
				Data: []watttime.ForecastDataPoint{
					{PointTime: start, Value: 100},
					{PointTime: start.Add(5 * time.Minute), Value: 50},
					{PointTime: start.Add(10 * time.Minute), Value: 100},
					{PointTime: start.Add(15 * time.Minute), Value: 50},
				},
			}
			windows := []sharedtypes.TimeRange{{Start: start, End: start.Add(20 * time.Minute)}}
			res, err := scheduling.CalculateBestSchedule(ctx, mockClient, windows, "5m", zoneList, zoneStrList, 4)
			Expect(err).NotTo(HaveOccurred())
			// Should be sorted: 50, 50, 100, 100 (times: 5m, 15m, 0m, 10m)
			Expect(res.Options[0].CO2Intensity).To(Equal(50.0))
			Expect(res.Options[0].Time).To(Equal(start.Add(5 * time.Minute)))
			Expect(res.Options[1].CO2Intensity).To(Equal(50.0))
			Expect(res.Options[1].Time).To(Equal(start.Add(15 * time.Minute)))
			Expect(res.Options[2].CO2Intensity).To(Equal(100.0))
			Expect(res.Options[2].Time).To(Equal(start))
			Expect(res.Options[3].CO2Intensity).To(Equal(100.0))
			Expect(res.Options[3].Time).To(Equal(start.Add(10 * time.Minute)))
		})

		It("returns options sorted by time when all intensities are identical", func() {
			start := time.Now().UTC().Truncate(time.Hour)
			mockClient.ForecastResponse = &watttime.ForecastResponse{
				Meta: watttime.ForecastMeta{Region: string(zone), SignalType: "co2_moer", DataPointPeriodSeconds: 300, GeneratedAt: start},
				Data: []watttime.ForecastDataPoint{
					{PointTime: start, Value: 77},
					{PointTime: start.Add(5 * time.Minute), Value: 77},
					{PointTime: start.Add(10 * time.Minute), Value: 77},
				},
			}
			windows := []sharedtypes.TimeRange{{Start: start, End: start.Add(15 * time.Minute)}}
			res, err := scheduling.CalculateBestSchedule(ctx, mockClient, windows, "5m", zoneList, zoneStrList, 3)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Options[0].Time).To(Equal(start))
			Expect(res.Options[1].Time).To(Equal(start.Add(5 * time.Minute)))
			Expect(res.Options[2].Time).To(Equal(start.Add(10 * time.Minute)))
		})

		It("produces deterministic output with shuffled input", func() {
			start := time.Now().UTC().Truncate(time.Hour)
			data := []watttime.ForecastDataPoint{
				{PointTime: start.Add(10 * time.Minute), Value: 100},
				{PointTime: start, Value: 50},
				{PointTime: start.Add(5 * time.Minute), Value: 75},
				{PointTime: start.Add(15 * time.Minute), Value: 50},
			}
			mockClient.ForecastResponse = &watttime.ForecastResponse{
				Meta: watttime.ForecastMeta{Region: string(zone), SignalType: "co2_moer", DataPointPeriodSeconds: 300, GeneratedAt: start},
				Data: data,
			}
			windows := []sharedtypes.TimeRange{{Start: start, End: start.Add(20 * time.Minute)}}
			res1, err := scheduling.CalculateBestSchedule(ctx, mockClient, windows, "5m", zoneList, zoneStrList, 4)
			Expect(err).NotTo(HaveOccurred())
			// Shuffle data and try again
			mockClient.ForecastResponse.Data = []watttime.ForecastDataPoint{
				data[2], data[0], data[3], data[1],
			}
			res2, err := scheduling.CalculateBestSchedule(ctx, mockClient, windows, "5m", zoneList, zoneStrList, 4)
			Expect(err).NotTo(HaveOccurred())
			Expect(res1.Options).To(Equal(res2.Options))
		})
	})
})
