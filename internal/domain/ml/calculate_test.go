package ml_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bruli/go-core/ptr"
	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestCalculate_Do(t *testing.T) {
	errTest := errors.New("error")
	defaultStatus := ml.NewStatus(true, false)
	defaultExecutions := ml.Executions{
		bonsaiBigZone:   ptr.FromPointer(ml.NewExecution(bonsaiBigZone, time.Now().Add(-5*time.Hour))),
		bonsaiSmallZone: ptr.FromPointer(ml.NewExecution(bonsaiSmallZone, time.Now().Add(-5*time.Hour))),
	}
	defaultMeasures := []ml.SoilMeasure{
		ptr.FromPointer(ml.NewSoilMeasure(bonsaiBigZone, mediumHumidity)),
		ptr.FromPointer(ml.NewSoilMeasure(bonsaiSmallZone, mediumHumidity)),
	}
	tests := []struct {
		name string
		expectedErr, statusErr,
		executionsErr, soilMeasureErr,
		humidityRefErr, predictErr error
		status     *ml.Status
		executions ml.Executions
		measures   []ml.SoilMeasure
		humRef     *ml.HumidityReference
		prediction []ml.Prediction
	}{
		{
			name:        "and status repository returns error, then it returns same error",
			expectedErr: errTest,
			statusErr:   errTest,
		},
		{
			name:          "and executions repository returns error, then it returns same error",
			expectedErr:   errTest,
			status:        defaultStatus,
			executionsErr: errTest,
		},
		{
			name:           "and soil measure repository returns error, then it returns same error",
			expectedErr:    errTest,
			status:         defaultStatus,
			executions:     defaultExecutions,
			soilMeasureErr: errTest,
		},
		{
			name:           "and humidity reference repository returns error, then it returns same error",
			expectedErr:    errTest,
			status:         defaultStatus,
			executions:     defaultExecutions,
			measures:       defaultMeasures,
			humidityRefErr: errTest,
		},
		{
			name:        "and calculated watering returns error, then it returns same error",
			expectedErr: ml.ErrUnknownZone,
			status:      defaultStatus,
			executions: ml.Executions{
				"wrong": ptr.FromPointer(ml.NewExecution(bonsaiBigZone, time.Now())),
			},
			measures: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure(bonsaiBigZone, lowHumidity)),
				ptr.FromPointer(ml.NewSoilMeasure(bonsaiSmallZone, lowHumidity)),
			},
			humRef: ml.NewHumidityReference(humidity40, humidity100),
		},
		{
			name:        "and getting prediction returns an returns error, then it returns same error",
			expectedErr: errTest,
			predictErr:  errTest,
			status:      defaultStatus,
			executions:  defaultExecutions,
			measures:    defaultMeasures,
			humRef:      ml.NewHumidityReference(humidity40, humidity100),
		},
		{
			name:        "and getting prediction returns an returns error, then it returns same error",
			expectedErr: errTest,
			predictErr:  errTest,
			status:      defaultStatus,
			executions:  defaultExecutions,
			measures:    defaultMeasures,
			humRef:      ml.NewHumidityReference(humidity40, humidity100),
		},
		{
			name:       "and getting prediction returns an returns predictions, then it returns a calculated object",
			status:     defaultStatus,
			executions: defaultExecutions,
			measures:   defaultMeasures,
			humRef:     ml.NewHumidityReference(humidity40, humidity100),
			prediction: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction(uuid.New(), bonsaiBigZone, false, 0, "no needed", 0)),
				ptr.FromPointer(ml.NewPrediction(uuid.New(), bonsaiSmallZone, false, 0, "no needed", 0)),
			},
		},
		{
			name:       "and calculate watering returns an returns calculated, then it returns a calculated object",
			status:     defaultStatus,
			executions: defaultExecutions,
			measures: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure(bonsaiBigZone, lowHumidity)),
				ptr.FromPointer(ml.NewSoilMeasure(bonsaiSmallZone, lowHumidity)),
			},
			humRef: ml.NewHumidityReference(humidity40, humidity100),
		},
	}
	for _, tt := range tests {
		t.Run(`Given a Calculate service,
		when Do method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			predictionRepo := &ml.PredictionRepositoryMock{
				GetFunc: func(_ context.Context) ([]ml.Prediction, error) {
					return tt.prediction, tt.predictErr
				},
			}
			measureRepo := &ml.SoilMeasureRepositoryMock{
				GetFunc: func(_ context.Context) ([]ml.SoilMeasure, error) {
					return tt.measures, tt.soilMeasureErr
				},
			}
			humRefRepo := &ml.HumidityReferenceRepositoryMock{
				GetByZoneFunc: func(_ context.Context, _ string) (*ml.HumidityReference, error) {
					return tt.humRef, tt.humidityRefErr
				},
			}
			execRepo := &ml.ExecutionRepositoryMock{
				GetLastExecutionFunc: func(_ context.Context) (ml.Executions, error) {
					return tt.executions, tt.executionsErr
				},
			}
			statusRepo := &ml.StatusRepositoryMock{
				GetStatusFunc: func(_ context.Context) (*ml.Status, error) {
					return tt.status, tt.statusErr
				},
			}
			svc := ml.NewCalculate(predictionRepo, measureRepo, humRefRepo, execRepo, statusRepo, buildTracer(), dayTimeFunc)
			cp, err := svc.Do(t.Context())
			if err != nil {
				require.Equal(t, tt.expectedErr, err)
				return
			}
			require.NotNil(t, cp)
			require.NotNil(t, cp.Events())
		})
	}
}

func buildTracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}
