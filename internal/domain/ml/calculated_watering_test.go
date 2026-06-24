package ml_test

import (
	"testing"
	"time"

	"github.com/bruli/go-core/event"
	"github.com/bruli/go-core/ptr"
	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

const (
	bonsaiBigZone   = "bonsai big"
	bonsaiSmallZone = "bonsai small"
	humidity100     = 1.2
	humidity40      = 1.8
	highHumidity    = 1.45
	lowHumidity     = 1.85
	mediumHumidity  = 1.7
)

var dayTimeFunc = func() time.Time {
	return time.Date(2021, 1, 1, 10, 0, 0, 0, time.UTC)
}

func TestNewCalculatedPrediction(t *testing.T) {
	humidityChecker := ml.NewHumidityReference(humidity40, humidity100)

	type args struct {
		isRaining       bool
		systemActivated bool
		exec            ml.Executions
		zonesHumidity   []*ml.ZoneHumidity
	}
	tests := []struct {
		name               string
		args               args
		expectedCalculated bool
		expectedEvents     []event.Event
		expectedEventsLen  int
		expectedReasons    []string
	}{
		{
			name: "with a high humidity value in both zones, then it returns a watering zone skipped events",
			args: args{
				zonesHumidity: []*ml.ZoneHumidity{
					ml.NewZoneHumidity(bonsaiBigZone, highHumidity, humidityChecker),
					ml.NewZoneHumidity(bonsaiSmallZone, highHumidity, humidityChecker),
				},
			},
			expectedCalculated: true,
			expectedEventsLen:  2,
			expectedEvents: []event.Event{
				&ml.WateringZoneSkippedEvent{},
				&ml.WateringZoneSkippedEvent{},
			},
			expectedReasons: []string{
				ml.AboveMaxThresholdReason,
				ml.AboveMaxThresholdReason,
			},
		},
		{
			name: "with a low humidity in alls zones but is raining, then it returns watering system skipped events",
			args: args{
				zonesHumidity: []*ml.ZoneHumidity{
					ml.NewZoneHumidity(bonsaiBigZone, lowHumidity, humidityChecker),
					ml.NewZoneHumidity(bonsaiSmallZone, lowHumidity, humidityChecker),
				},
				isRaining: true,
			},
			expectedEventsLen:  2,
			expectedCalculated: true,
			expectedEvents: []event.Event{
				&ml.WateringSystemSkippedEvent{},
				&ml.WateringSystemSkippedEvent{},
			},
			expectedReasons: []string{
				ml.RainingReason,
				ml.RainingReason,
			},
		},
		{
			name: "with a low humidity in one zone but is raining, then it returns a watering zone skipped and watering system skipped events",
			args: args{
				zonesHumidity: []*ml.ZoneHumidity{
					ml.NewZoneHumidity(bonsaiBigZone, highHumidity, humidityChecker),
					ml.NewZoneHumidity(bonsaiSmallZone, lowHumidity, humidityChecker),
				},
				isRaining: true,
			},
			expectedEventsLen:  2,
			expectedCalculated: true,
			expectedEvents: []event.Event{
				&ml.WateringZoneSkippedEvent{},
				&ml.WateringSystemSkippedEvent{},
			},
			expectedReasons: []string{
				ml.AboveMaxThresholdReason,
				ml.RainingReason,
			},
		},
		{
			name: "with a low humidity in alls zones but system is deactivated, then it returns watering system skipped events",
			args: args{
				zonesHumidity: []*ml.ZoneHumidity{
					ml.NewZoneHumidity(bonsaiBigZone, lowHumidity, humidityChecker),
					ml.NewZoneHumidity(bonsaiSmallZone, lowHumidity, humidityChecker),
				},
				systemActivated: false,
			},
			expectedEventsLen:  2,
			expectedCalculated: true,
			expectedEvents: []event.Event{
				&ml.WateringSystemSkippedEvent{},
				&ml.WateringSystemSkippedEvent{},
			},
			expectedReasons: []string{
				ml.SystemDisabledReason,
				ml.SystemDisabledReason,
			},
		},
		{
			name: "with a low humidity in alls zones but with wrong zone declared in executions, then it returns error",
			args: args{
				zonesHumidity: []*ml.ZoneHumidity{
					ml.NewZoneHumidity(bonsaiBigZone, lowHumidity, humidityChecker),
					ml.NewZoneHumidity(bonsaiSmallZone, lowHumidity, humidityChecker),
				},
				systemActivated: true,
				exec: map[string]ml.Execution{
					"wrong": ptr.FromPointer(ml.NewExecution("wrong", time.Now())),
				},
			},
		},
		{
			name: "with a low humidity in alls zones but recent executed in all zones, then it returns watering zone skipped events",
			args: args{
				zonesHumidity: []*ml.ZoneHumidity{
					ml.NewZoneHumidity(bonsaiBigZone, lowHumidity, humidityChecker),
					ml.NewZoneHumidity(bonsaiSmallZone, lowHumidity, humidityChecker),
				},
				systemActivated: true,
				exec: map[string]ml.Execution{
					bonsaiBigZone:   ptr.FromPointer(ml.NewExecution(bonsaiBigZone, time.Now().Add(-1*time.Hour))),
					bonsaiSmallZone: ptr.FromPointer(ml.NewExecution(bonsaiSmallZone, time.Now().Add(-1*time.Hour))),
				},
			},
			expectedEventsLen:  2,
			expectedCalculated: true,
			expectedEvents: []event.Event{
				&ml.WateringZoneSkippedEvent{},
				&ml.WateringZoneSkippedEvent{},
			},
			expectedReasons: []string{
				ml.ZoneRecentlyExecutedByModelReason,
				ml.ZoneRecentlyExecutedByModelReason,
			},
		},
		{
			name: "with a low humidity in alls zones but recent executed in one zone, then it returns watering system zone skipped and watering requested events",
			args: args{
				zonesHumidity: []*ml.ZoneHumidity{
					ml.NewZoneHumidity(bonsaiBigZone, lowHumidity, humidityChecker),
					ml.NewZoneHumidity(bonsaiSmallZone, lowHumidity, humidityChecker),
				},
				systemActivated: true,
				exec: map[string]ml.Execution{
					bonsaiBigZone:   ptr.FromPointer(ml.NewExecution(bonsaiBigZone, time.Now().Add(-1*time.Hour))),
					bonsaiSmallZone: ptr.FromPointer(ml.NewExecution(bonsaiSmallZone, time.Now().Add(-5*time.Hour))),
				},
			},
			expectedEventsLen:  2,
			expectedCalculated: true,
			expectedEvents: []event.Event{
				&ml.WateringZoneSkippedEvent{},
				&ml.WateringRequestedEvent{},
			},
			expectedReasons: []string{
				ml.ZoneRecentlyExecutedByModelReason,
				ml.BelowMinThresholdReason,
			},
		},
		{
			name: "with a medium humidity in alls zones, then it returns nil events and calculated false",
			args: args{
				zonesHumidity: []*ml.ZoneHumidity{
					ml.NewZoneHumidity(bonsaiBigZone, mediumHumidity, humidityChecker),
					ml.NewZoneHumidity(bonsaiSmallZone, mediumHumidity, humidityChecker),
				},
				exec: map[string]ml.Execution{
					bonsaiBigZone:   ptr.FromPointer(ml.NewExecution(bonsaiBigZone, time.Now().Add(-5*time.Hour))),
					bonsaiSmallZone: ptr.FromPointer(ml.NewExecution(bonsaiSmallZone, time.Now().Add(-5*time.Hour))),
				},
			},
			expectedCalculated: false,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a CalculatedPrediction struct,
		when the constructor is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ml.NewCalculatedWatering(tt.args.isRaining, tt.args.systemActivated, func() time.Time {
				return time.Now()
			}, tt.args.exec, tt.args.zonesHumidity)
			if err != nil {
				require.ErrorIs(t, err, ml.ErrUnknownZone)
				return
			}
			require.Equal(t, tt.expectedCalculated, got.Calculated())
			if got.Calculated() {
				events := got.Events()
				require.Len(t, events, tt.expectedEventsLen)
				for i, ev := range events {
					require.IsType(t, tt.expectedEvents[i], ev)
					expected := tt.expectedReasons[i]
					var reas string
					switch ev.(type) {
					case *ml.WateringZoneSkippedEvent:
						reas = ev.(*ml.WateringZoneSkippedEvent).Reason
					case *ml.WateringSystemSkippedEvent:
						reas = ev.(*ml.WateringSystemSkippedEvent).Reason
					case *ml.WateringRequestedEvent:
						reas = ev.(*ml.WateringRequestedEvent).Reason
					}
					require.Equal(t, expected, reas)
				}
			}
		})
	}
}

