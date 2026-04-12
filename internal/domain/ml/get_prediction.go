package ml

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	LowHumidity    = 40
	MediumHumidity = 65
)

type GetPrediction struct {
	predictionRepo  PredictionRepository
	soilMeasureRepo SoilMeasureRepository
	tracer          trace.Tracer
	predictions     map[string]*Prediction
	m               sync.Mutex
}

func (g *GetPrediction) Get(ctx context.Context) ([]Prediction, error) {
	ctx, span := g.tracer.Start(ctx, "GetPrediction")
	defer span.End()
	defer clear(g.predictions)

	sm, err := g.soilMeasureRepo.Get(ctx)
	if err != nil {
		err := GetPredictionError{msg: "error getting soil measures", err: err}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	result := make([]Prediction, 0)
	for _, m := range sm {
		switch {
		case m.Humidity() < LowHumidity:
			pred := NewPrediction(m.Zone(), true, 20, "Low humidity")
			result = append(result, *pred)
		case m.Humidity() < MediumHumidity:
			pred, err := g.getPrediction(ctx, m.Zone(), span)
			if err != nil {
				err := GetPredictionError{msg: "error getting prediction", err: err}
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
				return nil, err
			}
			if pred != nil {
				result = append(result, *pred)
			}
		default:
			continue
		}
	}

	span.SetStatus(codes.Ok, "OK")
	return result, nil
}

func (g *GetPrediction) getPrediction(ctx context.Context, zone string, span trace.Span) (*Prediction, error) {
	g.m.Lock()
	defer g.m.Unlock()
	savedPred, ok := g.predictions[zone]
	if ok {
		switch {
		case savedPred.ShouldWater():
			return savedPred, nil
		default:
			return nil, nil
		}
	}
	pred, err := g.predictionRepo.Get(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	for _, p := range pred {
		g.predictions[p.Zone()] = &p
	}
	predZone, ok := g.predictions[zone]
	if ok && predZone.ShouldWater() {
		return predZone, nil
	}
	return nil, nil
}

func NewGetPrediction(predictionRepo PredictionRepository, soilMeasureRepo SoilMeasureRepository, tracer trace.Tracer) *GetPrediction {
	return &GetPrediction{predictionRepo: predictionRepo, tracer: tracer, soilMeasureRepo: soilMeasureRepo, predictions: make(map[string]*Prediction)}
}

type GetPredictionError struct {
	msg string
	err error
}

func (g GetPredictionError) Error() string {
	return g.msg
}

func (g GetPredictionError) Unwrap() error {
	return g.err
}
