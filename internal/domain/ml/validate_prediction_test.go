package ml_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bruli/go-core/ptr"
	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"github.com/bruli/watersystem-ml/internal/fixtures"
	"github.com/stretchr/testify/require"
)

func TestValidatePrediction_Validate(t *testing.T) {
	errTest := errors.New("error")
	tests := []struct {
		name string
		expectedErr, soilMeasureErr,
		predictionLogErr, saveErr error
		moisture      []ml.SoilMeasure
		predictionLog *ml.PredictionLog
	}{
		{
			name:           "and soil measure returns an error, then it returns same error",
			soilMeasureErr: errTest,
			expectedErr:    errTest,
		},
		{
			name:             "and prediction log returns an error, then it returns same error",
			predictionLogErr: errTest,
			expectedErr:      errTest,
			moisture: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure(bonsaiBigZone, highHumidity)),
				ptr.FromPointer(ml.NewSoilMeasure(bonsaiSmallZone, highHumidity)),
			},
		},
		{
			name: "and prediction log save returns an error, then it returns same error",
			moisture: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure(bonsaiBigZone, highHumidity)),
				ptr.FromPointer(ml.NewSoilMeasure(bonsaiSmallZone, highHumidity)),
			},
			predictionLog: new(fixtures.PredictionLogBuilder{}.Build(t)),
			saveErr:       errTest,
			expectedErr:   errTest,
		},
		{
			name: "and prediction log save returns nil, then it returns nil error",
			moisture: []ml.SoilMeasure{
				ptr.FromPointer(ml.NewSoilMeasure(bonsaiBigZone, highHumidity)),
				ptr.FromPointer(ml.NewSoilMeasure(bonsaiSmallZone, highHumidity)),
			},
			predictionLog: new(fixtures.PredictionLogBuilder{}.Build(t)),
		},
	}
	for _, tt := range tests {
		t.Run(`Given a ValidatePrediction service,
		when Validate method is called `+tt.name, func(t *testing.T) {
			t.Parallel()
			soilMeasureRepo := &ml.SoilMeasureRepositoryMock{
				GetFunc: func(_ context.Context) ([]ml.SoilMeasure, error) {
					return tt.moisture, tt.soilMeasureErr
				},
			}
			predictionLogRepo := &ml.PredictionLogRepositoryMock{
				GetPendingByZoneFunc: func(_ context.Context, _ string, _ time.Time) (*ml.PredictionLog, error) {
					return tt.predictionLog, tt.predictionLogErr
				},
				SaveFunc: func(_ context.Context, _ *ml.PredictionLog) error {
					return tt.saveErr
				},
			}
			svc := ml.NewValidatePrediction(soilMeasureRepo, predictionLogRepo, buildTracer())
			got, err := svc.Validate(t.Context(), time.Now())
			if err != nil {
				require.Equal(t, tt.expectedErr, err)
				return
			}
			require.NotEmpty(t, got)
			require.Len(t, got, len(tt.moisture))
		})
	}
}
