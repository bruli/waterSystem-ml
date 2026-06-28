package ml

import (
	"context"
	"time"
)

//go:generate go tool moq -out repositories_mock.go . PredictionRepository SoilMeasureRepository ExecutionRepository HumidityReferenceRepository StatusRepository PredictionLogRepository
type PredictionRepository interface {
	Get(ctx context.Context) ([]Prediction, error)
}

type SoilMeasureRepository interface {
	Get(ctx context.Context) ([]SoilMeasure, error)
}

type ExecutionRepository interface {
	GetLastExecution(ctx context.Context) (Executions, error)
}

type HumidityReferenceRepository interface {
	GetByZone(ctx context.Context, zone string) (*HumidityReference, error)
}

type StatusRepository interface {
	GetStatus(ctx context.Context) (*Status, error)
}

type WateringSkippedLogRepository interface {
	Save(ctx context.Context, skp *WateringSkippedLog) error
}

type PredictionLogRepository interface {
	Save(ctx context.Context, pl *PredictionLog) error
	GetPendingByZone(ctx context.Context, zone string, limit time.Time) (*PredictionLog, error)
	GetPendingValidationZones(ctx context.Context) (map[string]bool, error)
}

type ModelHealthRepository interface {
	GetModelHealth(ctx context.Context, zone string) (*ModelHealth, error)
}
