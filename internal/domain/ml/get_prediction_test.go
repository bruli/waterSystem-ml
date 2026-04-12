package ml_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"github.com/bruli/watersystem-ml/internal/ptr"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestGetPrediction_Get(t *testing.T) {
	errTest := errors.New("test")
	type args struct {
		ctx context.Context
	}
	defaultArgs := args{
		ctx: t.Context(),
	}
	tests := []struct {
		name        string
		args        args
		expected    []ml.Prediction
		predictions []ml.Prediction
		expectedErr, soilMeasureErr,
		predictionErr error
		soilMeasure   []ml.SoilMeasure
		predRepoCalls int
	}{
		{
			name:           "and soilMeasureRepo returns an error, then it returns a get prediction error",
			args:           defaultArgs,
			soilMeasureErr: errTest,
			expectedErr:    ml.GetPredictionError{},
		},
		{
			name: "and soilMeasureRepo returns soil measures with high humidity, then it returns empty list",
			args: defaultArgs,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Zone small", 100)),
				ptr.FromPointer(ml.NewSoilMeasure("Zone small", 100)),
			},
			expected: []ml.Prediction{},
		},
		{
			name: "and soilMeasureRepo returns soil measures with low humidity in one zone, then it returns one prediction",
			args: defaultArgs,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Zone small", 100)),
				ptr.FromPointer(ml.NewSoilMeasure("Zone small", 30)),
			},
			expected: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Zone small", true, 20, "Low humidity")),
			},
		},
		{
			name: "and soilMeasureRepo returns soil measures with low humidity in all zones, then it returns two predictions",
			args: defaultArgs,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Zone big", 30)),
				ptr.FromPointer(ml.NewSoilMeasure("Zone small", 30)),
			},
			expected: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Zone big", true, 20, "Low humidity")),
				ptr.FromPointer(ml.NewPrediction("Zone small", true, 20, "Low humidity")),
			},
		},
		{
			name: "and soilMeasureRepo returns soil measures with medium humidity in all zones and prediction repository returns an error, then it returns a get prediction error",
			args: defaultArgs,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Zone big", 50)),
				ptr.FromPointer(ml.NewSoilMeasure("Zone small", 50)),
			},
			expectedErr:   ml.GetPredictionError{},
			predictionErr: errTest,
			predRepoCalls: 1,
		},
		{
			name: "and soilMeasureRepo returns soil measures with medium humidity in all zones and prediction repository returns an zones to water, then it returns two predictions",
			args: defaultArgs,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Zone big", 50)),
				ptr.FromPointer(ml.NewSoilMeasure("Zone small", 50)),
			},
			predictions: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Zone big", true, 32, "Medium humidity level")),
				ptr.FromPointer(ml.NewPrediction("Zone small", true, 20, "Medium humidity level")),
			},
			expected: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Zone big", true, 32, "Medium humidity level")),
				ptr.FromPointer(ml.NewPrediction("Zone small", true, 20, "Medium humidity level")),
			},
			predRepoCalls: 1,
		},
		{
			name: "and soilMeasureRepo returns soil measures with medium humidity only in one zone and prediction repository returns an zones to water, then it returns one prediction",
			args: defaultArgs,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Zone big", 50)),
				ptr.FromPointer(ml.NewSoilMeasure("Zone small", 80)),
			},
			predictions: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Zone big", true, 32, "Medium humidity level")),
				ptr.FromPointer(ml.NewPrediction("Zone small", true, 20, "Medium humidity level")),
			},
			expected: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Zone big", true, 32, "Medium humidity level")),
			},
			predRepoCalls: 1,
		},
		{
			name: "and soilMeasureRepo returns soil measures with medium and low humidity and prediction repository returns an zones to water, then it returns two prediction",
			args: defaultArgs,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Zone big", 50)),
				ptr.FromPointer(ml.NewSoilMeasure("Zone small", 30)),
			},
			predictions: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Zone big", true, 32, "Medium humidity level")),
				ptr.FromPointer(ml.NewPrediction("Zone small", true, 22, "Medium humidity level")),
			},
			expected: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Zone big", true, 32, "Medium humidity level")),
				ptr.FromPointer(ml.NewPrediction("Zone small", true, 20, "Low humidity")),
			},
			predRepoCalls: 1,
		},
		{
			name: "and soilMeasureRepo returns soil measures with medium humidity and prediction repository returns an zones not water, then it returns empty list",
			args: defaultArgs,
			soilMeasure: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure("Zone big", 50)),
				ptr.FromPointer(ml.NewSoilMeasure("Zone small", 50)),
			},
			predictions: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction("Zone big", false, 0, "Enough humidity")),
				ptr.FromPointer(ml.NewPrediction("Zone small", false, 0, "Enough humidity")),
			},
			predRepoCalls: 1,
			expected:      []ml.Prediction{},
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
			g := ml.NewGetPrediction(predRepo, soilMeasureRepo, noop.NewTracerProvider().Tracer("test"))
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
