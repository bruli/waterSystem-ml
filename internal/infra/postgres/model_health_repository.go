package postgres

import (
	"context"
	"fmt"

	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ModelHealthRepository struct {
	db     bun.IDB
	tracer trace.Tracer
}

func (m ModelHealthRepository) GetModelHealth(ctx context.Context, zone string) (*ml.ModelHealth, error) {
	ctx, span := m.tracer.Start(ctx, "ModelHealthRepository.GetModelHealth")
	defer span.End()
	var result resultPrediction
	err := m.db.NewSelect().
		With("latest",
			m.db.NewSelect().
				Model((*modelPrediction)(nil)).
				Column("reached_target").
				Where("zone = ?", zone).
				Where("validation_at IS NOT NULL").
				Where("watering_executed = TRUE").
				Where("reached_target IS NOT NULL").
				OrderExpr("validation_at DESC").
				Limit(20),
		).
		TableExpr("latest").
		ColumnExpr("COUNT(*) FILTER (WHERE reached_target = TRUE) AS successful_predictions").
		ColumnExpr("COUNT(*) FILTER (WHERE reached_target = FALSE) AS failed_predictions").
		Scan(ctx, &result)
	if err != nil {
		err := fmt.Errorf("get model health for zone %s: %w", zone, err)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, err
	}
	return ml.NewModelHealth(zone, result.SuccessfulPredictions, result.FailedPredictions), err
}

func NewModelHealthRepository(db bun.IDB, tracer trace.Tracer) *ModelHealthRepository {
	return &ModelHealthRepository{db: db, tracer: tracer}
}
