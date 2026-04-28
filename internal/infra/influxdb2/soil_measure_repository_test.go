//go:build integration

package influxdb2_test

import (
	"testing"

	"github.com/bruli/watersystem-ml/internal/infra/influxdb2"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewSoilMeasureRepository(t *testing.T) {
	token := ""
	repo := influxdb2.NewSoilMeasureRepository("http://localhost:8086", token, "home", "bonsai-data", noop.NewTracerProvider().Tracer("test"))
	_, err := repo.Get(t.Context())
	require.NoError(t, err)
}
