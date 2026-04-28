package ml_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math/rand"
	"testing"
	"time"

	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"github.com/bruli/watersystem-ml/internal/ptr"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestGetPrediction_Get(t *testing.T) {
	humidity := ml.Humidities
	bb, ok := humidity["Bonsai big"]
	require.True(t, ok)
	bbHigh := randomFloat(bb.V100(), bb.HighHumidity())
	bbLow := randomFloat(bb.LowHumidity(), 2.000)
	bbMedium := randomFloat(bb.V40(), bb.HighHumidity())
	bs, ok := humidity["Bonsai small"]
	require.True(t, ok)
	bsHigh := randomFloat(bs.V100(), bs.HighHumidity())
	bsLow := randomFloat(bs.LowHumidity(), 2.000)
	bsMedium := randomFloat(bs.V40(), bs.HighHumidity())
	errTest := errors.New("test")
	oldExecutions := ml.Executions{
		"Bonsai small": *ml.NewExecution("Bonsai small", time.Now().Add(-24*time.Hour)),
		"Bonsai big":   *ml.NewExecution("Bonsai big", time.Now().Add(-24*time.Hour)),
	}

	type args struct {
		ctx context.Context
	}
	defaultArgs := args{
		ctx: t.Context(),
	}
	defaultTimeFunc := func() time.Time { return time.Date(2021, 1, 1, 10, 0, 0, 0, time.UTC) }
	tests := []struct {
		name        string
		args        args
		expected    []ml.Prediction
		predictions []ml.Prediction
		expectedErr, soilMeasureErr,
		predictionErr, executionRepoErr error
		soilMeasure   []ml.SoilMeasure
		predRepoCalls int
		timeFunc      func() time.Time
		executions    ml.Executions
	}{
		{
			name:           "and soilMeasureRepo returns an error, then it returns a get prediction error",
			args:           defaultArgs,
			timeFunc:       defaultTimeFunc,
			soilMeasureErr: errTest,
			expectedErr:    ml.GetPredictionError{},
		},
		{
			name:     "and executionRepo returns an error, then it returns same error",
			args:     defaultArgs,
			timeFunc: defaultTimeFunc,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai big", bbHigh)),
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai small", bbHigh)),
			},
			executionRepoErr: errTest,
			expectedErr:      ml.GetPredictionError{},
		},
		{
			name:     "and soilMeasureRepo returns soil measures with high humidity, then it returns empty list",
			args:     defaultArgs,
			timeFunc: defaultTimeFunc,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai big", bbHigh)),
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai small", bbHigh)),
			},
			expected: []ml.Prediction{},
		},
		{
			name:     "and soilMeasureRepo returns soil measures with low humidity in one zone, then it returns one prediction",
			args:     defaultArgs,
			timeFunc: defaultTimeFunc,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai big", bbLow)),
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai small", bsHigh)),
			},
			expected: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Bonsai big", true, 20, "Low humidity")),
			},
			executions: oldExecutions,
		},
		{
			name:     "and soilMeasureRepo returns soil measures with low humidity in one zone but this zone is executed, then it returns nil",
			args:     defaultArgs,
			timeFunc: defaultTimeFunc,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai big", bbLow)),
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai small", bsHigh)),
			},
			expected: []ml.Prediction{},
			executions: map[string]ml.Execution{
				"Bonsai big":   ptr.FromPointer(ml.NewExecution("Bonsai big", time.Now().Add(-1*time.Hour))),
				"Bonsai small": ptr.FromPointer(ml.NewExecution("Bonsai small", time.Now().Add(-1*time.Hour))),
			},
		},
		{
			name:     "and soilMeasureRepo returns soil measures with low humidity in all zones, then it returns two predictions",
			args:     defaultArgs,
			timeFunc: defaultTimeFunc,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai big", bbLow)),
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai small", bsLow)),
			},
			expected: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Bonsai big", true, 20, "Low humidity")),
				ptr.FromPointer(ml.NewPrediction("Bonsai small", true, 20, "Low humidity")),
			},
			executions: oldExecutions,
		},
		{
			name:     "and soilMeasureRepo returns soil measures with medium humidity in all zones and prediction repository returns an error, then it returns a get prediction error",
			args:     defaultArgs,
			timeFunc: defaultTimeFunc,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai big", bbMedium)),
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai small", bsMedium)),
			},
			expectedErr:   ml.GetPredictionError{},
			predictionErr: errTest,
			predRepoCalls: 1,
			executions:    oldExecutions,
		},
		{
			name:     "and soilMeasureRepo returns soil measures with medium humidity in all zones in night range, then it returns empty prediction",
			args:     defaultArgs,
			timeFunc: func() time.Time { return time.Date(2021, 1, 1, 23, 0, 0, 0, time.UTC) },
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai big", bbMedium)),
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai small", bsMedium)),
			},
			expected:   []ml.Prediction{},
			executions: oldExecutions,
		},
		{
			name:     "and soilMeasureRepo returns soil measures with medium humidity in all zones and prediction repository returns an zones to water, then it returns two predictions",
			args:     defaultArgs,
			timeFunc: defaultTimeFunc,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai big", bbMedium)),
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai small", bsMedium)),
			},
			predictions: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Bonsai big", true, 32, "Medium humidity level")),
				ptr.FromPointer(ml.NewPrediction("Bonsai small", true, 20, "Medium humidity level")),
			},
			expected: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Bonsai big", true, 32, "Medium humidity level")),
				ptr.FromPointer(ml.NewPrediction("Bonsai small", true, 20, "Medium humidity level")),
			},
			predRepoCalls: 1,
			executions:    oldExecutions,
		},
		{
			name:     "and soilMeasureRepo returns soil measures with medium humidity in one zones and zones was executed, then it returns one prediction",
			args:     defaultArgs,
			timeFunc: defaultTimeFunc,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai big", bbMedium)),
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai small", bsMedium)),
			},
			predictions: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Bonsai big", true, 32, "Medium humidity level")),
				ptr.FromPointer(ml.NewPrediction("Bonsai small", true, 20, "Medium humidity level")),
			},
			expected: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Bonsai big", true, 32, "Medium humidity level")),
			},
			predRepoCalls: 1,
			executions: map[string]ml.Execution{
				"Bonsai big":   ptr.FromPointer(ml.NewExecution("Bonsai big", time.Now().Add(-3*time.Hour))),
				"Bonsai small": ptr.FromPointer(ml.NewExecution("Bonsai small", time.Now().Add(-1*time.Hour))),
			},
		},
		{
			name:     "and soilMeasureRepo returns soil measures with medium humidity only in one zone and prediction repository returns an zones to water, then it returns one prediction",
			args:     defaultArgs,
			timeFunc: defaultTimeFunc,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai big", bbMedium)),
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai small", bsHigh)),
			},
			predictions: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Bonsai big", true, 32, "Medium humidity level")),
				ptr.FromPointer(ml.NewPrediction("Bonsai small", true, 20, "Medium humidity level")),
			},
			expected: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Bonsai big", true, 32, "Medium humidity level")),
			},
			predRepoCalls: 1,
			executions:    oldExecutions,
		},
		{
			name:     "and soilMeasureRepo returns soil measures with medium and low humidity and prediction repository returns an zones to water, then it returns two prediction",
			args:     defaultArgs,
			timeFunc: defaultTimeFunc,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai big", bbMedium)),
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai small", bsLow)),
			},
			predictions: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Bonsai big", true, 32, "Medium humidity level")),
				ptr.FromPointer(ml.NewPrediction("Bonsai small", true, 22, "Medium humidity level")),
			},
			expected: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Bonsai big", true, 32, "Medium humidity level")),
				ptr.FromPointer(ml.NewPrediction("Bonsai small", true, 20, "Low humidity")),
			},
			predRepoCalls: 1,
			executions:    oldExecutions,
		},
		{
			name:     "and soilMeasureRepo returns soil measures with medium humidity and prediction repository returns an zones not water, then it returns empty list",
			args:     defaultArgs,
			timeFunc: defaultTimeFunc,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai big", bbMedium)),
				ptr.FromPointer(ml.NewSoilMeasure("Bonsai small", bsMedium)),
			},
			predictions: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Bonsai big", false, 0, "Enough humidity")),
				ptr.FromPointer(ml.NewPrediction("Bonsai small", false, 0, "Enough humidity")),
			},
			predRepoCalls: 1,
			expected:      []ml.Prediction{},
			executions:    oldExecutions,
		},
	}
	for _, tt := range tests {
		t.Run(`Given a GetPrediction struct,
		when Get method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			predRepo := &ml.PredictionRepositoryMock{}
			predRepo.GetFunc = func(_ context.Context) ([]ml.Prediction, error) {
				return tt.predictions, tt.predictionErr
			}
			soilMeasureRepo := &ml.SoilMeasureRepositoryMock{}
			soilMeasureRepo.GetFunc = func(_ context.Context) ([]ml.SoilMeasure, error) {
				return tt.soilMeasure, tt.soilMeasureErr
			}
			executionRepo := &ml.ExecutionRepositoryMock{}
			executionRepo.GetLastExecutionFunc = func(_ context.Context) (ml.Executions, error) {
				return tt.executions, tt.executionRepoErr
			}
			g := ml.NewGetPrediction(predRepo, soilMeasureRepo, executionRepo, noop.NewTracerProvider().Tracer("test"), testLogger(), tt.timeFunc)
			got, err := g.Get(tt.args.ctx)
			require.Equal(t, tt.predRepoCalls, len(predRepo.GetCalls()), "PredictionRepository.Get should be called once")
			if err != nil {
				require.ErrorAs(t, err, &ml.GetPredictionError{})
				return
			}
			require.Equal(t, tt.expected, got)
		})
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func randomFloat(minimum, maximum float64) float64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return minimum + r.Float64()*(maximum-minimum)
}
