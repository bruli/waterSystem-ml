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
	Time                             string  `json:"_time"`
	Zone                             string  `json:"zone"`
	Temperature                      float64 `json:"temperature"`
	WeatherIsRainingLast             int     `json:"weather_is_raining_last"`
	ForecastTemperature              float64 `json:"forecast_temperature"`
	ForecastRelativeHumidity         float64 `json:"forecast_relative_humidity"`
	ForecastPrecipitationProbability float64 `json:"forecast_precipitation_probability"`
	ForecastCloudCover               float64 `json:"forecast_cloud_cover"`
	ForecastShortwaveRadiation       float64 `json:"forecast_shortwave_radiation"`
	ForecastDryingFactor             float64 `json:"forecast_drying_factor"`
	DaysSinceLastWatering            float64 `json:"days_since_last_watering"`
	SoilMoisture                     float64 `json:"soil_moisture"`
	SoilTemperature                  float64 `json:"soil_temperature"`
	SoilMoistureDiff                 float64 `json:"soil_moisture_diff"`
	WateringProba                    float64 `json:"watering_proba"`
	RawPredictedSeconds              float64 `json:"raw_predicted_seconds"`
	ShouldWater                      bool    `json:"should_water"`
	PredictedSeconds                 float64 `json:"predicted_seconds"`
	DecisionReason                   string  `json:"decision_reason"`
}

func parsePredictions(output string) ([]Prediction, error) {
	var predictions []Prediction
	if err := json.Unmarshal([]byte(output), &predictions); err != nil {
		return nil, fmt.Errorf("error parsing predictions JSON: %w; output=%q", err, output)
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

	for i := range predictions {
		p := &predictions[i]
		prediction := ml.NewPrediction(p.Zone, p.ShouldWater, p.PredictedSeconds, p.DecisionReason)
		pred[i] = *prediction
	}
	return pred
}

func NewPredictionRepository(tracer trace.Tracer, pythonPath string, timeout time.Duration) *PredictionRepository {
	return &PredictionRepository{tracer: tracer, pythonPath: pythonPath, timeout: timeout}
}
