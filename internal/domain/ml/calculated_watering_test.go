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

var nightTimeFunc = func() time.Time {
	return time.Date(2021, 1, 1, 23, 0, 0, 0, time.UTC)
}

func oldExecution(zone string) ml.Execution {
	return ptr.FromPointer(ml.NewExecution(zone, time.Now().Add(-5*time.Hour)))
}

func recentExecution(zone string) ml.Execution {
	return ptr.FromPointer(ml.NewExecution(zone, time.Now().Add(-1*time.Hour)))
}

func executionsForAllZones() ml.Executions {
	return ml.Executions{
		bonsaiBigZone:   oldExecution(bonsaiBigZone),
		bonsaiSmallZone: oldExecution(bonsaiSmallZone),
	}
}

func humidityRef() *ml.HumidityReference {
	return ml.NewHumidityReference(humidity40, humidity100)
}

func zoneHumidity(zone string, humidity float64) *ml.ZoneHumidity {
	return ml.NewZoneHumidity(zone, humidity, humidityRef())
}

func reasonFromEvent(t *testing.T, ev event.Event) string {
	t.Helper()

	switch e := ev.(type) {
	case *ml.WateringRequestedEvent:
		return e.Reason
	case *ml.WateringZoneSkippedEvent:
		return e.Reason
	case *ml.WateringSystemSkippedEvent:
		return e.Reason
	default:
		t.Fatalf("unexpected event type: %T", ev)
		return ""
	}
}

func requireEvent(t *testing.T, ev, expectedEvent event.Event, expectedReason string) {
	t.Helper()

	require.IsType(t, expectedEvent, ev)
	require.Equal(t, expectedReason, reasonFromEvent(t, ev))
}

func TestNewCalculatedWatering(t *testing.T) {
	tests := []struct {
		name                       string
		isRaining                  bool
		systemActivated            bool
		timeFunc                   func() time.Time
		exec                       ml.Executions
		zonesHumidity              []*ml.ZoneHumidity
		pendingPredictionLogsZones map[string]bool
		expectedCalculated         bool
		expectedPendingZones       []string
		expectedEvents             []event.Event
		expectedReasons            []string
		expectedErr                error
	}{
		{
			name:            "raining skips the whole system before checking zones",
			isRaining:       true,
			systemActivated: true,
			timeFunc:        dayTimeFunc,
			exec:            executionsForAllZones(),
			zonesHumidity: []*ml.ZoneHumidity{
				zoneHumidity(bonsaiBigZone, lowHumidity),
				zoneHumidity(bonsaiSmallZone, mediumHumidity),
			},
			expectedCalculated: true,
			expectedEvents: []event.Event{
				&ml.WateringSystemSkippedEvent{},
			},
			expectedReasons: []string{
				ml.RainingReason,
			},
		},
		{
			name:            "disabled system skips the whole system before checking zones",
			systemActivated: false,
			timeFunc:        dayTimeFunc,
			exec:            executionsForAllZones(),
			zonesHumidity: []*ml.ZoneHumidity{
				zoneHumidity(bonsaiBigZone, lowHumidity),
				zoneHumidity(bonsaiSmallZone, mediumHumidity),
			},
			expectedCalculated: true,
			expectedEvents: []event.Event{
				&ml.WateringSystemSkippedEvent{},
			},
			expectedReasons: []string{
				ml.SystemDisabledReason,
			},
		},
		{
			name:            "night skips the whole system before checking zones",
			systemActivated: true,
			timeFunc:        nightTimeFunc,
			exec:            executionsForAllZones(),
			zonesHumidity: []*ml.ZoneHumidity{
				zoneHumidity(bonsaiBigZone, lowHumidity),
				zoneHumidity(bonsaiSmallZone, mediumHumidity),
			},
			expectedCalculated: true,
			expectedEvents: []event.Event{
				&ml.WateringSystemSkippedEvent{},
			},
			expectedReasons: []string{
				ml.IsNightRangeReason,
			},
		},
		{
			name:            "pending validation skips only that zone",
			systemActivated: true,
			timeFunc:        dayTimeFunc,
			exec:            executionsForAllZones(),
			pendingPredictionLogsZones: map[string]bool{
				bonsaiBigZone: true,
			},
			zonesHumidity: []*ml.ZoneHumidity{
				zoneHumidity(bonsaiBigZone, mediumHumidity),
				zoneHumidity(bonsaiSmallZone, mediumHumidity),
			},
			expectedCalculated:   false,
			expectedPendingZones: []string{bonsaiSmallZone},
			expectedEvents: []event.Event{
				&ml.WateringZoneSkippedEvent{},
			},
			expectedReasons: []string{
				ml.PredictionPendingValidationReason,
			},
		},
		{
			name:            "high humidity skips the zone and low humidity requests watering",
			systemActivated: true,
			timeFunc:        dayTimeFunc,
			exec:            executionsForAllZones(),
			zonesHumidity: []*ml.ZoneHumidity{
				zoneHumidity(bonsaiBigZone, highHumidity),
				zoneHumidity(bonsaiSmallZone, lowHumidity),
			},
			expectedCalculated: true,
			expectedEvents: []event.Event{
				&ml.WateringZoneSkippedEvent{},
				&ml.WateringRequestedEvent{},
			},
			expectedReasons: []string{
				ml.AboveMaxThresholdReason,
				ml.BelowMinThresholdReason,
			},
		},
		{
			name:            "recent execution skips only that zone and low humidity requests watering in the other zone",
			systemActivated: true,
			timeFunc:        dayTimeFunc,
			exec: ml.Executions{
				bonsaiBigZone:   recentExecution(bonsaiBigZone),
				bonsaiSmallZone: oldExecution(bonsaiSmallZone),
			},
			zonesHumidity: []*ml.ZoneHumidity{
				zoneHumidity(bonsaiBigZone, lowHumidity),
				zoneHumidity(bonsaiSmallZone, lowHumidity),
			},
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
			name:            "medium humidity remains pending for prediction",
			systemActivated: true,
			timeFunc:        dayTimeFunc,
			exec:            executionsForAllZones(),
			zonesHumidity: []*ml.ZoneHumidity{
				zoneHumidity(bonsaiBigZone, mediumHumidity),
				zoneHumidity(bonsaiSmallZone, mediumHumidity),
			},
			expectedCalculated:   false,
			expectedPendingZones: []string{bonsaiBigZone, bonsaiSmallZone},
		},
		{
			name:            "recent execution is checked even when humidity is medium",
			systemActivated: true,
			timeFunc:        dayTimeFunc,
			exec: ml.Executions{
				bonsaiBigZone:   recentExecution(bonsaiBigZone),
				bonsaiSmallZone: oldExecution(bonsaiSmallZone),
			},
			zonesHumidity: []*ml.ZoneHumidity{
				zoneHumidity(bonsaiBigZone, mediumHumidity),
				zoneHumidity(bonsaiSmallZone, mediumHumidity),
			},
			expectedCalculated:   false,
			expectedPendingZones: []string{bonsaiSmallZone},
			expectedEvents: []event.Event{
				&ml.WateringZoneSkippedEvent{},
			},
			expectedReasons: []string{
				ml.ZoneRecentlyExecutedByModelReason,
			},
		},
		{
			name:            "unknown execution zone returns error",
			systemActivated: true,
			timeFunc:        dayTimeFunc,
			exec: ml.Executions{
				"wrong": oldExecution("wrong"),
			},
			zonesHumidity: []*ml.ZoneHumidity{
				zoneHumidity(bonsaiBigZone, lowHumidity),
			},
			expectedErr: ml.ErrUnknownZone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ml.NewCalculatedWatering(
				tt.isRaining,
				tt.systemActivated,
				tt.timeFunc,
				tt.exec,
				tt.zonesHumidity,
				tt.pendingPredictionLogsZones,
			)

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.expectedCalculated, got.Calculated())
			require.ElementsMatch(t, tt.expectedPendingZones, got.PendingPredictionZones())

			events := got.Events()
			require.Len(t, events, len(tt.expectedEvents))
			for i, ev := range events {
				requireEvent(t, ev, tt.expectedEvents[i], tt.expectedReasons[i])
			}
		})
	}
}

