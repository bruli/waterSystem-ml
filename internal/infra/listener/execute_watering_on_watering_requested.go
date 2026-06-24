package listener

import (
	"context"
	"fmt"

	"github.com/bruli/go-core/cqs"
	"github.com/bruli/go-core/event"
	"github.com/bruli/watersystem-ml/internal/app"
	"github.com/bruli/watersystem-ml/internal/domain/ml"
)

type ExecuteWateringOnWateringRequested struct {
	ch cqs.CommandHandler
}

func (e ExecuteWateringOnWateringRequested) Listen(ctx context.Context, ev event.Event) error {
	swr, ok := ev.(*ml.WateringRequestedEvent)
	if !ok {
		return fmt.Errorf("invalid event type: %T", ev)
	}
	if _, err := e.ch.Handle(ctx, app.ExecuteWateringCommand{
		Zone:    swr.Zone,
		Seconds: int(swr.Seconds),
	}); err != nil {
		return fmt.Errorf("error executing watering: %w", err)
	}
	return nil
}

func NewExecuteWateringOnWateringRequested(ch cqs.CommandHandler) *ExecuteWateringOnWateringRequested {
	return &ExecuteWateringOnWateringRequested{ch: ch}
}
