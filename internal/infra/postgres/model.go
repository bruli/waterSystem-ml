package postgres

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type modelPrediction struct {
	bun.BaseModel `bun:"table:model_predictions"`

	ID               uuid.UUID `bun:",pk"`
	CreatedAt        time.Time
	Zone             string
	ShouldWater      bool
	PredictedSeconds float64
	DecisionReason   string
	MoistureBefore   float64
	WateringExecuted bool
	TargetMoisture   float64
	ValidationStatus string
	ValidationAt     *time.Time
	MoistureAfter    *float64
	ReachedTarget    *bool
	ValidateAfter    time.Time
}

type resultPrediction struct {
	SuccessfulPredictions int `bun:"successful_predictions"`
	FailedPredictions     int `bun:"failed_predictions"`
}