func TestCalculatedWatering_FromPrediction(t *testing.T) {
	tests := []struct {
		name               string
		pred               *ml.Prediction
		zh                 *ml.ZoneHumidity
		expectedCalculated bool
		expectedEvents     []event.Event
		expectedReasons    []string
	}{
		{
			name: "prediction says watering",
			pred: ml.NewPrediction(uuid.New(), bonsaiBigZone, true, 10, "reason test", 0.5),
			zh:   zoneHumidity(bonsaiBigZone, mediumHumidity),
			expectedEvents: []event.Event{
				&ml.WateringRequestedEvent{},
			},
			expectedReasons: []string{
				ml.ModelPredictionReason,
			},
		},
		{
			name: "prediction says no watering",
			pred: ml.NewPrediction(uuid.New(), bonsaiBigZone, false, 0, "reason test", 0.5),
			zh:   zoneHumidity(bonsaiBigZone, mediumHumidity),
			expectedEvents: []event.Event{
				&ml.WateringZoneSkippedEvent{},
			},
			expectedReasons: []string{
				ml.ModelNotEstimatedReason,
			},
		},
		{
			name: "nil prediction does nothing",
			pred: nil,
			zh:   zoneHumidity(bonsaiBigZone, mediumHumidity),
		},
		{
			name: "nil zone humidity does nothing",
			pred: ml.NewPrediction(uuid.New(), bonsaiBigZone, true, 10, "reason test", 0.5),
			zh:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cw, err := ml.NewCalculatedWatering(
				false,
				true,
				dayTimeFunc,
				executionsForAllZones(),
				[]*ml.ZoneHumidity{
					zoneHumidity(bonsaiBigZone, mediumHumidity),
					zoneHumidity(bonsaiSmallZone, mediumHumidity),
				},
				map[string]bool{},
			)
			require.NoError(t, err)
			require.False(t, cw.Calculated())

			cw.FromPrediction(tt.pred, tt.zh)

			events := cw.Events()
			require.Len(t, events, len(tt.expectedEvents))
			for i, ev := range events {
				requireEvent(t, ev, tt.expectedEvents[i], tt.expectedReasons[i])
			}
		})
	}
}
