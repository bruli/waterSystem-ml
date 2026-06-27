package ml

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	PredictionLogStatusSuccess PredictionLogStatus = "success"
	PredictionLogStatusFailed  PredictionLogStatus = "failed"
	PredictionLogStatusPending PredictionLogStatus = "pending"
)

var (
	ErrInvalidPredictionLogID             = errors.New("invalid prediction log id")
	ErrInvalidPredictionLogZone           = errors.New("invalid prediction log zone")
	ErrInvalidPredictionLogDecisionReason = errors.New("invalid prediction log decision reason")
	ErrInvalidPredictionLogMoistureBefore = errors.New("invalid prediction log moisture before")
	ErrInvalidPredictionLogTargetMoisture = errors.New("invalid prediction log target moisture")
	ErrPredictionLogNotFound              = errors.New("prediction log not found")
)

type PredictionLogStatus string

func (s PredictionLogStatus) String() string {
	return string(s)
}

func ParsePredictionLogStatus(s string) (PredictionLogStatus, error) {
	switch s {
	case PredictionLogStatusSuccess.String():
		return PredictionLogStatusSuccess, nil
	case PredictionLogStatusFailed.String():
		return PredictionLogStatusFailed, nil
	case PredictionLogStatusPending.String():
		return PredictionLogStatusPending, nil
	default:
		return "", errors.New("invalid prediction log status")
	}
}

type PredictionLog struct {
	id               uuid.UUID
	createdAt        time.Time
	zone             string
	shouldWater      bool
	predictedSeconds float64
	decisionReason   string
	moistureBefore   float64
	wateringExecuted bool
	status           PredictionLogStatus
	targetMoisture   float64
	validationAt     *time.Time
	moistureAfter    *float64
}

func (l *PredictionLog) ValidationAt() *time.Time {
	return l.validationAt
}

func (l *PredictionLog) MoistureAfter() *float64 {
	return l.moistureAfter
}

func (l *PredictionLog) ReachedTarget() *bool {
	if l.moistureAfter == nil {
		return nil
	}
	switch {
	case *l.moistureAfter >= l.targetMoisture:
		return new(true)
	default:
		return new(false)
	}
}

func (l *PredictionLog) Id() uuid.UUID {
	return l.id
}

func (l *PredictionLog) CreatedAt() time.Time {
	return l.createdAt
}

func (l *PredictionLog) Zone() string {
	return l.zone
}

func (l *PredictionLog) ShouldWater() bool {
	return l.shouldWater
}

func (l *PredictionLog) PredictedSeconds() float64 {
	return l.predictedSeconds
}

func (l *PredictionLog) DecisionReason() string {
	return l.decisionReason
}

func (l *PredictionLog) MoistureBefore() float64 {
	return l.moistureBefore
}

func (l *PredictionLog) WateringExecuted() bool {
	return l.wateringExecuted
}

func (l *PredictionLog) Status() PredictionLogStatus {
	return l.status
}

func (l *PredictionLog) TargetMoisture() float64 {
	return l.targetMoisture
}

func (l *PredictionLog) validate() error {
	switch {
	case l.id == uuid.Nil:
		return ErrInvalidPredictionLogID
	case l.zone == "":
		return ErrInvalidPredictionLogZone
	case l.decisionReason == "":
		return ErrInvalidPredictionLogDecisionReason
	case l.moistureBefore == 0:
		return ErrInvalidPredictionLogMoistureBefore
	case l.targetMoisture == 0:
		return ErrInvalidPredictionLogTargetMoisture
	default:
	}
	return nil
}

func (l *PredictionLog) Hydrate(
	id uuid.UUID,
	zone string,
	status PredictionLogStatus,
	shouldWater bool,
	predictedSeconds float64,
	decisionReason string,
	moistureBefore float64,
	wateringExecuted bool,
	targetMoisture float64,
	validationAt *time.Time,
	moistureAfter *float64,
) error {
	l.id = id
	l.zone = zone
	l.shouldWater = shouldWater
	l.predictedSeconds = predictedSeconds
	l.decisionReason = decisionReason
	l.moistureBefore = moistureBefore
	l.wateringExecuted = wateringExecuted
	l.targetMoisture = targetMoisture
	l.validationAt = validationAt
	l.moistureAfter = moistureAfter
	l.status = status
	return l.validate()
}

func (l *PredictionLog) AddValidation(at *time.Time, moistureAfter *float64) {
	l.validationAt = at
	l.moistureAfter = moistureAfter
	if reachedTarget := l.ReachedTarget(); reachedTarget != nil {
		switch {
		case *reachedTarget:
			l.status = PredictionLogStatusSuccess
		default:
			l.status = PredictionLogStatusFailed
		}
	}
}

func NewPredictionLog(
	id uuid.UUID,
	zone string,
	shouldWater bool,
	predictedSeconds float64,
	decisionReason string,
	moistureBefore float64,
	wateringExecuted bool,
	targetMoisture float64,
) (*PredictionLog, error) {
	pl := PredictionLog{
		id:               id,
		zone:             zone,
		shouldWater:      shouldWater,
		predictedSeconds: predictedSeconds,
		decisionReason:   decisionReason,
		moistureBefore:   moistureBefore,
		wateringExecuted: wateringExecuted,
		targetMoisture:   targetMoisture,
		createdAt:        time.Now(),
		status:           PredictionLogStatusPending,
	}

	if err := pl.validate(); err != nil {
		return nil, err
	}
	return &pl, nil
}
