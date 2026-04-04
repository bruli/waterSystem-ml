package ml

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Train struct {
	executor TrainExecutor
	tracer   trace.Tracer
}

func (t *Train) Run(ctx context.Context) error {
	ctx, span := t.tracer.Start(ctx, "Train")
	defer span.End()
	if err := t.executor.Run(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "OK")
	return nil
}

func NewTrain(executor TrainExecutor, tracer trace.Tracer) *Train {
	return &Train{executor: executor, tracer: tracer}
}