func TestCalculatedWatering_FromPrediction(t *testing.T) {
	type args struct {
		pred     *ml.Prediction
		zh       *ml.ZoneHumidity
		timeFunc func() time.Time
	}
	tests := []struct {
		name           string
		args           args
		expectedEvents event.Event
		expectedReason string
	}{
		{
			name: "when is night, then it returns watering zone skipped event",
			args: args{
				pred: ml.NewPrediction(uuid.New(), bonsaiBigZone, true, 10, "reason test", 0.5),
				zh:   ml.NewZoneHumidity(bonsaiBigZone, mediumHumidity, ml.NewHumidityReference(humidity40, humidity100)),
				timeFunc: func() time.Time {
					return time.Date(2021, 1, 1, 23, 0, 0, 0, time.UTC)
				},
			},
			expectedEvents: &ml.WateringSystemSkippedEvent{},
			expectedReason: ml.IsNightRangeReason,
		},
		{
			name: "and predictions say to watering, then it returns watering requested event",
			args: args{
				pred:     ml.NewPrediction(uuid.New(), bonsaiBigZone, true, 10, "reason test", 0.5),
				zh:       ml.NewZoneHumidity(bonsaiBigZone, mediumHumidity, ml.NewHumidityReference(humidity40, humidity100)),
				timeFunc: dayTimeFunc,
			},
			expectedEvents: &ml.WateringRequestedEvent{},
			expectedReason: ml.ModelPredictionReason,
		},
		{
			name: "and predictions say no watering, then it returns watering requested event",
			args: args{
				pred:     ml.NewPrediction(uuid.New(), bonsaiBigZone, false, 0, "reason test", 0.5),
				zh:       ml.NewZoneHumidity(bonsaiBigZone, mediumHumidity, ml.NewHumidityReference(humidity40, humidity100)),
				timeFunc: dayTimeFunc,
			},
			expectedEvents: &ml.WateringZoneSkippedEvent{},
			expectedReason: ml.ModelNotEstimatedReason,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a built CalculateWatering struct, without events,
		when FromPrediction method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			c, err := ml.NewCalculatedWatering(false, false, tt.args.timeFunc, map[string]ml.Execution{}, []*ml.ZoneHumidity{
				ml.NewZoneHumidity(bonsaiBigZone, mediumHumidity, ml.NewHumidityReference(humidity40, humidity100)),
				ml.NewZoneHumidity(bonsaiSmallZone, mediumHumidity, ml.NewHumidityReference(humidity40, humidity100)),
			})
			require.NoError(t, err)
			require.Len(t, c.Events(), 0)
			require.False(t, c.Calculated())
			c.FromPrediction(tt.args.pred, tt.args.zh)
			events := c.Events()
			require.Len(t, events, 1)
			require.IsType(t, tt.expectedEvents, events[0])
			var reas string
			switch events[0].(type) {
			case *ml.WateringRequestedEvent:
				reas = events[0].(*ml.WateringRequestedEvent).Reason
			case *ml.WateringZoneSkippedEvent:
				reas = events[0].(*ml.WateringZoneSkippedEvent).Reason
			case *ml.WateringSystemSkippedEvent:
				reas = events[0].(*ml.WateringSystemSkippedEvent).Reason
			}
			require.Equal(t, tt.expectedReason, reas)
		})
	}
}
