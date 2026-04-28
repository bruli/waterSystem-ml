//go:build integration

package influxdb2_test

import (
	"testing"

	"github.com/bruli/watersystem-ml/internal/infra/influxdb2"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestExecutionRepository_GetLastExecution(t *testing.T) {
	token := ""
	repo := influxdb2.NewExecutionRepository("http://localhost:8086", token, "home", "bonsai-data", noop.NewTracerProvider().Tracer("test"))
	got, err := repo.GetLastExecution(t.Context())
	require.NoError(t, err)
	require.Len(t, got, 2)
}
