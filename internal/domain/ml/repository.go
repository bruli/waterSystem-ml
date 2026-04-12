package ml

import "context"

//go:generate go tool moq -out repositories_mock.go . PredictionRepository SoilMeasureRepository
type PredictionRepository interface {
	Get(ctx context.Context) ([]Prediction, error)
}

type SoilMeasureRepository interface {
	Get(ctx context.Context) ([]SoilMeasure, error)
}
