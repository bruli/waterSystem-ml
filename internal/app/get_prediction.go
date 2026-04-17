package app

import (
	"context"
	"fmt"

	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"github.com/bruli/watersystem-ml/internal/domain/watering"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type GetPrediction struct {
	predictionSvc *ml.GetPrediction
	pub           Publisher
	tracer        trace.Tracer
	executeSvc    *watering.Execute
}

func (p GetPrediction) Get(ctx context.Context) ([]ml.Prediction, error) {
	ctx, span := p.tracer.Start(ctx, "app.GetPrediction")
	defer span.End()
	predictions, err := p.predictionSvc.Get(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	for _, prediction := range predictions {
		if prediction.ShouldWater() {
			i, err := p.publishMessage(ctx, prediction, span)
			if err != nil {
				return i, err
			}
			if err := p.executeSvc.Execute(ctx, watering.New(prediction.Zone(), int(prediction.PredictedSeconds()))); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return i, err
			}
		}
	}

	span.SetStatus(codes.Ok, "OK")
	return predictions, nil
}

func (p GetPrediction) publishMessage(ctx context.Context, prediction ml.Prediction, span trace.Span) ([]ml.Prediction, error) {
	msg := fmt.Sprintf(
		"ML Prediction\n Watering zone: %s\n seconds: %v\n reason: %s\n",
		prediction.Zone(),
		prediction.PredictedSeconds(),
		prediction.DecisionReason(),
	)
	if err := p.pub.Publish(ctx, msg); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return nil, nil
}

func NewGetPrediction(svc *ml.GetPrediction, pub Publisher, tracer trace.Tracer, executeSvc *watering.Execute) *GetPrediction {
	return &GetPrediction{predictionSvc: svc, pub: pub, tracer: tracer, executeSvc: executeSvc}
}
