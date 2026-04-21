package ml

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type GetPrediction struct {
	predictionRepo  PredictionRepository
	soilMeasureRepo SoilMeasureRepository
	tracer          trace.Tracer
	predictions     map[string]*Prediction
	m               sync.Mutex
	log             *slog.Logger
	timeFunc        func() time.Time
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
	span.SetAttributes(attribute.Int("soil_measures_voltage_count", len(sm)))
	if len(sm) == 0 {
		g.log.WarnContext(ctx, "no soil measures found")
	}
	result := make([]Prediction, 0)
	for _, m := range sm {
		hum, ok := Humidities[m.Zone()]
		if !ok {
			g.log.WarnContext(ctx, "unknown zone", slog.String("zone", m.Zone()))
			continue
		}
		humidity := m.Humidity()
		span.SetAttributes(
			attribute.Float64("minimum_humidity", hum.MinHumidity()),
			attribute.Float64("maximum_humidity", hum.MaxHumidity()),
			attribute.Float64("high_humidity", hum.HighHumidity()),
			attribute.Float64("low_humidity", hum.LowHumidity()),
			attribute.Float64("current_humidity", humidity),
		)
		switch {
		case hum.IsLow(humidity):
			pred := NewPrediction(m.Zone(), true, 20, "Low humidity")
			result = append(result, *pred)
		case hum.IsHigh(humidity):
			continue
		default:
			if g.isNightRange() {
				span.SetStatus(codes.Ok, "night time")
				return result, nil
			}
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

func (g *GetPrediction) isNightRange() bool {
	now := g.timeFunc().Hour()
	return now > 22 || now < 8
}

func NewGetPrediction(
	predictionRepo PredictionRepository,
	soilMeasureRepo SoilMeasureRepository,
	tracer trace.Tracer,
	log *slog.Logger,
	timeFunc func() time.Time,
) *GetPrediction {
	return &GetPrediction{
		predictionRepo:  predictionRepo,
		soilMeasureRepo: soilMeasureRepo,
		tracer:          tracer,
		predictions:     make(map[string]*Prediction),
		m:               sync.Mutex{},
		log:             log,
		timeFunc:        timeFunc,
	}
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
