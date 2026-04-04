package ml

import "context"

type PredictionRepository interface {
	Get(ctx context.Context) ([]Prediction, error)
}
