package python

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bruli/watersystem-ml/internal/domain/ml"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Prediction struct {
	Time                             string  `json:"time"`
	Zone                             string  `json:"zone"`
	Temperature                      float64 `json:"temperature"`
	WeatherIsRainingLast             int     `json:"weather_is_raining_last"`
	ForecastPrecipitationProbability float64 `json:"forecast_precipitation_probability"`
	DaysSinceLastWatering            float64 `json:"days_since_last_watering"`
	WateringProba                    float64 `json:"watering_proba"`
	ShouldWater                      bool    `json:"should_water"`
	PredictedSeconds                 float64 `json:"predicted_seconds"`
	DecisionReason                   string  `json:"decision_reason"`
}

func parsePredictions(output string) ([]Prediction, error) {
	var predictions []Prediction
	if err := json.Unmarshal([]byte(output), &predictions); err != nil {
		return nil, fmt.Errorf("error parsing predictions: %s", err)
	}
	return predictions, nil
}

type PredictionRepository struct {
	tracer     trace.Tracer
	pythonPath string
	timeout    time.Duration
}

func (p PredictionRepository) Get(ctx context.Context) ([]ml.Prediction, error) {
	ctx, span := p.tracer.Start(ctx, "GetPredictionRepository")
	defer span.End()

	output, err := runCommand(ctx, p.timeout, p.pythonPath, predictFile, "--json")
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("error running prediction: %s", err)
	}
	predictions, err := parsePredictions(output)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("error parsing predictions: %s", err)
	}
	span.SetStatus(codes.Ok, "OK")
	return buildPredictionsDomain(predictions), nil
}

func buildPredictionsDomain(predictions []Prediction) []ml.Prediction {
	pred := make([]ml.Prediction, len(predictions))

	for i, p := range predictions {
		prediction := ml.NewPrediction(p.Zone, p.ShouldWater, p.PredictedSeconds, p.DecisionReason)
		pred[i] = *prediction
	}
	return pred
}

func NewPredictionRepository(tracer trace.Tracer, pythonPath string, timeout time.Duration) *PredictionRepository {
	return &PredictionRepository{tracer: tracer, pythonPath: pythonPath, timeout: timeout}
}
