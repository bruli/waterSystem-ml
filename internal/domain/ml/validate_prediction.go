package ml

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ValidatePrediction struct {
	soilMeasureRepo   SoilMeasureRepository
	predictionLogRepo PredictionLogRepository
	tracer            trace.Tracer
}

func (v ValidatePrediction) Validate(ctx context.Context, limit time.Time) error {
	ctx, span := v.tracer.Start(ctx, "ValidatePrediction.Validate")
	defer span.End()
	moisture, err := v.soilMeasureRepo.Get(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	for _, measure := range moisture {
		pl, err := v.predictionLogRepo.GetPendingByZone(ctx, measure.Zone(), limit)
		if err != nil {
			if errors.Is(err, ErrPredictionLogNotFound) {
				continue
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
		pl.AddValidation(new(time.Now()), new(measure.Humidity()))
		if err := v.predictionLogRepo.Save(ctx, pl); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return err
		}
	}
	return nil
}

func NewValidatePrediction(soilMeasureRepo SoilMeasureRepository, predictionLogRepo PredictionLogRepository, tracer trace.Tracer) *ValidatePrediction {
	return &ValidatePrediction{soilMeasureRepo: soilMeasureRepo, predictionLogRepo: predictionLogRepo, tracer: tracer}
}
