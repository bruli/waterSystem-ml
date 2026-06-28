package ml

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type CheckModel struct {
	repo   ModelHealthRepository
	tracer trace.Tracer
}

func (c CheckModel) Check(ctx context.Context, zone string) (*ModelHealth, error) {
	ctx, span := c.tracer.Start(ctx, "CheckModel.Check")
	defer span.End()
	modelHealth, err := c.repo.GetModelHealth(ctx, zone)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	modelHealth.Check()
	return modelHealth, nil
}

func NewCheckModel(repo ModelHealthRepository, tracer trace.Tracer) *CheckModel {
	return &CheckModel{repo: repo, tracer: tracer}
}
