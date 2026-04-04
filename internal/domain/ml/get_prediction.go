package ml

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type GetPrediction struct {
	repo   PredictionRepository
	tracer trace.Tracer
}

func (g *GetPrediction) Get(ctx context.Context) ([]Prediction, error) {
	ctx, span := g.tracer.Start(ctx, "GetPrediction")
	defer span.End()
	pred, err := g.repo.Get(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	span.SetStatus(codes.Ok, "OK")
	return pred, nil
}

func NewGetPrediction(repo PredictionRepository, tracer trace.Tracer) *GetPrediction {
	return &GetPrediction{repo: repo, tracer: tracer}
}
