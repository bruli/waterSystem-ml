package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/schema"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type PredictionLogRepository struct {
	db     bun.IDB
	tracer trace.Tracer
}

func (p PredictionLogRepository) GetPendingByZone(ctx context.Context, zone string, limit time.Time) (*ml.PredictionLog, error) {
	ctx, span := p.tracer.Start(ctx, "PredictionLogRepository.GetPendingByZone")
	defer span.End()
	model := &modelPrediction{}
	err := p.db.NewSelect().
		Model(model).
		Where("zone = ?", zone).
		Where("validation_at IS NULL").
		Where("validate_after <= ?", limit).
		OrderBy("validate_after", schema.OrderDesc).
		Limit(1).
		Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ml.ErrPredictionLogNotFound
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	pl := ml.PredictionLog{}
	st, err := ml.ParsePredictionLogStatus(model.ValidationStatus)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	if err := pl.Hydrate(
		model.ID,
		model.Zone,
		st,
		model.ShouldWater,
		model.PredictedSeconds,
		model.DecisionReason,
		model.MoistureBefore,
		model.WateringExecuted,
		model.TargetMoisture,
		nil,
		nil,
		model.ValidateAfter,
	); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	return &pl, nil
}

func (p PredictionLogRepository) GetPendingValidationZones(ctx context.Context) (map[string]bool, error) {
	ctx, span := p.tracer.Start(ctx, "PredictionLogRepository.IsPendingValidationByZone")
	defer span.End()
	var predictions []modelPrediction

	err := p.db.NewSelect().
		Model(&predictions).
		Column("zone").
		Where("validation_at IS NULL").
		Scan(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	result := make(map[string]bool, len(predictions))
	for i := range predictions {
		result[predictions[i].Zone] = true
	}

	return result, nil
}

func (p PredictionLogRepository) Save(ctx context.Context, pl *ml.PredictionLog) error {
	ctx, span := p.tracer.Start(ctx, "PredictionLogRepository.Save")
	defer span.End()
	model := buildModel(pl)
	_, err := p.db.NewInsert().Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("validation_at = EXCLUDED.validation_at").
		Set("moisture_after = EXCLUDED.moisture_after").
		Set("reached_target = EXCLUDED.reached_target").
		Set("validation_status = EXCLUDED.validation_status").
		Exec(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	return nil
}

func buildModel(pl *ml.PredictionLog) *modelPrediction {
	return &modelPrediction{
		BaseModel:        bun.BaseModel{},
		ID:               pl.Id(),
		CreatedAt:        pl.CreatedAt(),
		Zone:             pl.Zone(),
		ShouldWater:      pl.ShouldWater(),
		PredictedSeconds: pl.PredictedSeconds(),
		DecisionReason:   pl.DecisionReason(),
		MoistureBefore:   pl.MoistureBefore(),
		WateringExecuted: pl.WateringExecuted(),
		TargetMoisture:   pl.TargetMoisture(),
		ValidationStatus: pl.Status().String(),
		ValidationAt:     pl.ValidationAt(),
		MoistureAfter:    pl.MoistureAfter(),
		ReachedTarget:    pl.ReachedTarget(),
		ValidateAfter:    pl.ValidateAfter(),
	}
}

func NewPredictionLogRepository(db bun.IDB, tracer trace.Tracer) *PredictionLogRepository {
	return &PredictionLogRepository{db: db, tracer: tracer}
}
