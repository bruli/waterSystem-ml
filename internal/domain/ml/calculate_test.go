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
	defaultHumidityRef := ml.NewHumidityReference(humidity40, humidity100)

	tests := []struct {
		name string

		expectedErr error

		statusErr        error
		executionsErr    error
		soilMeasureErr   error
		predictionLogErr error
		humidityRefErr   error
		predictErr       error

		status                *ml.Status
		executions            ml.Executions
		measures              []ml.SoilMeasure
		humRef                *ml.HumidityReference
		pendingValidationZone map[string]bool
		prediction            []ml.Prediction
		wantEvents            int
		wantPredict           bool
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
			name:             "and prediction log repository returns error, then it returns same error",
			expectedErr:      errTest,
			status:           defaultStatus,
			executions:       defaultExecutions,
			measures:         defaultMeasures,
			predictionLogErr: errTest,
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
				"wrong": ptr.FromPointer(ml.NewExecution("wrong", time.Now().Add(-5*time.Hour))),
			},
			measures: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure(bonsaiBigZone, lowHumidity)),
				ptr.FromPointer(ml.NewSoilMeasure(bonsaiSmallZone, lowHumidity)),
			},
			humRef: defaultHumidityRef,
		},
		{
			name:        "and prediction repository returns error for medium humidity zones, then it returns same error",
			expectedErr: errTest,
			predictErr:  errTest,
			status:      defaultStatus,
			executions:  defaultExecutions,
			measures:    defaultMeasures,
			humRef:      defaultHumidityRef,
			wantPredict: true,
		},
		{
			name:       "and prediction repository returns predictions for medium humidity zones, then it returns a calculated object",
			status:     defaultStatus,
			executions: defaultExecutions,
			measures:   defaultMeasures,
			humRef:     defaultHumidityRef,
			prediction: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction(uuid.New(), bonsaiBigZone, false, 0, "no needed", 0)),
				ptr.FromPointer(ml.NewPrediction(uuid.New(), bonsaiSmallZone, false, 0, "no needed", 0)),
			},
			wantEvents:  2,
			wantPredict: true,
		},
		{
			name:       "and one zone has pending validation, then only the other medium zone asks prediction",
			status:     defaultStatus,
			executions: defaultExecutions,
			measures:   defaultMeasures,
			humRef:     defaultHumidityRef,
			pendingValidationZone: map[string]bool{
				bonsaiBigZone: true,
			},
			prediction: []ml.Prediction{
				ptr.FromPointer(ml.NewPrediction(uuid.New(), bonsaiSmallZone, false, 0, "no needed", 0)),
			},
			wantEvents:  2,
			wantPredict: true,
		},
		{
			name:       "and all medium zones have pending validation, then it returns without asking predictions",
			status:     defaultStatus,
			executions: defaultExecutions,
			measures:   defaultMeasures,
			humRef:     defaultHumidityRef,
			pendingValidationZone: map[string]bool{
				bonsaiBigZone:   true,
				bonsaiSmallZone: true,
			},
			wantEvents: 2,
		},
		{
			name:       "and calculated watering is already resolved by low humidity, then it returns without asking predictions",
			status:     defaultStatus,
			executions: defaultExecutions,
			measures: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure(bonsaiBigZone, lowHumidity)),
				ptr.FromPointer(ml.NewSoilMeasure(bonsaiSmallZone, lowHumidity)),
			},
			humRef:     defaultHumidityRef,
			wantEvents: 2,
		},
		{
			name:       "and system is blocked by raining, then it returns without asking predictions",
			status:     ml.NewStatus(true, true),
			executions: defaultExecutions,
			measures:   defaultMeasures,
			humRef:     defaultHumidityRef,
			wantEvents: 1,
		},
	}

	for _, tt := range tests {
		t.Run(`Given a Calculate service, when Do method is called `+tt.name, func(t *testing.T) {
			t.Parallel()

			predictionCalled := false
			predictionRepo := &ml.PredictionRepositoryMock{
				GetFunc: func(_ context.Context) ([]ml.Prediction, error) {
					predictionCalled = true
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
			predictionLogRepo := &ml.PredictionLogRepositoryMock{
				GetPendingValidationZonesFunc: func(_ context.Context) (map[string]bool, error) {
					return tt.pendingValidationZone, tt.predictionLogErr
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

			svc := ml.NewCalculate(
				predictionRepo,
				measureRepo,
				humRefRepo,
				execRepo,
				predictionLogRepo,
				statusRepo,
				buildTracer(),
				dayTimeFunc,
			)
			cw, err := svc.Do(t.Context())

			if tt.expectedErr != nil {
				require.ErrorIs(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cw)
			require.Equal(t, tt.wantPredict, predictionCalled)
			require.Len(t, cw.Events(), tt.wantEvents)
		})
	}
}

func buildTracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}
