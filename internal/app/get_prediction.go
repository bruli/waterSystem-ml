package app

import (
	"context"
	"fmt"

	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type GetPrediction struct {
	svc    *ml.GetPrediction
	pub    Publisher
	tracer trace.Tracer
}

func (p GetPrediction) Get(ctx context.Context) ([]ml.Prediction, error) {
	ctx, span := p.tracer.Start(ctx, "app.GetPrediction")
	defer span.End()
	predictions, err := p.svc.Get(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	for _, prediction := range predictions {
		if prediction.ShouldWater() {
			msg := fmt.Sprintf(
				"ML Prediction\n Watering zone: %s\n seconds: %v\n reason: %s\n",
				prediction.Zone(),
				prediction.PredictedSeconds(),
				prediction.DecisionReason(),
			)
			if err = p.pub.Publish(ctx, msg); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, err
			}
		}
	}

	span.SetStatus(codes.Ok, "OK")
	return predictions, nil
}

func NewGetPrediction(svc *ml.GetPrediction, pub Publisher, tracer trace.Tracer) *GetPrediction {
	return &GetPrediction{svc: svc, pub: pub, tracer: tracer}
}
