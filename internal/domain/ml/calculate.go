package ml

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Calculate struct {
	predictionRepo    PredictionRepository
	soilMeasureRepo   SoilMeasureRepository
	humidityRefRepo   HumidityReferenceRepository
	predictionLogRepo PredictionLogRepository
	executionRepo     ExecutionRepository
	statusRepo        StatusRepository
	tracer            trace.Tracer
	timeFunc          func() time.Time
}

func (c *Calculate) Do(ctx context.Context) (*CalculatedWatering, error) {
	ctx, span := c.tracer.Start(ctx, "Calculate.Do")
	defer span.End()

	st, err := c.statusRepo.GetStatus(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	exec, err := c.executionRepo.GetLastExecution(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	measures, err := c.soilMeasureRepo.Get(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	pendingPredictionLogs, err := c.predictionLogRepo.GetPendingValidationZones(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	zoneHumMap := make(map[string]*ZoneHumidity, len(measures))
	zoneHum := make([]*ZoneHumidity, 0, len(measures))
	for _, measure := range measures {
		ref, err := c.humidityRefRepo.GetByZone(ctx, measure.Zone())
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		hum := NewZoneHumidity(measure.Zone(), measure.Humidity(), ref)
		zoneHum = append(zoneHum, hum)
		zoneHumMap[measure.Zone()] = hum
	}

	cw, err := NewCalculatedWatering(st.Raining(), st.Active(), c.timeFunc, exec, zoneHum, pendingPredictionLogs)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if cw.Calculated() {
		return cw, nil
	}

	predictions, err := c.predictionRepo.Get(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	for _, p := range predictions {
		zh, ok := zoneHumMap[p.Zone()]
		if !ok {
			continue
		}
		cw.FromPrediction(&p, zh)
	}

	return cw, nil
}

func NewCalculate(
	predictionRepo PredictionRepository,
	soilMeasureRepo SoilMeasureRepository,
	humidityRefRepo HumidityReferenceRepository,
	executionRepo ExecutionRepository,
	predictionLogRepo PredictionLogRepository,
	statusRepo StatusRepository,
	tracer trace.Tracer,
	timeFunc func() time.Time,
) *Calculate {
	return &Calculate{
		predictionRepo:    predictionRepo,
		soilMeasureRepo:   soilMeasureRepo,
		humidityRefRepo:   humidityRefRepo,
		executionRepo:     executionRepo,
		statusRepo:        statusRepo,
		tracer:            tracer,
		timeFunc:          timeFunc,
		predictionLogRepo: predictionLogRepo,
	}
}
