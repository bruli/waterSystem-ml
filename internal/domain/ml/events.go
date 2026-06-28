package ml

import (
	"github.com/bruli/go-core/event"
	"github.com/google/uuid"
)

const (
	WateringSystemSkippedEventName      = "watering_system_skipped"
	WateringZoneSkippedEventName        = "watering_zone_skipped"
	WateringRequestedEventName          = "watering_requested"
	PredictionValidationFailedEventName = "prediction_validation_failed"

	PredictionPendingValidationReason = "prediction_pending_validation"
)

type PredictionValidationFailedEvent struct {
	event.BasicEvent
	Zone string
}

func NewPredictionValidationFailedEvent(predictionID uuid.UUID, zone string) *PredictionValidationFailedEvent {
	return &PredictionValidationFailedEvent{
		BasicEvent: event.NewBasicEvent(PredictionValidationFailedEventName, uuid.New(), predictionID.String()),
		Zone:       zone,
	}
}

type WateringSystemSkippedEvent struct {
	event.BasicEvent
	Reason string
}

func NewWateringSystemSkippedEvent(reason string) *WateringSystemSkippedEvent {
	return &WateringSystemSkippedEvent{
		BasicEvent: event.NewBasicEvent(WateringSystemSkippedEventName, uuid.New(), uuid.NewString()),
		Reason:     reason,
	}
}

type WateringZoneSkippedEvent struct {
	event.BasicEvent
	Zone           string
	Reason         string
	Moisture       float64
	PredictionID   *uuid.UUID
	DecisionReason *string
	WateringProba  *float64
}

func NewWateringZoneSkippedEvent(
	zone string,
	reason string,
	moisture float64,
	predictionID *uuid.UUID,
	decisionReason *string,
	wateringProba *float64,
) *WateringZoneSkippedEvent {
	return &WateringZoneSkippedEvent{
		BasicEvent:     event.NewBasicEvent(WateringZoneSkippedEventName, uuid.New(), uuid.NewString()),
		Zone:           zone,
		Reason:         reason,
		Moisture:       moisture,
		PredictionID:   predictionID,
		DecisionReason: decisionReason,
		WateringProba:  wateringProba,
	}
}

type WateringRequestedEvent struct {
	event.BasicEvent
	Zone           string
	Reason         string
	Seconds        float64
	MoistureBefore float64
	TargetMoisture float64
	PredictionID   *uuid.UUID
	DecisionReason *string
	WateringProba  *float64
}

func NewWateringRequestedEvent(
	zone string,
	reason string,
	seconds, moistureBefore, targetMoisture float64,
	predictionID *uuid.UUID,
	decisionReason *string,
	wateringProba *float64,
) *WateringRequestedEvent {
	return &WateringRequestedEvent{
		BasicEvent:     event.NewBasicEvent(WateringRequestedEventName, uuid.New(), uuid.NewString()),
		Zone:           zone,
		Reason:         reason,
		Seconds:        seconds,
		MoistureBefore: moistureBefore,
		TargetMoisture: targetMoisture,
		PredictionID:   predictionID,
		DecisionReason: decisionReason,
		WateringProba:  wateringProba,
	}
}
