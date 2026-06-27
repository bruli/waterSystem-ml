//go:build infra

package postgres_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"github.com/bruli/watersystem-ml/internal/fixtures"
	"github.com/bruli/watersystem-ml/internal/infra/postgres"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestPredictionLogRepository(t *testing.T) {
	t.Run(`Given a PredictionLogRepository`, func(t *testing.T) {
		sqldb, err := sql.Open("postgres", "postgres://userdb:passdb@localhost:5432/watersystem_ml?sslmode=disable")
		require.NoError(t, err)
		db := bun.NewDB(sqldb, pgdialect.New())
		zone := uuid.NewString()
		repo := postgres.NewPredictionLogRepository(db, noop.NewTracerProvider().Tracer("test"))
		t.Run(`when Save method is called,
		then it insert data and returns nil error`, func(t *testing.T) {
			err = repo.Save(t.Context(), new(fixtures.PredictionLogBuilder{Zone: new(zone)}.Build(t)))
			require.NoError(t, err)
		})
		t.Run(`when IsPendingValidationByZone method is called,
		then it should return true`, func(t *testing.T) {
			got, err := repo.IsPendingValidationByZone(t.Context(), zone)
			require.NoError(t, err)
			require.True(t, got)
		})
		t.Run(`when GetPendingByZone method is called `, func(t *testing.T) {
			t.Run(`and does not exists, 
			then it should return a prediction log not found error`, func(t *testing.T) {
				got, err := repo.GetPendingByZone(t.Context(), zone, time.Now().Add(-10*time.Minute))
				require.Nil(t, got)
				require.ErrorIs(t, err, ml.ErrPredictionLogNotFound)
			})
			t.Run(`and exists, 
			then it should return a prediction log`, func(t *testing.T) {
				got, err := repo.GetPendingByZone(t.Context(), zone, time.Now().Add(1*time.Minute))
				require.NoError(t, err)
				require.NotNil(t, got)
			})
		})
	})
}
