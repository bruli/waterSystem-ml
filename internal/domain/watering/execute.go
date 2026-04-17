package watering

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Executor interface {
	Execute(ctx context.Context, w *Watering) error
}

type Execute struct {
	exec   Executor
	tracer trace.Tracer
}

func (e Execute) Execute(ctx context.Context, w *Watering) error {
	ctx, span := e.tracer.Start(ctx, "Watering.Execute")
	defer span.End()
	if err := e.exec.Execute(ctx, w); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	span.SetStatus(codes.Ok, "OK")
	return nil
}

func NewExecute(exec Executor, tracer trace.Tracer) *Execute {
	return &Execute{exec: exec, tracer: tracer}
}
