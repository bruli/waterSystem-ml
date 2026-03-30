package ml

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
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

func RunPrediction(ctx context.Context) ([]Prediction, error) {
	output, err := runCommand(ctx, 30*time.Minute, pythonEnv, predictFile, "--json")
	if err != nil {
		return nil, fmt.Errorf("error running prediction: %s", err)
	}
	predictions, err := parsePredictions(output)
	if err != nil {
		return nil, fmt.Errorf("error parsing predictions: %s", err)
	}
	return predictions, nil
}

func parsePredictions(output string) ([]Prediction, error) {
	var predictions []Prediction
	if err := json.Unmarshal([]byte(output), &predictions); err != nil {
		return nil, fmt.Errorf("error parsing predictions: %s", err)
	}
	return predictions, nil
}
