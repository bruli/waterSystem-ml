package app

import (
	"context"
	"time"

	"github.com/bruli/go-core/cqs"
	"github.com/bruli/go-core/event"
	"github.com/bruli/watersystem-ml/internal/domain/ml"
)

const ValidatePredictionCommandName = "validate_prediction"

type ValidatePredictionCommand struct {
	Limit time.Time
}

func (v ValidatePredictionCommand) Name() string {
	return ValidatePredictionCommandName
}

type ValidatePrediction struct {
	svc *ml.ValidatePrediction
}

func (v ValidatePrediction) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	co, ok := cmd.(ValidatePredictionCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(ValidatePredictionCommandName, cmd.Name())
	}
	pred, err := v.svc.Validate(ctx, co.Limit)
	if err != nil {
		return nil, err
	}
	events := make([]event.Event, 0)
	for _, p := range pred {
		events = append(events, p.Events()...)
	}
	return events, nil
}

func NewValidatePrediction(svc *ml.ValidatePrediction) *ValidatePrediction {
	return &ValidatePrediction{svc: svc}
}
